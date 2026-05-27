import { useState } from 'react';
import { useDeleteProject, useProjects } from '../../api/hooks';
import type { Project } from '../../api/types';
import { NewProjectForm } from './NewProjectForm';

type Props = {
  activeId: string | null;
  onSelect: (id: string | null) => void;
};

export function ProjectSwitcher({ activeId, onSelect }: Props) {
  const { data, isLoading, error } = useProjects();
  const del = useDeleteProject();
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
            <li
              key={p.id}
              className={`group flex items-center hover:bg-border/40 ${
                p.id === activeId ? 'bg-border/60 font-semibold text-fg' : 'text-fg'
              }`}
            >
              <button
                onClick={() => onSelect(p.id)}
                className="min-w-0 flex-1 truncate px-3 py-1.5 text-left text-xs"
                title={p.repo_local}
              >
                {p.name}
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  if (confirm(`Delete project ${p.name}?`)) {
                    del.mutate(p.id, {
                      onSuccess: () => {
                        if (p.id === activeId) onSelect(null);
                      },
                    });
                  }
                }}
                className="mr-2 hidden rounded px-1 py-0.5 text-[10px] text-muted hover:bg-danger hover:text-canvas group-hover:inline-block"
                title="delete project"
              >
                delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
