import { useState } from 'react';
import {
  useCreateLabel,
  useDeleteLabel,
  useLabels,
} from '../../api/hooks';
import { LABEL_PALETTE, labelTextColor } from './palette';

// Small label editor shown inside the Issues view toolbar. Allows
// creating and removing project-scoped labels. Phase 2 keeps it inline
// rather than behind a dedicated settings page.
export function LabelManager({ projectId }: { projectId: string }) {
  const { data: labels } = useLabels(projectId);
  const create = useCreateLabel(projectId);
  const del = useDeleteLabel(projectId);
  const [name, setName] = useState('');
  const [color, setColor] = useState(LABEL_PALETTE[0].hex);
  const [desc, setDesc] = useState('');

  return (
    <div className="space-y-3 rounded-md border border-border bg-canvas p-3 text-xs">
      <h3 className="text-[11px] font-semibold uppercase tracking-wide text-muted">
        Labels
      </h3>

      <div className="flex flex-wrap gap-1.5">
        {(labels ?? []).map((l) => (
          <span
            key={l.id}
            className="group inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold"
            style={{
              backgroundColor: `#${l.color}`,
              color: labelTextColor(l.color),
            }}
          >
            {l.name}
            <button
              type="button"
              onClick={() => {
                if (confirm(`Delete label "${l.name}"?`)) del.mutate(l.id);
              }}
              className="ml-0.5 opacity-70 hover:opacity-100"
              title="delete label"
            >
              ×
            </button>
          </span>
        ))}
        {labels && labels.length === 0 ? (
          <span className="text-muted">none yet</span>
        ) : null}
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (!name.trim()) return;
          create.mutate(
            {
              name: name.trim(),
              color,
              description: desc.trim() || undefined,
            },
            {
              onSuccess: () => {
                setName('');
                setDesc('');
              },
            },
          );
        }}
        className="flex flex-wrap items-end gap-2"
      >
        <label className="flex flex-col">
          <span className="mb-0.5 text-[10px] text-muted">Name</span>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="bug"
            className="rounded-md border border-border bg-canvas px-2 py-1 text-xs outline-none focus:border-accent"
          />
        </label>
        <label className="flex flex-col">
          <span className="mb-0.5 text-[10px] text-muted">Color</span>
          <div className="flex gap-0.5">
            {LABEL_PALETTE.map((p) => (
              <button
                key={p.hex}
                type="button"
                onClick={() => setColor(p.hex)}
                style={{ backgroundColor: `#${p.hex}` }}
                className={`h-5 w-5 rounded-full border ${
                  color === p.hex ? 'border-fg' : 'border-transparent'
                }`}
                title={p.name}
              />
            ))}
          </div>
        </label>
        <label className="flex flex-1 flex-col">
          <span className="mb-0.5 text-[10px] text-muted">Description</span>
          <input
            value={desc}
            onChange={(e) => setDesc(e.target.value)}
            className="rounded-md border border-border bg-canvas px-2 py-1 text-xs outline-none focus:border-accent"
          />
        </label>
        <button
          type="submit"
          disabled={create.isPending || !name.trim()}
          className="rounded-md border border-border bg-canvas-subtle px-3 py-1 text-xs font-semibold hover:bg-border/40 disabled:opacity-50"
        >
          add
        </button>
      </form>
      {create.error ? (
        <p className="text-[11px] text-danger">
          {(create.error as Error).message}
        </p>
      ) : null}
    </div>
  );
}
