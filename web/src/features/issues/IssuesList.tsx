import { useMemo, useState } from 'react';
import { useIssues, useLabels } from '../../api/hooks';
import type { Issue, IssueFilter } from '../../api/types';
import { LabelBadge } from './LabelBadge';
import { LabelManager } from './LabelManager';
import { NewIssueForm } from './NewIssueForm';

type Props = {
  projectId: string;
  onOpenIssue: (id: string) => void;
};

// GitHub-style issue list. Top-level issues are shown at the root;
// their children are nested under them. We fetch the full list (no
// parent filter) then group client-side — keeps filters simple and the
// server call count at one.
export function IssuesList({ projectId, onOpenIssue }: Props) {
  const [status, setStatus] = useState<'' | 'open' | 'closed'>('open');
  const [labelID, setLabelID] = useState<string>('');
  const [q, setQ] = useState('');
  const [showNew, setShowNew] = useState(false);
  const [showLabels, setShowLabels] = useState(false);

  const filter: IssueFilter = useMemo(
    () => ({ status: status || undefined, label: labelID || undefined, q: q || undefined }),
    [status, labelID, q],
  );

  const { data: issues, isLoading } = useIssues(projectId, filter);
  const { data: labels } = useLabels(projectId);

  // Group: top-level + children map
  const groups = useMemo(() => {
    const top: Issue[] = [];
    const kids: Record<string, Issue[]> = {};
    for (const i of issues ?? []) {
      if (i.parent_id) {
        (kids[i.parent_id] ??= []).push(i);
      } else {
        top.push(i);
      }
    }
    // Children that we fetched but whose parent was filtered out still
    // need to show up so the user is aware; surface them at the top too.
    for (const i of issues ?? []) {
      if (i.parent_id && !top.find((p) => p.id === i.parent_id)) {
        if (!top.find((p) => p.id === i.id)) top.push(i);
      }
    }
    return { top, kids };
  }, [issues]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b border-border bg-canvas-subtle px-4 py-2">
        <div className="flex rounded-md border border-border">
          {(['open', 'closed', ''] as const).map((s) => (
            <button
              key={s || 'all'}
              onClick={() => setStatus(s)}
              className={`px-2 py-1 text-xs ${
                status === s
                  ? 'bg-border/60 font-semibold text-fg'
                  : 'text-muted hover:bg-border/30'
              }`}
            >
              {s || 'all'}
            </button>
          ))}
        </div>

        <select
          value={labelID}
          onChange={(e) => setLabelID(e.target.value)}
          className="rounded-md border border-border bg-canvas px-2 py-1 text-xs"
        >
          <option value="">all labels</option>
          {(labels ?? []).map((l) => (
            <option key={l.id} value={l.id}>
              {l.name}
            </option>
          ))}
        </select>

        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="search title/body…"
          className="flex-1 rounded-md border border-border bg-canvas px-2 py-1 text-xs outline-none focus:border-accent"
        />

        <button
          onClick={() => setShowLabels((v) => !v)}
          className="rounded-md border border-border bg-canvas px-2 py-1 text-xs hover:bg-border/40"
        >
          labels
        </button>
        <button
          onClick={() => setShowNew((v) => !v)}
          className="rounded-md border border-border bg-accent px-3 py-1 text-xs font-semibold text-canvas hover:opacity-90"
        >
          new issue
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="space-y-3 p-4">
          {showNew ? (
            <NewIssueForm
              projectId={projectId}
              onCreated={() => setShowNew(false)}
              onCancel={() => setShowNew(false)}
            />
          ) : null}
          {showLabels ? <LabelManager projectId={projectId} /> : null}

          {isLoading ? (
            <p className="text-xs text-muted">loading…</p>
          ) : !issues || issues.length === 0 ? (
            <p className="text-xs text-muted">
              no issues match. try changing filters or{' '}
              <button
                className="underline"
                onClick={() => setShowNew(true)}
              >
                create one
              </button>
              .
            </p>
          ) : (
            <ul className="divide-y divide-border rounded-md border border-border bg-canvas">
              {groups.top.map((iss) => (
                <IssueRow
                  key={iss.id}
                  issue={iss}
                  indent={0}
                  onOpen={onOpenIssue}
                  children={groups.kids[iss.id] ?? []}
                />
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}

function IssueRow({
  issue,
  indent,
  onOpen,
  children,
}: {
  issue: Issue;
  indent: number;
  onOpen: (id: string) => void;
  children: Issue[];
}) {
  return (
    <>
      <li
        onClick={() => onOpen(issue.id)}
        className="flex cursor-pointer items-start gap-3 px-3 py-2 text-xs hover:bg-border/20"
        style={{ paddingLeft: `${12 + indent * 24}px` }}
      >
        <span
          className={`mt-0.5 inline-block h-2 w-2 flex-shrink-0 rounded-full ${
            issue.status === 'open' ? 'bg-success' : 'bg-muted'
          }`}
          title={issue.status}
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="font-mono text-[11px] text-muted">
              #{issue.number}
            </span>
            <span className="truncate font-semibold">{issue.title}</span>
            {issue.child_count > 0 ? (
              <span className="rounded bg-canvas-subtle px-1 text-[10px] text-muted">
                {issue.child_count} sub
              </span>
            ) : null}
          </div>
          {issue.labels.length > 0 ? (
            <div className="mt-1 flex flex-wrap gap-1">
              {issue.labels.map((l) => (
                <LabelBadge key={l.id} label={l} />
              ))}
            </div>
          ) : null}
        </div>
        {issue.assignee ? (
          <span className="flex-shrink-0 rounded bg-canvas-subtle px-1.5 py-0.5 text-[10px] text-muted">
            {issue.assignee}
          </span>
        ) : null}
      </li>
      {children.map((c) => (
        <IssueRow
          key={c.id}
          issue={c}
          indent={indent + 1}
          onOpen={onOpen}
          children={[]}
        />
      ))}
    </>
  );
}
