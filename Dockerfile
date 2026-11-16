# bitdispatcher on Fly.io — Lakes Area Security (Fargo, ND)
#
# Upstream: https://github.com/BioDream-LLC/bitdispatcher (Apache 2.0, v0.1.0)
# The web UI is committed into internal/web/dist, so the build needs Go only
# (no Node/pnpm) — see the upstream Makefile comment on the `build` target.
#
# Runtime shape mirrors upstream's own demo deployment:
#   bitdispatcher on 127.0.0.1:8443 (HTTPS, self-signed cert in the data dir)
#   Caddy in front, terminating the Fly-proxied HTTP on :8080
#   Data (ledger, users, recorded map) on a Fly persistent volume
# Fly terminates public TLS at the edge; Caddy proxies to the app over HTTPS
# with certificate verification skipped (loopback, same machine).

# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=v0.1.0-las17" \
      -o /out/bitdispatcher ./cmd/bitdispatcher

FROM caddy:2 AS caddy

FROM debian:12-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates \
 && rm -rf /var/lib/apt/lists/*
COPY --from=caddy /usr/bin/caddy /usr/local/bin/caddy
COPY --from=build /out/bitdispatcher /opt/bitdispatcher/bitdispatcher
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY deploy/start.sh /opt/bitdispatcher/start.sh
RUN chmod +x /opt/bitdispatcher/start.sh \
 && mkdir -p /var/lib/bitdispatcher
EXPOSE 8080
ENTRYPOINT ["/opt/bitdispatcher/start.sh"]
