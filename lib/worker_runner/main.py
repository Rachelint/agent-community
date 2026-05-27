#!/usr/bin/env python3
"""Deterministic worker supervisor for agent-community bundled workers."""

from __future__ import annotations

import argparse
import json
import os
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


class WorkerRuntime:
    def __init__(self, name: str) -> None:
        self.name = name
        self.run_id = require_env("AC_RUN_ID")
        self.run_dir = Path(require_env("AC_RUN_DIR"))
        self.callback_url = require_env("AC_CALLBACK_URL").rstrip("/")
        self.token = require_env("AC_PLUGIN_TOKEN")
        self.prompt_file = self.run_dir / "prompt.md"
        self.done_file = self.run_dir / "done.json"
        self.child: subprocess.Popen[str] | None = None
        self.terminal_reported = False
        self.terminating = False

    def read_prompt(self) -> str:
        return self.prompt_file.read_text(encoding="utf-8")

    def callback(self, path: str, payload: dict[str, Any]) -> bool:
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        req = urllib.request.Request(
            self.callback_url + path,
            data=data,
            headers={
                "Authorization": f"Bearer {self.token}",
                "Content-Type": "application/json",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=10):
                return True
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            print(f"callback failed {path}: {exc}", file=sys.stderr, flush=True)
            return False

    def ready(self) -> None:
        self.callback(f"/plugin/runs/{self.run_id}/ready", {})

    def event(self, event: str, **fields: Any) -> None:
        payload = {
            "event": event,
            "worker": self.name,
            "run_id": self.run_id,
            "ts": int(time.time() * 1000),
            **fields,
        }
        self.callback(
            f"/plugin/runs/{self.run_id}/log",
            {"stream": "events", "data": json.dumps(payload, ensure_ascii=False) + "\n"},
        )

    def write_done(self, status: str, exit_code: int, summary: str, mr_url: str = "") -> None:
        self.run_dir.mkdir(parents=True, exist_ok=True)
        data = {
            "status": status,
            "exit_code": exit_code,
            "summary": summary,
            "mr_url": mr_url,
        }
        tmp = self.done_file.with_suffix(".json.tmp")
        tmp.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
        os.replace(tmp, self.done_file)

    def complete(self, status: str, exit_code: int, summary: str, mr_url: str = "") -> None:
        self.write_done(status, exit_code, summary, mr_url)
        self.callback(
            f"/plugin/runs/{self.run_id}/complete",
            {"status": status, "exit_code": exit_code, "summary": summary, "mr_url": mr_url},
        )
        self.terminal_reported = True

    def fail(self, exit_code: int, summary: str) -> None:
        if exit_code == 0:
            exit_code = 1
        self.write_done("failed", exit_code, summary)
        self.callback(
            f"/plugin/runs/{self.run_id}/fail",
            {"exit_code": exit_code, "summary": summary},
        )
        self.terminal_reported = True

    def run_phase(self, phase: str, command: list[str]) -> int:
        self.event("phase.started", phase=phase, command=command)
        started = time.time()
        self.child = subprocess.Popen(command, text=True)
        code = self.child.wait()
        self.child = None
        duration_ms = int((time.time() - started) * 1000)
        if code == 0:
            self.event("phase.completed", phase=phase, exit_code=code, duration_ms=duration_ms)
        else:
            self.event("phase.failed", phase=phase, exit_code=code, duration_ms=duration_ms)
        return code

    def terminate(self, signum: int, _frame: object) -> None:
        self.terminating = True
        self.event("worker.terminating", signal=signum)
        if self.child and self.child.poll() is None:
            self.child.terminate()


def require_env(name: str) -> str:
    value = os.environ.get(name)
    if not value:
        raise RuntimeError(f"{name} is required")
    return value


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="agent-community deterministic worker runner")
    parser.add_argument("--name", required=True)
    parser.add_argument("--phase", default="agent")
    parser.add_argument("--status-on-success", default="needs_review", choices=("completed", "needs_review"))
    parser.add_argument("--summary-success", required=True)
    parser.add_argument("--summary-failure", required=True)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    if args.command and args.command[0] == "--":
        args.command = args.command[1:]
    if not args.command:
        parser.error("child command is required after --")
    return args


def expand_command(command: list[str], prompt: str) -> list[str]:
    return [part.replace("{prompt}", prompt) for part in command]


def main() -> int:
    args = parse_args()
    rt = WorkerRuntime(args.name)
    signal.signal(signal.SIGTERM, rt.terminate)
    signal.signal(signal.SIGINT, rt.terminate)

    try:
        prompt = rt.read_prompt()
    except Exception as exc:
        rt.ready()
        summary = f"prompt.md is missing or unreadable: {exc}"
        rt.event("phase.failed", phase="prepare", exit_code=1, summary=summary)
        rt.fail(1, summary)
        return 1

    command = expand_command(args.command, prompt)
    rt.ready()
    code = 1
    try:
        code = rt.run_phase(args.phase, command)
        if code == 0:
            rt.complete(args.status_on_success, 0, args.summary_success)
            return 0
        summary = args.summary_failure
        if rt.terminating:
            summary = f"{summary}; worker terminated"
        rt.fail(code, summary)
        return code
    except Exception as exc:
        summary = f"{args.summary_failure}: {exc}"
        rt.event("phase.failed", phase=args.phase, exit_code=1, summary=str(exc))
        rt.fail(1, summary)
        return 1
    finally:
        if not rt.terminal_reported:
            rt.fail(code if code else 1, "worker exited before reporting terminal state")


if __name__ == "__main__":
    raise SystemExit(main())
