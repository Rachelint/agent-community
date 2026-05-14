#!/usr/bin/env bash
set -u
: "${AC_RUN_ID:?}"
: "${AC_RUN_DIR:?}"
: "${AC_WORKSPACE:?}"

printf '{"jsonrpc":"2.0","method":"worker.ready","params":{}}\n'

prompt_file="${AC_RUN_DIR}/prompt.md"
if [[ ! -f "${prompt_file}" ]]; then
  printf '{"status":"failed","exit_code":1,"summary":"prompt.md is missing","mr_url":""}' > "${AC_RUN_DIR}/done.json"
  exit 1
fi

set +e
codebuddy --print -y --model claude-opus-4.7-1m "$(cat "${prompt_file}")"
code=$?
set -e

if [[ ${code} -eq 0 ]]; then
  printf '{"status":"needs_review","exit_code":0,"summary":"codebuddy worker completed; review stdout.log and the issue worktree","mr_url":""}' > "${AC_RUN_DIR}/done.json"
else
  printf '{"status":"failed","exit_code":%d,"summary":"codebuddy worker failed; review stdout.log and stderr.log","mr_url":""}' "${code}" > "${AC_RUN_DIR}/done.json"
fi

exit "${code}"
