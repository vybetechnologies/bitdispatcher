#!/bin/sh
# Start Caddy in the background and bitdispatcher in the foreground.
# If bitdispatcher exits, the container exits and Fly restarts the machine.
# First start prints the one-time admin password to stderr — capture it
# from `flyctl logs` and change it immediately in Admin > Users.
set -e
mkdir -p /var/lib/bitdispatcher
caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &
exec /opt/bitdispatcher/bitdispatcher -data /var/lib/bitdispatcher -addr 127.0.0.1:8443
