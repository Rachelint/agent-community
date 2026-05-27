#!/usr/bin/env bash
# Build and run the production binary in the background.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PID_FILE="${PID_FILE:-bin/agent-community.pid}"
LOG_FILE="${LOG_FILE:-bin/agent-community.log}"

if [[ -f "$PID_FILE" ]] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
  echo "agent-community is already running: pid $(cat "$PID_FILE")"
  exit 0
fi

make build
mkdir -p "$(dirname "$PID_FILE")" "$(dirname "$LOG_FILE")"

nohup ./bin/agent-community > "$LOG_FILE" 2>&1 &
echo $! > "$PID_FILE"

echo "agent-community started: pid $(cat "$PID_FILE")"
echo "url: http://127.0.0.1${AC_ADDR:-:8080}"
echo "log: $LOG_FILE"
echo "pid: $PID_FILE"
