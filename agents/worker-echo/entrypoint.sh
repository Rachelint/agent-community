#!/usr/bin/env bash
set -u
: "${AC_RUN_ID:?}"
: "${AC_RUN_DIR:?}"

# Signal readiness to host
printf '{"jsonrpc":"2.0","method":"worker.ready","params":{}}\n'

echo "worker-echo started for run ${AC_RUN_ID}"
for i in 1 2 3 4 5; do
  echo "tick ${i}"
  sleep 1
done

# Write done.json (deterministic - always runs after echo worker logic)
printf '{"status":"needs_review","exit_code":0,"summary":"echo worker finished 5 ticks","mr_url":"https://example.com/mr/%s"}' \
  "$AC_RUN_ID" > "${AC_RUN_DIR}/done.json"
