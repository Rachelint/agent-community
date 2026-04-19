import { useState } from 'react';
import { useProjects } from '../../api/hooks';
import type { Project } from '../../api/types';
import { NewProjectForm } from './NewProjectForm';

type Props = {
  activeId: string | null;
  onSelect: (id: string) => void;
};

export function ProjectSwitcher({ activeId, onSelect }: Props) {
  const { data, isLoading, error } = useProjects();
  const [showForm, setShowForm] = useState(false);

  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-between px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-muted">
        <span>Projects</span>
        <button
          onClick={() => setShowForm((v) => !v)}
          className="rounded px-1.5 py-0.5 text-muted hover:bg-border/40 hover:text-fg"
          title={showForm ? 'Cancel' : 'New project'}
        >
          {showForm ? '×' : '+'}
        </button>
      </div>

      {showForm ? (
        <NewProjectForm
          onCreated={(p: Project) => {
            setShowForm(false);
            onSelect(p.id);
          }}
        />
      ) : null}

      {isLoading ? (
        <p className="px-3 py-2 text-xs text-muted">loading…</p>
      ) : error ? (
        <p className="px-3 py-2 text-xs text-danger">
          {(error as Error).message}
        </p>
      ) : !data || data.length === 0 ? (
        <p className="px-3 py-2 text-xs text-muted">
          no projects yet. click <span className="font-mono">+</span> to add one.
        </p>
      ) : (
        <ul className="flex flex-col">
          {data.map((p) => (
            <li key={p.id}>
              <button
                onClick={() => onSelect(p.id)}
                className={`w-full truncate px-3 py-1.5 text-left text-xs hover:bg-border/40 ${
                  p.id === activeId
                    ? 'bg-border/60 font-semibold text-fg'
                    : 'text-fg'
                }`}
                title={p.repo_local}
              >
                {p.name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
