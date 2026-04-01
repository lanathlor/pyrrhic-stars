package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"codex-online/server/internal/relay"
	"codex-online/server/internal/telemetry"

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

	r := relay.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // Allow any origin for dev.
		})
		if err != nil {
			slog.Error("websocket accept", "error", err)
			return
		}

		connCtx, connSpan := telemetry.Tracer().Start(req.Context(), "connection",
			trace.WithAttributes(attribute.String("remote_addr", req.RemoteAddr)),
		)

		client := relay.NewClient(conn)
		defer func() {
			r.RemoveClient(connCtx, client)
			client.Close()
			conn.CloseNow()
			connSpan.End()
		}()

		slog.Info("new connection", "remote_addr", req.RemoteAddr)

		for {
			data, err := client.ReadMessage()
			if err != nil {
				slog.Info("connection closed", "peer_id", client.PeerID, "error", err)
				return
			}
			if err := r.HandleMessage(connCtx, client, data); err != nil {
				slog.Error("handle message", "peer_id", client.PeerID, "error", err)
			}
		}
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
		srv.Shutdown(shutCtx)
	}()

	slog.Info("gateway listening", "addr", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("listen failed", "error", err)
		os.Exit(1)
	}
}
