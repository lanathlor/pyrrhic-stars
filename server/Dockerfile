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

# Ship the game data and point the server at every directory it loads.
COPY shared /shared
ENV CODEX_LEVELS_DIR=/shared/levels \
    CODEX_MOBS_DIR=/shared/mobs \
    CODEX_ENCOUNTERS_DIR=/shared/encounters \
    CODEX_ITEMS_DIR=/shared/items

# The ability catalog is loaded from a CWD-relative "data/abilities" path, so the
# server data lands at /data and the container runs from /.
COPY server/data /data
WORKDIR /

ENTRYPOINT ["/bin/gateway"]
