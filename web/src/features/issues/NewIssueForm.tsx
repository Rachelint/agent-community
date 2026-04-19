import { useState } from 'react';
import { useCreateIssue, useLabels } from '../../api/hooks';
import type { Issue } from '../../api/types';
import { labelTextColor } from './palette';

// Inline form to create a top-level issue. Child issues are created from
// inside a parent's detail view.
export function NewIssueForm({
  projectId,
  onCreated,
  onCancel,
}: {
  projectId: string;
  onCreated: (iss: Issue) => void;
  onCancel: () => void;
}) {
  const { data: labels } = useLabels(projectId);
  const create = useCreateIssue(projectId);
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const toggle = (id: string) => {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelected(next);
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        if (!title.trim()) return;
        create.mutate(
          {
            title: title.trim(),
            body: body.trim() || undefined,
            labels: Array.from(selected),
          },
          {
            onSuccess: (iss) => {
              setTitle('');
              setBody('');
              setSelected(new Set());
              onCreated(iss);
            },
          },
        );
      }}
      className="space-y-2 rounded-md border border-border bg-canvas p-3 text-xs"
    >
      <h3 className="text-[11px] font-semibold uppercase tracking-wide text-muted">
        New issue
      </h3>
      <label className="flex flex-col">
        <span className="mb-1 text-[10px] text-muted">Title</span>
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          required
          className="rounded-md border border-border bg-canvas px-2 py-1 text-xs outline-none focus:border-accent"
        />
      </label>
      <label className="flex flex-col">
        <span className="mb-1 text-[10px] text-muted">Body (markdown)</span>
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          rows={4}
          className="rounded-md border border-border bg-canvas px-2 py-1 font-mono text-xs outline-none focus:border-accent"
        />
      </label>
      {labels && labels.length > 0 ? (
        <div>
          <p className="mb-1 text-[10px] text-muted">Labels</p>
          <div className="flex flex-wrap gap-1.5">
            {labels.map((l) => {
              const active = selected.has(l.id);
              return (
                <button
                  type="button"
                  key={l.id}
                  onClick={() => toggle(l.id)}
                  className="rounded-full px-2 py-0.5 text-[10px] font-semibold transition"
                  style={{
                    backgroundColor: active ? `#${l.color}` : 'transparent',
                    color: active ? labelTextColor(l.color) : '#656d76',
                    border: `1px solid #${l.color}`,
                  }}
                >
                  {l.name}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}
      {create.error ? (
        <p className="text-[11px] text-danger">
          {(create.error as Error).message}
        </p>
      ) : null}
      <div className="flex justify-end gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md border border-border px-3 py-1 text-[11px] text-muted hover:text-fg"
        >
          cancel
        </button>
        <button
          type="submit"
          disabled={create.isPending || !title.trim()}
          className="rounded-md border border-border bg-canvas-subtle px-3 py-1 text-[11px] font-semibold hover:bg-border/40 disabled:opacity-50"
        >
          {create.isPending ? 'creating…' : 'create'}
        </button>
      </div>
    </form>
  );
}
