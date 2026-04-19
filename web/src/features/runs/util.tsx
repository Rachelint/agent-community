import type { RunStatus } from '../../api/types';

// Status pill shared across the Issue / Run views.
const styles: Record<RunStatus, string> = {
  queued:       'bg-canvas-subtle text-muted',
  running:      'bg-accent/10 text-accent',
  needs_review: 'bg-[#dbeafe] text-accent',
  completed:    'bg-[#dcfce7] text-success',
  failed:       'bg-[#fee2e2] text-danger',
  cancelled:    'bg-canvas-subtle text-muted',
  orphan:       'bg-[#fef3c7] text-[#92400e]',
};

export function RunStatusPill({ status }: { status: RunStatus }) {
  return (
    <span
      className={`rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase ${styles[status]}`}
    >
      {status.replace('_', ' ')}
    </span>
  );
}

// Pretty millisecond duration, e.g. "1m 23s".
export function formatDuration(start?: number | null, end?: number | null): string {
  if (!start) return '';
  const stop = end ?? Date.now();
  const ms = Math.max(0, stop - start);
  const s = Math.floor(ms / 1000);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  const rs = s % 60;
  if (m < 60) return rs ? `${m}m ${rs}s` : `${m}m`;
  const h = Math.floor(m / 60);
  const rm = m % 60;
  return rm ? `${h}h ${rm}m` : `${h}h`;
}
