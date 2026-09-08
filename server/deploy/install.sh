#!/bin/sh
# Push the server, broker config and service onto the GL-MT3000 and start them.
# Run from the repo root on the dev machine:
#   sh server/deploy/install.sh [root@192.168.8.1]
# Requires the router on its WAN port with internet for opkg, and the arm64
# binary built by server/deploy/build.sh.
set -eu

HOST="${1:-root@192.168.8.1}"
HERE="$(cd "$(dirname "$0")" && pwd)"
BIN="$HERE/../dist/showcase-server-linux-arm64"

[ -f "$BIN" ] || { echo "build first: sh server/deploy/build.sh" >&2; exit 1; }

echo "== packages"
ssh "$HOST" 'opkg update >/dev/null && opkg install mosquitto-nossl mosquitto-client-nossl 2>&1 | tail -2'

echo "== files"
scp "$BIN" "$HOST:/usr/bin/showcase-server"
scp "$HERE/mosquitto.conf" "$HOST:/etc/mosquitto/mosquitto.conf"
scp "$HERE/showcase-server.init" "$HOST:/etc/init.d/showcase-server"
ssh "$HOST" 'chmod +x /usr/bin/showcase-server /etc/init.d/showcase-server; mkdir -p /etc/showcase /var/lib/mosquitto'

echo "== config"
if ssh "$HOST" 'test -f /etc/showcase/server.env'; then
  echo "keeping existing /etc/showcase/server.env"
else
  ssh "$HOST" 'cat > /etc/showcase/server.env' <<EOF
LISTEN=:8080
MQTT_URL=tcp://127.0.0.1:1883
EVENT_DATETIME=2026-09-17T14:00:00+10:00
EVENT_NAME=[red]Westpac[/] + [#00A4EF]Microsoft[/] Hackathon
ORGANISER_SECRET=change-me
FAKE_STICKS=0
# BRIDGE_PORT=/dev/ttyUSB0
EOF
  echo "wrote /etc/showcase/server.env - set ORGANISER_SECRET"
fi

echo "== services"
ssh "$HOST" '/etc/init.d/mosquitto enable; /etc/init.d/mosquitto restart; /etc/init.d/showcase-server enable; /etc/init.d/showcase-server restart; sleep 2; logread -e showcase | tail -5'

echo "== done: dashboard http://${HOST#*@}:8080/  MCP http://${HOST#*@}:8080/mcp/<code>"
