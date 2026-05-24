package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"codex-online/server/internal/ability"
	"codex-online/server/internal/abilitycatalog"
	"codex-online/server/internal/combatlog"
	combatapi "codex-online/server/internal/combatlog/api"
	chrepo "codex-online/server/internal/combatlog/clickhouse"
	"codex-online/server/internal/container"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/item"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/session"
	"codex-online/server/internal/telemetry"
	"codex-online/server/internal/validation"
	"codex-online/server/internal/zone"

	clickhousedriver "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/coder/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()

	devMode := os.Getenv("CODEX_DEV") == "1"

	shutdown, err := telemetry.Init(ctx)
	if err != nil {
		slog.Error("telemetry init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.Error("telemetry shutdown", "error", err)
		}
	}()

	repo, err := initDatabase(devMode)
	if err != nil {
		os.Exit(1)
	}

	if err := loadGameData(); err != nil {
		os.Exit(1)
	}

	combatSink, logQueryRepo, err := initCombatLogSink(ctx, devMode)
	if err != nil {
		os.Exit(1)
	}
	defer func() {
		if err := combatSink.Close(); err != nil {
			slog.Error("combat log close", "error", err)
		}
	}()

	ctr := container.New(repo)
	ctr.CombatLogSink = combatSink
	gw := newGateway(ctr)
	if devMode {
		gw.devMode = true
		slog.Info("dev mode enabled — debug opcodes active, standalone (no external deps)")
	}

	configureGateway(gw)

	// Start periodic position flush (every 30s).
	flushCtx, flushCancel := context.WithCancel(ctx)
	defer flushCancel()
	go gw.periodicFlush(flushCtx)

	mux := setupHTTPServer(gw, logQueryRepo)
	if err := runHTTPServer(mux); err != nil {
		slog.Error("listen failed", "error", err)
		os.Exit(1)
	}
}

// runHTTPServer creates the http.Server, registers a graceful shutdown
// goroutine on SIGINT, and blocks until the server closes. It returns a
// non-nil error only on unexpected listener failures.
func runHTTPServer(mux *http.ServeMux) error {
	addr := ":7777"
	if envAddr := os.Getenv("GATEWAY_ADDR"); envAddr != "" {
		addr = envAddr
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		slog.Info("shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	slog.Info("gateway listening", "addr", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// configureGateway loads the ability catalog, creates the ability engine,
// creates the persistent hub zone, and any other one-time gateway setup.
func configureGateway(gw *gateway) {
	// Load ability catalog for loadout validation.
	cat, err := abilitycatalog.Load("data/abilities/arcanotechnicien.yaml")
	if err != nil {
		slog.Warn("ability catalog not loaded", "error", err)
		// Non-fatal: loadout validation will be skipped if catalog is nil.
	} else {
		slog.Info("ability catalog loaded", "abilities", len(cat.Abilities))
	}
	gw.catalog = cat

	// Create ability engine for stat lookups (catalog enrichment).
	gw.abilityEng = ability.NewEngine(nil)

	// Create persistent hub zone at startup.
	gw.getOrCreateZone(zone.ZoneHub, zone.ZoneTypeOpenWorld, 0)
}

// initDatabase selects the database driver, applies dev-mode overrides, and
// returns an initialized GormRepo. Errors are logged before being returned.
func initDatabase(devMode bool) (*persistence.GormRepo, error) { //nolint:revive // init helper, flag coupling is fine
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" || devMode {
		dbDriver = "sqlite"
	}
	pgDSN := os.Getenv("POSTGRES_DSN")
	if devMode {
		pgDSN = ""
	}
	repo, err := persistence.NewGormRepo(dbDriver, pgDSN)
	if err != nil {
		slog.Error("database init failed", "driver", dbDriver, "error", err)
		return nil, err
	}
	slog.Info("database initialized", "driver", dbDriver)
	return repo, nil
}

// loadGameData loads enemy mobs, encounters, and items from YAML data files.
// Errors are logged before being returned.
func loadGameData() error {
	if err := enemyai.LoadMobs(enemyai.MobsDir()); err != nil {
		slog.Error("load mobs failed", "error", err)
		return err
	}
	if err := enemyai.LoadEncounters(enemyai.EncountersDir()); err != nil {
		slog.Error("load encounters failed", "error", err)
		return err
	}
	if err := item.LoadItems(item.ItemsDir()); err != nil {
		slog.Error("load items failed", "error", err)
		return err
	}
	return nil
}

// initCombatLogSink sets up the combat log sink. Dev mode uses an in-memory
// sink; production connects to ClickHouse when CLICKHOUSE_ADDR is set. Returns
// the sink (never nil), an optional ReadRepository, and any fatal error.
func initCombatLogSink(ctx context.Context, devMode bool) (combatlog.EventSink, combatlog.ReadRepository, error) { //nolint:revive // init helper, flag coupling is fine
	if devMode {
		slog.Info("dev mode: in-memory combat log (no ClickHouse required)")
		return combatlog.NewInMemorySink(), nil, nil
	}

	chAddr := os.Getenv("CLICKHOUSE_ADDR")
	if chAddr == "" {
		return combatlog.NullSink{}, nil, nil
	}

	chDB := os.Getenv("CLICKHOUSE_DB")
	if chDB == "" {
		chDB = "codex"
	}
	chUser := os.Getenv("CLICKHOUSE_USER")
	if chUser == "" {
		chUser = "default"
	}
	chPass := os.Getenv("CLICKHOUSE_PASSWORD")

	chConn, err := clickhousedriver.Open(&clickhousedriver.Options{
		Addr: []string{chAddr},
		Auth: clickhousedriver.Auth{
			Database: chDB,
			Username: chUser,
			Password: chPass,
		},
	})
	if err != nil {
		slog.Error("clickhouse connect failed, combat logging disabled", "addr", chAddr, "error", err)
		return combatlog.NullSink{}, nil, nil
	}
	if err = chrepo.EnsureSchema(ctx, chConn); err != nil {
		slog.Error("clickhouse schema init failed, combat logging disabled", "error", err)
		_ = chConn.Close()
		return combatlog.NullSink{}, nil, nil
	}
	chRepo := chrepo.NewRepo(chConn)
	slog.Info("combat logging enabled", "clickhouse_addr", chAddr, "db", chDB)
	return combatlog.NewLogger(chRepo), chRepo, nil
}

// setupHTTPServer builds the HTTP mux with the WebSocket endpoint and,
// optionally, the combat log REST API when a query repository is available.
func setupHTTPServer(gw *gateway, logQueryRepo combatlog.ReadRepository) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		handleConnection(gw, w, req)
	})

	if logQueryRepo != nil {
		apiMux := http.NewServeMux()
		logAPI := combatapi.NewHandler(logQueryRepo)
		logAPI.Register(apiMux)
		mux.Handle("/api/", combatapi.CORS(apiMux))
		slog.Info("combat log API enabled")
	}

	return mux
}

func handleConnection(gw *gateway, w http.ResponseWriter, req *http.Request) {
	userUUID, username, allChars, ok := authenticateRequest(gw, w, req)
	if !ok {
		return
	}

	conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow any origin for dev.
	})
	if err != nil {
		slog.Error("websocket accept", "error", err)
		return
	}

	_, connSpan := telemetry.Tracer().Start(req.Context(), "connection",
		trace.WithAttributes(attribute.String("remote_addr", req.RemoteAddr)),
	)

	client := network.NewClient(conn)
	sess := gw.sessions.Register(client)
	sess.UserUUID = userUUID
	sess.Username = username
	sess.Class = "gunner"
	if len(allChars) > 0 {
		sess.Class = allChars[0].ClassName
	}

	// Dev auto-join: skip character select, auto-create character, join zone directly.
	if gw.devMode && req.URL.Query().Get("dev_auto") == "1" {
		devAutoJoin(gw, sess, client, req, allChars)
	} else {
		// Normal flow: send character list for the selection screen.
		client.Send(encodeCharacterListMsg(username, allChars))
	}
	defer func() {
		// Save character position before cleanup (hub only).
		if sess.UserUUID != "" && sess.Class != "" && sess.ZoneID == zone.ZoneHub {
			gw.savePlayerPosition(sess)
		}
		// Remove from group.
		if g, disbanded := gw.groups.LeaveGroup(sess.ID); !disbanded && g != nil {
			gw.broadcastGroupState(g)
		}
		gw.leaveZone(sess)
		gw.sessions.Remove(client)
		client.Close()
		_ = conn.CloseNow()
		connSpan.End()
	}()

	slog.Info("new connection", "remote_addr", req.RemoteAddr, "player_id", sess.ID)
	runMessageLoop(gw, sess, client)
}

// runMessageLoop reads messages from client and dispatches them until the
// connection is closed.
func runMessageLoop(gw *gateway, sess *session.Session, client *network.Client) {
	for {
		data, readErr := client.ReadMessage()
		if readErr != nil {
			slog.Info("connection closed", "player_id", sess.ID, "peer_id", sess.PeerID, "error", readErr)
			return
		}

		opcode, _, payload, decErr := message.Decode(data)
		if decErr != nil {
			slog.Error("decode", "error", decErr)
			continue
		}

		if message.IsServerHandled(opcode) {
			handleServerMessage(gw, sess, opcode, payload)
			continue
		}

		if message.IsClientInput(opcode) || (gw.devMode && message.IsDebugInput(opcode)) {
			routeClientInput(gw, sess, opcode, payload)
			continue
		}

		relayLegacyMessage(gw, sess, opcode, payload)
	}
}

// relayLegacyMessage relays a message via the zone broadcast for backward
// compatibility during protocol migration.
func relayLegacyMessage(gw *gateway, sess *session.Session, opcode uint16, payload []byte) {
	if sess.ZoneID == "" {
		return
	}
	zi := gw.getZone(sess.ZoneID)
	if zi == nil {
		return
	}
	outMsg := message.Encode(opcode, sess.PeerID, payload)
	if message.BroadcastExcludeSender(opcode) {
		zi.zone.Broadcast(outMsg, sess.PeerID)
	} else {
		zi.zone.Broadcast(outMsg, 0)
	}
}

// authenticateRequest validates the WebSocket handshake query parameters,
// upserts the user in the DB, and returns the authoritative username plus the
// character list. Returns ok=false (and has already written an HTTP error) if
// auth fails.
func authenticateRequest(gw *gateway, w http.ResponseWriter, req *http.Request) (userUUID, username string, allChars []*persistence.Character, ok bool) {
	userUUID = req.URL.Query().Get("uuid")
	username = req.URL.Query().Get("username")

	if !validation.IsValidUUID(userUUID) {
		http.Error(w, "invalid or missing uuid", http.StatusUnauthorized)
		return "", "", nil, false
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = "Player"
	}
	if len(username) > 20 {
		username = username[:20]
	}

	// Upsert user in DB (only sets username on first creation).
	if err := gw.container.Repo.UpsertUser(userUUID, username); err != nil {
		slog.Error("upsert user", "uuid", userUUID, "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return "", "", nil, false
	}

	// Use the stored username as authoritative (query param only used for creation).
	u, err := gw.container.Repo.GetUser(userUUID)
	if err != nil || u == nil {
		slog.Error("get user after upsert", "uuid", userUUID, "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return "", "", nil, false
	}
	username = u.Username

	// Load all characters for the selection screen.
	allChars, _ = gw.container.Repo.GetCharacters(userUUID)
	return userUUID, username, allChars, true
}

// devAutoJoin handles the dev auto-join flow: finds or creates a character of
// the requested class, applies starter gear, and immediately joins the zone.
func devAutoJoin(gw *gateway, sess *session.Session, client *network.Client, req *http.Request, allChars []*persistence.Character) {
	devClass := req.URL.Query().Get("dev_class")
	if devClass == "" {
		devClass = "gunner"
	}
	devZone := req.URL.Query().Get("dev_zone")
	sess.Class = devClass

	// Find existing character of this class, or create one.
	var devChar *persistence.Character
	for _, c := range allChars {
		if c.ClassName == devClass {
			devChar = c
			break
		}
	}
	if devChar == nil {
		devChar, _ = gw.characters.Create(sess.UserUUID, devClass, "Dev")
		if devChar != nil {
			_ = gw.inventory.SpawnStarterGear(devChar.ID)
		}
	}
	if devChar != nil {
		sess.CharID = devChar.ID
		sess.CharName = devChar.Name
		sess.Spec = devChar.SpecID
		client.Send(encodeCharacterStateMsg(devChar))
	}

	gw.devJoinZone(sess, devZone)
	slog.Info("dev auto-join", "class", devClass, "zone", devZone, "peer_id", sess.PeerID)
}

// routeClientInput tracks session-level class/spec state for interact inputs
// and queues the input into the player's current zone.
func routeClientInput(gw *gateway, sess *session.Session, opcode uint16, payload []byte) {
	// Track class/spec selection in gateway session.
	if opcode == message.OpInteractInput && len(payload) >= 3 {
		switch payload[0] {
		case message.InteractClassSelect:
			nameLen := int(payload[1])
			if len(payload) >= 2+nameLen {
				sess.Class = string(payload[2 : 2+nameLen])
			}
			sess.Spec = "" // reset spec when class changes
		case message.InteractSpecSelect:
			nameLen := int(payload[1])
			if len(payload) >= 2+nameLen {
				sess.Spec = string(payload[2 : 2+nameLen])
				if sess.CharID != 0 {
					if err := gw.container.Repo.UpdateCharacterSpec(sess.CharID, sess.Spec); err != nil {
						slog.Error("persist spec", "char_id", sess.CharID, "spec", sess.Spec, "error", err)
					}
				}
			}
		}
	}
	if sess.ZoneID != "" {
		zi := gw.getZone(sess.ZoneID)
		if zi != nil {
			zi.zone.QueueInput(sess.PeerID, opcode, payload)
		}
	}
}
