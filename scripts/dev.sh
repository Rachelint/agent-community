#!/usr/bin/env bash
# Run Go server and Vite dev server in parallel; Ctrl-C kills both.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

pids=()
cleanup() {
  echo
  echo "[dev] stopping..."
  for pid in "${pids[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "[dev] starting Go server on :8080"
(go run ./cmd/server) &
pids+=($!)

echo "[dev] starting Vite on :5173"
(cd web && npm run dev) &
pids+=($!)

wait
