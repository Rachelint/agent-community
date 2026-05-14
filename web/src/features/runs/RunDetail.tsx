import { useEffect, useRef, useState } from 'react';
import {
  fetchLogChunk,
  useCancelRun,
  useIssue,
  useMarkOrphan,
  useProbeRun,
  useRun,
} from '../../api/hooks';
import type { LogChunk, WorkerRun } from '../../api/types';
import { RunStatusPill, formatDuration } from './util';

type Props = {
  runId: string;
  onBack: () => void;
};

type LogStreamName = 'stdout' | 'stderr' | 'events';

// Run detail page. Hidden by default; reached by clicking a run row in
// the issue's right panel. Polls status and stdout incrementally.
export function RunDetail({ runId, onBack }: Props) {
  const { data: run } = useRun(runId);
  const { data: issue } = useIssue(run?.issue_id ?? null);
  const [stream, setStream] = useState<LogStreamName>('stdout');

  if (!run) {
    return (
      <div className="p-6 text-xs text-muted">
        <button onClick={onBack} className="underline">
          ← back
        </button>
        <p className="mt-4">loading…</p>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b border-border bg-canvas-subtle px-4 py-2 text-xs">
        <button onClick={onBack} className="text-muted hover:text-fg">
          ← issue
        </button>
        <span className="text-muted">/</span>
        <span className="font-mono">{runId.slice(0, 8)}</span>
        <RunStatusPill status={run.status} />
        {issue ? (
          <span className="ml-auto text-[11px] text-muted">
            #{issue.number} {issue.title}
          </span>
        ) : null}
      </div>
      <div className="flex-1 overflow-hidden">
        <div className="flex h-full gap-4 p-4">
          <div className="flex min-w-0 flex-1 flex-col overflow-hidden rounded-md border border-border bg-canvas">
            <div className="flex items-center gap-2 border-b border-border px-3 py-1.5 text-[10px] uppercase text-muted">
              {(['stdout', 'stderr', 'events'] as LogStreamName[]).map((name) => (
                <button
                  key={name}
                  type="button"
                  onClick={() => setStream(name)}
                  className={`rounded px-1.5 py-0.5 ${
                    stream === name
                      ? 'bg-border/60 font-semibold text-fg'
                      : 'hover:text-fg'
                  }`}
                >
                  {name}
                </button>
              ))}
              {run.status === 'running' ? (
                <span className="text-success">· live</span>
              ) : null}
            </div>
            <LogStream run={run} stream={stream} />
          </div>
          <aside className="w-60 shrink-0 space-y-3 text-xs">
            <MetaRow label="plugin">{run.plugin}</MetaRow>
            <MetaRow label="started">
              {run.started_at ? new Date(run.started_at).toLocaleString() : '—'}
            </MetaRow>
            <MetaRow label="duration">
              {run.started_at
                ? formatDuration(run.started_at, run.finished_at)
                : '—'}
            </MetaRow>
            {run.pid ? <MetaRow label="pid">{run.pid}</MetaRow> : null}
            {run.mr_url ? (
              <MetaRow label="mr">
                <a
                  href={run.mr_url}
                  target="_blank"
                  rel="noreferrer"
                  className="break-all text-accent hover:underline"
                >
                  {run.mr_url}
                </a>
              </MetaRow>
            ) : null}
            {run.summary ? (
              <MetaRow label="summary">
                <p className="whitespace-pre-wrap">{run.summary}</p>
              </MetaRow>
            ) : null}
            {run.status === 'running' ? <RunControls run={run} /> : null}
          </aside>
        </div>
      </div>
    </div>
  );
}

function MetaRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="text-[10px] uppercase text-muted">{label}</div>
      <div>{children}</div>
    </div>
  );
}

function LogStream({ run, stream }: { run: WorkerRun; stream: LogStreamName }) {
  const [logs, setLogs] = useState<Record<LogStreamName, string>>({
    stdout: '',
    stderr: '',
    events: '',
  });
  const offsetsRef = useRef<Record<LogStreamName, number>>({
    stdout: 0,
    stderr: 0,
    events: 0,
  });
  const [err, setErr] = useState<string | null>(null);
  const scrollerRef = useRef<HTMLPreElement>(null);

  // Polling cadence: fast while running, slow after terminal. We
  // deliberately don't unmount the effect on status change so one final
  // tail always sweeps the remaining bytes.
  const active = run.status === 'running' || run.status === 'queued';

  useEffect(() => {
    setLogs({ stdout: '', stderr: '', events: '' });
    offsetsRef.current = { stdout: 0, stderr: 0, events: 0 };
    setErr(null);
  }, [run.id]);

  useEffect(() => {
    let cancelled = false;
    const tick = async () => {
      try {
        const chunk: LogChunk = await fetchLogChunk(
          run.id,
          stream,
          offsetsRef.current[stream],
        );
        if (cancelled) return;
        if (chunk.chunk) {
          offsetsRef.current[stream] = chunk.next;
          setLogs((prev) => ({ ...prev, [stream]: prev[stream] + chunk.chunk }));
          // Auto-scroll to bottom if user was already near the end.
          const el = scrollerRef.current;
          if (el && el.scrollHeight - el.scrollTop - el.clientHeight < 40) {
            requestAnimationFrame(() => {
              el.scrollTop = el.scrollHeight;
            });
          }
        }
        setErr(null);
      } catch (e) {
        setErr(String(e));
      }
    };
    tick();
    if (!active) return;
    const id = setInterval(tick, 1500);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
    // rerun when run id changes, stream changes, or the status flips terminal
  }, [run.id, active, stream]);

  return (
    <pre
      ref={scrollerRef}
      className="flex-1 overflow-auto whitespace-pre-wrap px-3 py-2 font-mono text-[11px] leading-relaxed text-fg"
    >
      {logs[stream] || <span className="text-muted">no output yet</span>}
      {err ? <span className="text-danger">\n{err}</span> : null}
    </pre>
  );
}

function RunControls({ run }: { run: WorkerRun }) {
  const cancel = useCancelRun(run.issue_id);
  const probe = useProbeRun();
  const markOrphan = useMarkOrphan(run.issue_id);
  const [probed, setProbed] = useState<{ alive: boolean } | null>(null);

  return (
    <div className="space-y-1 border-t border-border pt-2">
      <button
        onClick={() => {
          probe.mutate(run.id, {
            onSuccess: (r) => setProbed({ alive: r.alive }),
          });
        }}
        className="w-full rounded-md border border-border px-2 py-1 text-xs text-muted hover:text-fg"
      >
        probe liveness
      </button>
      {probed ? (
        <p
          className={`text-[11px] ${probed.alive ? 'text-success' : 'text-danger'}`}
        >
          {probed.alive ? 'process is alive' : 'process is dead'}
        </p>
      ) : null}
      {probed && !probed.alive ? (
        <button
          onClick={() => {
            if (confirm('Mark this run as orphan?')) {
              markOrphan.mutate(run.id);
            }
          }}
          className="w-full rounded-md border border-danger px-2 py-1 text-xs text-danger hover:bg-danger hover:text-canvas"
        >
          mark orphan
        </button>
      ) : null}
      <button
        onClick={() => {
          if (confirm('Cancel this run? SIGTERM will be sent.')) {
            cancel.mutate(run.id);
          }
        }}
        className="w-full rounded-md border border-border px-2 py-1 text-xs text-muted hover:text-fg"
      >
        cancel
      </button>
    </div>
  );
}
