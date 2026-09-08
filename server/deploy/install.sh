#!/bin/sh
# Push the server onto the GL-MT3000 and start it. The binary is broker, MCP
# server and dashboard in one, so nothing is installed from opkg.
# Run from the repo root on the dev machine:
#   sh server/deploy/install.sh [root@192.168.8.1]
# Requires the arm64 binary built by server/deploy/build.sh and SSH access.
set -eu

HOST="${1:-root@192.168.8.1}"
SSH="${SSH:-ssh}"   # e.g. SSH="ssh -i ~/.ssh/unraid" ; SCP follows the same key
SCP="${SCP:-scp -O}"   # -O: the router has no sftp-server
HERE="$(cd "$(dirname "$0")" && pwd)"
BIN="$HERE/../dist/showcase-server-linux-arm64"

[ -f "$BIN" ] || { echo "build first: sh server/deploy/build.sh" >&2; exit 1; }

echo "== files"
$SSH "$HOST" '/etc/init.d/showcase-server stop 2>/dev/null || true; mkdir -p /etc/showcase'
$SCP "$BIN" "$HOST:/usr/bin/showcase-server"
$SCP "$HERE/showcase-server.init" "$HOST:/etc/init.d/showcase-server"
$SSH "$HOST" 'chmod +x /usr/bin/showcase-server /etc/init.d/showcase-server'

echo "== config"
if $SSH "$HOST" 'test -f /etc/showcase/server.env'; then
  echo "keeping existing /etc/showcase/server.env"
else
  $SSH "$HOST" 'cat > /etc/showcase/server.env' <<EOF
LISTEN=:8080
EMBEDDED_BROKER=:1883
EVENT_DATETIME=2026-09-17T14:00:00+10:00
EVENT_NAME=[red]Westpac[/] + [#00A4EF]Microsoft[/] Hackathon
ORGANISER_SECRET=change-me
FAKE_STICKS=0
TEAMS_FILE=/etc/showcase/teams.json
EOF
  echo "wrote /etc/showcase/server.env - set ORGANISER_SECRET"
fi

echo "== service"
$SSH "$HOST" '/etc/init.d/showcase-server enable; /etc/init.d/showcase-server restart; sleep 2; logread -e showcase | tail -5'

echo "== done: dashboard http://${HOST#*@}:8080/  MCP http://${HOST#*@}:8080/mcp/<code>  broker ${HOST#*@}:1883"
