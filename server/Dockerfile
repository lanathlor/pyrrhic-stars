# Build context is the repo ROOT (see .github/workflows + .dockerignore) so the
# image can include the shared game data (shared/levels, shared/mobs) that the
# server loads at startup.
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY server/go.mod server/go.sum* ./
RUN go mod download 2>/dev/null || true

COPY server/ ./

RUN CGO_ENABLED=0 go build -o /bin/gateway ./cmd/gateway/ && \
    CGO_ENABLED=0 go build -o /bin/zone    ./cmd/zone/    && \
    CGO_ENABLED=0 go build -o /bin/chat    ./cmd/chat/

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/gateway /bin/gateway
COPY --from=builder /bin/zone    /bin/zone
COPY --from=builder /bin/chat    /bin/chat

# Ship the level/mob definitions and point the server at them.
COPY shared /shared
ENV CODEX_LEVELS_DIR=/shared/levels \
    CODEX_MOBS_DIR=/shared/mobs

ENTRYPOINT ["/bin/gateway"]
