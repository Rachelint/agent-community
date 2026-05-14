import { useMemo, useState } from 'react';
import {
  useAgentMembers,
  useCancelRun,
  useDispatch,
  useIssueRuns,
  useMarkOrphan,
  useProbeRun,
  useReloadAgents,
} from '../../api/hooks';
import type { WorkerRun } from '../../api/types';
import { RunStatusPill, formatDuration } from './util';

type Props = {
  issueId: string;
  onOpenRun: (runId: string) => void;
};

// Dispatch + recent runs UI. Lives inside IssueDetail's right-hand
// panel starting in phase 3.
export function DispatchPanel({ issueId, onOpenRun }: Props) {
  const { data: workers } = useAgentMembers('worker');
  const { data: runs } = useIssueRuns(issueId);
  const dispatch = useDispatch(issueId);
  const reload = useReloadAgents();

  const enabledWorkers = useMemo(
    () => (workers ?? []).filter((w) => w.enabled),
    [workers],
  );
  const [plugin, setPlugin] = useState<string>('');
  const activePlugin = plugin || enabledWorkers[0]?.name || '';
  const running = (runs ?? []).find((r) => r.status === 'running' || r.status === 'queued');

  return (
    <div className="space-y-3">
      <div>
        <div className="mb-1 flex items-center justify-between text-[10px] uppercase text-muted">
          <span>Dispatch</span>
          <button
            onClick={() => reload.mutate()}
            className="text-muted hover:text-fg"
            title="rescan agents/ manifests"
          >
            {reload.isPending ? 'reloading…' : 'reload'}
          </button>
        </div>
        <select
          value={activePlugin}
          onChange={(e) => setPlugin(e.target.value)}
          className="w-full rounded-md border border-border bg-canvas px-2 py-1 text-xs"
          disabled={enabledWorkers.length === 0}
        >
          {enabledWorkers.length === 0 ? (
            <option value="">no worker agents</option>
          ) : (
            enabledWorkers.map((w) => (
              <option key={w.name} value={w.name}>
                {w.name}
              </option>
            ))
          )}
        </select>
        <button
          disabled={!activePlugin || !!running || dispatch.isPending}
          onClick={() =>
            dispatch.mutate(
              { plugin: activePlugin },
              { onSuccess: (run) => onOpenRun(run.id) },
            )
          }
          className="mt-1 w-full rounded-md border border-border bg-accent px-2 py-1 text-xs font-semibold text-canvas hover:opacity-90 disabled:opacity-50"
        >
          {running
            ? 'a run is still active'
            : dispatch.isPending
              ? 'dispatching…'
              : 'dispatch worker'}
        </button>
        {dispatch.error ? (
          <p className="mt-1 text-[11px] text-danger">
            {(dispatch.error as Error).message}
          </p>
        ) : null}
      </div>

      <RunsList issueId={issueId} runs={runs ?? []} onOpenRun={onOpenRun} />
    </div>
  );
}

function RunsList({
  issueId,
  runs,
  onOpenRun,
}: {
  issueId: string;
  runs: WorkerRun[];
  onOpenRun: (id: string) => void;
}) {
  if (runs.length === 0) {
    return (
      <div>
        <div className="mb-1 text-[10px] uppercase text-muted">Runs</div>
        <p className="text-xs text-muted">no runs yet</p>
      </div>
    );
  }
  return (
    <div>
      <div className="mb-1 text-[10px] uppercase text-muted">Runs</div>
      <ul className="space-y-1">
        {runs.slice(0, 5).map((r) => (
          <RunRow key={r.id} run={r} issueId={issueId} onOpen={onOpenRun} />
        ))}
      </ul>
    </div>
  );
}

function RunRow({
  run,
  issueId,
  onOpen,
}: {
  run: WorkerRun;
  issueId: string;
  onOpen: (id: string) => void;
}) {
  const cancel = useCancelRun(issueId);
  const probe = useProbeRun();
  const markOrphan = useMarkOrphan(issueId);
  const [probeResult, setProbeResult] = useState<{ alive: boolean } | null>(null);

  return (
    <li className="rounded-md border border-border bg-canvas px-2 py-1.5 text-xs">
      <div className="flex items-center gap-2">
        <button
          onClick={() => onOpen(run.id)}
          className="truncate font-mono text-[11px] text-muted hover:text-fg"
          title="open run"
        >
          {run.id.slice(0, 8)}
        </button>
        <RunStatusPill status={run.status} />
        <span className="ml-auto text-[10px] text-muted">
          {run.plugin}
          {run.started_at ? ` · ${formatDuration(run.started_at, run.finished_at)}` : ''}
        </span>
      </div>
      {run.mr_url ? (
        <a
          href={run.mr_url}
          target="_blank"
          rel="noreferrer"
          className="mt-0.5 block truncate text-[11px] text-accent hover:underline"
        >
          {run.mr_url}
        </a>
      ) : null}
      {run.status === 'running' ? (
        <div className="mt-1 flex items-center gap-1.5 text-[10px]">
          <button
            onClick={() => {
              probe.mutate(run.id, {
                onSuccess: (r) => setProbeResult({ alive: r.alive }),
              });
            }}
            className="rounded border border-border px-1.5 py-0.5 text-muted hover:text-fg"
          >
            probe
          </button>
          {probeResult ? (
            <span
              className={probeResult.alive ? 'text-success' : 'text-danger'}
            >
              {probeResult.alive ? 'alive' : 'dead'}
            </span>
          ) : null}
          {probeResult && !probeResult.alive ? (
            <button
              onClick={() => {
                if (confirm(`Mark run ${run.id.slice(0, 8)} as orphan? This releases the issue slot.`)) {
                  markOrphan.mutate(run.id);
                }
              }}
              className="rounded border border-danger px-1.5 py-0.5 text-danger hover:bg-danger hover:text-canvas"
            >
              mark orphan
            </button>
          ) : null}
          <button
            onClick={() => cancel.mutate(run.id)}
            className="ml-auto rounded border border-border px-1.5 py-0.5 text-muted hover:text-fg"
            title="send SIGTERM"
          >
            cancel
          </button>
        </div>
      ) : null}
    </li>
  );
}
