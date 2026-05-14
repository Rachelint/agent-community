#!/usr/bin/env bash
set -u
: "${AC_RUN_ID:?}"
: "${AC_RUN_DIR:?}"
: "${AC_CALLBACK_URL:?}"
: "${AC_PLUGIN_TOKEN:?}"

callback() {
  local path="$1"
  local payload="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsS \
      -H "Authorization: Bearer ${AC_PLUGIN_TOKEN}" \
      -H "Content-Type: application/json" \
      -d "${payload}" \
      "${AC_CALLBACK_URL}${path}" >/dev/null || true
  fi
}

# Signal readiness to host.
printf '{"jsonrpc":"2.0","method":"worker.ready","params":{}}\n'

callback "/plugin/runs/${AC_RUN_ID}/log" '{"stream":"events","data":"{\"event\":\"echo.started\"}\n"}'
echo "worker-echo started for run ${AC_RUN_ID}"
for i in 1 2 3 4 5; do
  echo "tick ${i}"
  callback "/plugin/runs/${AC_RUN_ID}/log" "{\"stream\":\"events\",\"data\":\"{\\\"event\\\":\\\"echo.tick\\\",\\\"tick\\\":${i}}\\n\"}"
  sleep 1
done

summary="echo worker finished 5 ticks"
mr_url="https://example.com/mr/${AC_RUN_ID}"

# Write done.json as the filesystem fallback for the reconciler.
printf '{"status":"needs_review","exit_code":0,"summary":"%s","mr_url":"%s"}' \
  "${summary}" "${mr_url}" > "${AC_RUN_DIR}/done.json"

# Prefer the explicit callback path when the host is reachable.
callback "/plugin/runs/${AC_RUN_ID}/complete" \
  "{\"status\":\"needs_review\",\"exit_code\":0,\"summary\":\"${summary}\",\"mr_url\":\"${mr_url}\"}"
