import { useState } from 'react';
import {
  useAgentMembers,
  useCreateIssue,
  useCreateIssueComment,
  useDeleteIssue,
  useIssue,
  useIssueComments,
  useIssues,
  useLabels,
  usePatchIssue,
} from '../../api/hooks';
import type { Issue } from '../../api/types';
import { Markdown } from '../../components/Markdown';
import { DispatchPanel } from '../runs/DispatchPanel';
import { LabelBadge } from './LabelBadge';
import { labelTextColor } from './palette';

type Props = {
  issueId: string;
  onBack: () => void;
  onOpenIssue: (id: string) => void;
  onOpenRun: (runId: string) => void;
};

export function IssueDetail({ issueId, onBack, onOpenIssue, onOpenRun }: Props) {
  const { data: issue, isLoading } = useIssue(issueId);

  if (isLoading || !issue) {
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
          ← issues
        </button>
        <span className="text-muted">/</span>
        <span className="font-mono text-muted">#{issue.number}</span>
      </div>

      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 overflow-y-auto p-6">
          <IssueHeader issue={issue} />
          <section className="mt-4 rounded-md border border-border bg-canvas p-4">
            <div className="mb-2 text-[10px] uppercase text-muted">
              description
            </div>
            <Markdown source={issue.body} />
          </section>
          <ChildrenSection
            issue={issue}
            onOpen={onOpenIssue}
          />
          <CommentsSection issueId={issue.id} />
        </div>
        <RightPanel issue={issue} onOpenRun={onOpenRun} />
      </div>
    </div>
  );
}

function IssueHeader({ issue }: { issue: Issue }) {
  const patch = usePatchIssue(issue.id, issue.project_id);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(issue.title);

  if (editing) {
    return (
      <div className="flex items-center gap-2">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="flex-1 rounded-md border border-border bg-canvas px-2 py-1 text-base outline-none focus:border-accent"
        />
        <button
          onClick={() => {
            patch.mutate({ title }, { onSuccess: () => setEditing(false) });
          }}
          className="rounded-md border border-border bg-accent px-3 py-1 text-xs text-canvas"
        >
          save
        </button>
        <button
          onClick={() => {
            setTitle(issue.title);
            setEditing(false);
          }}
          className="rounded-md border border-border px-3 py-1 text-xs text-muted"
        >
          cancel
        </button>
      </div>
    );
  }

  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <h1
          className="text-xl font-semibold leading-tight"
          onDoubleClick={() => setEditing(true)}
          title="double click to rename"
        >
          {issue.title}
        </h1>
        <p className="mt-1 text-xs text-muted">
          <span
            className={
              issue.status === 'open' ? 'text-success' : 'text-muted'
            }
          >
            {issue.status}
          </span>{' '}
          · opened {new Date(issue.created_at).toLocaleString()}
          {issue.closed_at
            ? ` · closed ${new Date(issue.closed_at).toLocaleString()}`
            : ''}
        </p>
      </div>
      <button
        onClick={() => setEditing(true)}
        className="rounded-md border border-border px-2 py-1 text-[11px] text-muted hover:text-fg"
      >
        rename
      </button>
    </div>
  );
}

function RightPanel({
  issue,
  onOpenRun,
}: {
  issue: Issue;
  onOpenRun: (id: string) => void;
}) {
  const patch = usePatchIssue(issue.id, issue.project_id);
  const del = useDeleteIssue(issue.project_id);
  const { data: labels } = useLabels(issue.project_id);
  const { data: workers } = useAgentMembers('worker');
  const selected = new Set(issue.labels.map((l) => l.id));

  return (
    <aside className="w-64 overflow-y-auto border-l border-border bg-canvas-subtle p-4 text-xs">
      <div className="space-y-4">
        <DispatchPanel issueId={issue.id} onOpenRun={onOpenRun} />

        <div className="border-t border-border pt-3">
          <div className="mb-1 text-[10px] uppercase text-muted">Status</div>
          <button
            onClick={() =>
              patch.mutate({
                status: issue.status === 'open' ? 'closed' : 'open',
              })
            }
            className="w-full rounded-md border border-border bg-canvas px-2 py-1 text-xs hover:bg-border/40"
          >
            {issue.status === 'open' ? 'close issue' : 'reopen issue'}
          </button>
        </div>

        <div>
          <div className="mb-1 text-[10px] uppercase text-muted">Assignee</div>
          <select
            value={issue.assignee ?? ''}
            onChange={(e) => patch.mutate({ assignee: e.target.value })}
            className="w-full rounded-md border border-border bg-canvas px-2 py-1 text-xs"
          >
            <option value="">unassigned</option>
            {(workers ?? []).map((w) => (
              <option key={w.name} value={w.name}>
                {w.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <div className="mb-1 text-[10px] uppercase text-muted">Labels</div>
          <div className="flex flex-wrap gap-1">
            {(labels ?? []).map((l) => {
              const on = selected.has(l.id);
              return (
                <button
                  key={l.id}
                  onClick={() => {
                    const next = new Set(selected);
                    if (on) next.delete(l.id);
                    else next.add(l.id);
                    patch.mutate({ labels: Array.from(next) });
                  }}
                  className="rounded-full px-2 py-0.5 text-[10px] font-semibold"
                  style={{
                    backgroundColor: on ? `#${l.color}` : 'transparent',
                    color: on ? labelTextColor(l.color) : '#656d76',
                    border: `1px solid #${l.color}`,
                  }}
                >
                  {l.name}
                </button>
              );
            })}
          </div>
        </div>

        <div>
          <div className="mb-1 text-[10px] uppercase text-muted">Parent</div>
          <p className="text-xs">
            {issue.parent_id ? (
              <span className="font-mono">{issue.parent_id.slice(0, 8)}…</span>
            ) : (
              <span className="text-muted">top-level</span>
            )}
          </p>
        </div>

        <div className="border-t border-border pt-3">
          <button
            onClick={() => {
              if (confirm(`Delete #${issue.number}?`)) {
                del.mutate(issue.id);
              }
            }}
            className="w-full rounded-md border border-danger px-2 py-1 text-xs text-danger hover:bg-danger hover:text-canvas"
          >
            delete issue
          </button>
        </div>
      </div>
    </aside>
  );
}

function ChildrenSection({
  issue,
  onOpen,
}: {
  issue: Issue;
  onOpen: (id: string) => void;
}) {
  const { data: children } = useIssues(issue.project_id, {
    parent: issue.id,
  });
  const create = useCreateIssue(issue.project_id);
  const [show, setShow] = useState(false);
  const [title, setTitle] = useState('');

  // Only show the sub-issues section for a top-level issue.
  if (issue.parent_id) {
    return null;
  }

  return (
    <section className="mt-4 rounded-md border border-border bg-canvas p-4">
      <div className="mb-2 flex items-center justify-between">
        <div className="text-[10px] uppercase text-muted">sub-issues</div>
        <button
          onClick={() => setShow((v) => !v)}
          className="text-[11px] text-muted hover:text-fg"
        >
          {show ? 'cancel' : '+ add'}
        </button>
      </div>

      {show ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (!title.trim()) return;
            create.mutate(
              { title: title.trim(), parent_id: issue.id },
              {
                onSuccess: () => {
                  setTitle('');
                  setShow(false);
                },
              },
            );
          }}
          className="mb-2 flex gap-2"
        >
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="child issue title"
            className="flex-1 rounded-md border border-border bg-canvas px-2 py-1 text-xs outline-none focus:border-accent"
          />
          <button
            type="submit"
            disabled={create.isPending}
            className="rounded-md border border-border bg-canvas-subtle px-3 py-1 text-[11px] font-semibold hover:bg-border/40 disabled:opacity-50"
          >
            create
          </button>
        </form>
      ) : null}

      {!children || children.length === 0 ? (
        <p className="text-xs text-muted">none</p>
      ) : (
        <ul className="divide-y divide-border">
          {children.map((c) => (
            <li
              key={c.id}
              onClick={() => onOpen(c.id)}
              className="flex cursor-pointer items-center gap-2 py-1.5 text-xs hover:bg-border/20"
            >
              <span
                className={`inline-block h-1.5 w-1.5 rounded-full ${
                  c.status === 'open' ? 'bg-success' : 'bg-muted'
                }`}
              />
              <span className="font-mono text-[11px] text-muted">
                #{c.number}
              </span>
              <span className="truncate">{c.title}</span>
              <div className="ml-auto flex gap-1">
                {c.labels.map((l) => (
                  <LabelBadge key={l.id} label={l} />
                ))}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function CommentsSection({ issueId }: { issueId: string }) {
  const { data: comments } = useIssueComments(issueId);
  const create = useCreateIssueComment(issueId);
  const [body, setBody] = useState('');

  return (
    <section className="mt-4 space-y-3">
      <div className="text-[10px] uppercase text-muted">comments</div>
      {(comments ?? []).map((c) => (
        <article
          key={c.id}
          className="rounded-md border border-border bg-canvas p-3"
        >
          <header className="mb-1 flex items-center justify-between text-[10px] text-muted">
            <span className="font-semibold">{c.author}</span>
            <span>{new Date(c.created_at).toLocaleString()}</span>
          </header>
          <Markdown source={c.body} />
        </article>
      ))}

      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!body.trim()) return;
          create.mutate(
            { body: body.trim() },
            { onSuccess: () => setBody('') },
          );
        }}
        className="rounded-md border border-border bg-canvas p-3"
      >
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="leave a comment (markdown)…"
          rows={3}
          className="w-full rounded-md border border-border bg-canvas px-2 py-1 font-mono text-xs outline-none focus:border-accent"
        />
        <div className="mt-2 flex justify-end">
          <button
            type="submit"
            disabled={create.isPending || !body.trim()}
            className="rounded-md border border-border bg-canvas-subtle px-3 py-1 text-[11px] font-semibold hover:bg-border/40 disabled:opacity-50"
          >
            comment
          </button>
        </div>
      </form>
    </section>
  );
}
