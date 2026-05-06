package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"codex-online/server/internal/combatlog"
	combatapi "codex-online/server/internal/combatlog/api"
	chrepo "codex-online/server/internal/combatlog/clickhouse"
	"codex-online/server/internal/container"
	"codex-online/server/internal/enemyai"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/persistence"
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

	// Initialize persistence.
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "sqlite"
	}
	pgDSN := os.Getenv("POSTGRES_DSN")
	repo, err := persistence.NewGormRepo(dbDriver, pgDSN)
	if err != nil {
		slog.Error("database init failed", "driver", dbDriver, "error", err)
		os.Exit(1)
	}
	slog.Info("database initialized", "driver", dbDriver)

	// Load data-driven mob definitions from YAML.
	if err := enemyai.LoadMobs(enemyai.MobsDir()); err != nil {
		slog.Error("load mobs failed", "error", err)
		os.Exit(1)
	}
	if err := enemyai.LoadEncounters(enemyai.EncountersDir()); err != nil {
		slog.Error("load encounters failed", "error", err)
		os.Exit(1)
	}

	// Initialize combat log sink (ClickHouse or NullSink).
	var combatSink combatlog.EventSink = combatlog.NullSink{}
	var logQueryRepo combatlog.ReadRepository
	if chAddr := os.Getenv("CLICKHOUSE_ADDR"); chAddr != "" {
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
		} else if err = chrepo.EnsureSchema(ctx, chConn); err != nil {
			slog.Error("clickhouse schema init failed, combat logging disabled", "error", err)
			_ = chConn.Close()
		} else {
			chRepo := chrepo.NewRepo(chConn)
			combatSink = combatlog.NewLogger(chRepo)
			logQueryRepo = chRepo
			slog.Info("combat logging enabled", "clickhouse_addr", chAddr, "db", chDB)
		}
	}
	defer func() {
		if err := combatSink.Close(); err != nil {
			slog.Error("combat log close", "error", err)
		}
	}()

	ctr := container.New(repo)
	ctr.CombatLogSink = combatSink
	gw := newGateway(ctr)

	// Create persistent hub zone at startup.
	gw.getOrCreateZone(zone.ZoneHub, zone.ZoneTypeOpenWorld, 0)

	// Start periodic position flush (every 30s).
	flushCtx, flushCancel := context.WithCancel(ctx)
	defer flushCancel()
	go gw.periodicFlush(flushCtx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		handleConnection(gw, w, req)
	})

	// Combat log REST API (only if ClickHouse is available).
	if logQueryRepo != nil {
		apiMux := http.NewServeMux()
		logAPI := combatapi.NewHandler(logQueryRepo)
		logAPI.Register(apiMux)
		mux.Handle("/api/", combatapi.CORS(apiMux))
		slog.Info("combat log API enabled")
	}

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
		slog.Error("listen failed", "error", err)
		os.Exit(1)
	}
}

func handleConnection(gw *gateway, w http.ResponseWriter, req *http.Request) {
	// Authenticate at HTTP handshake via query params.
	userUUID := req.URL.Query().Get("uuid")
	username := req.URL.Query().Get("username")

	if !validation.IsValidUUID(userUUID) {
		http.Error(w, "invalid or missing uuid", http.StatusUnauthorized)
		return
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
		return
	}

	// Use the stored username as authoritative (query param only used for creation).
	u, err := gw.container.Repo.GetUser(userUUID)
	if err != nil || u == nil {
		slog.Error("get user after upsert", "uuid", userUUID, "error", err)
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}
	username = u.Username

	// Load all characters for the selection screen.
	allChars, _ := gw.container.Repo.GetCharacters(userUUID)

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

	// Send character list for the selection screen.
	client.Send(encodeCharacterListMsg(username, allChars))
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

		if message.IsClientInput(opcode) {
			// Track class selection in gateway session.
			if opcode == message.OpInteractInput && len(payload) >= 3 && payload[0] == message.InteractClassSelect {
				nameLen := int(payload[1])
				if len(payload) >= 2+nameLen {
					sess.Class = string(payload[2 : 2+nameLen])
				}
			}
			if sess.ZoneID != "" {
				zi := gw.getZone(sess.ZoneID)
				if zi != nil {
					zi.zone.QueueInput(sess.PeerID, opcode, payload)
				}
			}
			continue
		}

		// Legacy relay messages (for backward compat during migration).
		if sess.ZoneID != "" {
			zi := gw.getZone(sess.ZoneID)
			if zi != nil {
				outMsg := message.Encode(opcode, sess.PeerID, payload)
				if message.BroadcastExcludeSender(opcode) {
					zi.zone.Broadcast(outMsg, sess.PeerID)
				} else {
					zi.zone.Broadcast(outMsg, 0)
				}
			}
		}
	}
}
