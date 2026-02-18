# ---------- Stage 1: Build Proxy ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY proxy/go.mod proxy/go.sum ./
RUN go mod download

COPY proxy/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o proxy

# ---------- Stage 2: Final Image ----------
FROM docker.io/quickwit/quickwit:qw-fulmicoton-lambda

WORKDIR /quickwit/config

# Copy proxy binary
COPY --from=builder /app/proxy /quickwit/config/proxy

# Copy start script
COPY start.sh /quickwit/config/start.sh
RUN chmod +x /quickwit/config/start.sh

EXPOSE 8080

ENTRYPOINT ["/quickwit/config/start.sh"]
