#!/usr/bin/env bash
# Stop the background production binary started by scripts/start.sh.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PID_FILE="${PID_FILE:-bin/agent-community.pid}"

if [[ ! -f "$PID_FILE" ]]; then
  echo "agent-community is not running (missing $PID_FILE)"
  exit 0
fi

pid="$(cat "$PID_FILE")"
if kill -0 "$pid" 2>/dev/null; then
  kill "$pid"
  rm -f "$PID_FILE"
  echo "agent-community stopped: pid $pid"
else
  rm -f "$PID_FILE"
  echo "agent-community was not running; removed stale $PID_FILE"
fi
