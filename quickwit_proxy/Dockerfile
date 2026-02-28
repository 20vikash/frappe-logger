# ---------- Stage 1: Build Proxy ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY proxy/go.mod proxy/go.sum ./
RUN go mod download

COPY proxy/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o proxy

# ---------- Stage 2: Final Image ----------
FROM docker.io/quickwit/quickwit:qw-fulmicoton-lambda

RUN apt-get update && \
    apt-get install -y supervisor && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /quickwit/config

COPY --from=builder /app/proxy /quickwit/config/proxy
COPY supervisord.conf /etc/supervisord.conf

EXPOSE 8080

ENTRYPOINT ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
