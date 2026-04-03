FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

COPY . .

RUN CGO_ENABLED=0 go build -o /bin/gateway ./cmd/gateway/ && \
    CGO_ENABLED=0 go build -o /bin/zone    ./cmd/zone/    && \
    CGO_ENABLED=0 go build -o /bin/chat    ./cmd/chat/

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/gateway /bin/gateway
COPY --from=builder /bin/zone    /bin/zone
COPY --from=builder /bin/chat    /bin/chat

ENTRYPOINT ["/bin/gateway"]
