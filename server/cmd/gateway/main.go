package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"codex-online/server/internal/container"
	"codex-online/server/internal/message"
	"codex-online/server/internal/network"
	"codex-online/server/internal/persistence"
	"codex-online/server/internal/telemetry"
	"codex-online/server/internal/validation"
	"codex-online/server/internal/zone"

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

	ctr := container.New(repo)
	gw := newGateway(ctr)

	// Create persistent hub zone at startup.
	gw.getOrCreateZone(zone.ZoneHub, zone.ZoneTypeHub)

	// Start periodic position flush (every 30s).
	flushCtx, flushCancel := context.WithCancel(ctx)
	defer flushCancel()
	go gw.periodicFlush(flushCtx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		handleConnection(gw, w, req)
	})

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
