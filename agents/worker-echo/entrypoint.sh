#!/usr/bin/env bash
# Reference worker agent. Demonstrates the host/agent contract:
#
#   1. Print a single worker.ready JSON-RPC frame to stdout so the host
#      knows we are alive; the host returns from Spawn() at that point.
#   2. Post log lines and the final completion to the host via HTTP.
#
# The host also writes a worker.start frame to our stdin and then
# closes it; we don't need the frame (the interesting fields — run id,
# callback URL, token, prompt — are mirrored into env vars), so we
# simply ignore stdin.
set -u

: "${AC_RUN_ID:?}"
: "${AC_CALLBACK_URL:?}"
: "${AC_PLUGIN_TOKEN:?}"

base="${AC_CALLBACK_URL%/}/plugin/runs/${AC_RUN_ID}"

# log posts a single line to the host's log sink. Errors are swallowed
# so transient network hiccups don't tear down the whole worker.
log() {
  local payload
  payload=$(python3 -c 'import sys,json;print(json.dumps({"stream":"stdout","content":sys.argv[1]+"\n"}))' "$1")
  curl -sS -X POST "${base}/log" \
    -H "Authorization: Bearer ${AC_PLUGIN_TOKEN}" \
    -H "Content-Type: application/json" \
    --data "$payload" >/dev/null 2>&1 || true
}

# Announce readiness. The newline is important — the host reads stdout
# line-by-line.
printf '{"jsonrpc":"2.0","method":"worker.ready","params":{}}\n'

log "worker-echo started for run ${AC_RUN_ID}"
for i in 1 2 3 4 5; do
  log "tick ${i}"
  sleep 1
done

summary="echo worker finished 5 ticks"
curl -sS -X POST "${base}/complete" \
  -H "Authorization: Bearer ${AC_PLUGIN_TOKEN}" \
  -H "Content-Type: application/json" \
  --data "{\"status\":\"needs_review\",\"mr_url\":\"https://example.com/mr/${AC_RUN_ID}\",\"summary\":\"${summary}\"}" \
  >/dev/null 2>&1 || true
