import { useAgentMembers, useDeleteProject, useProjects } from '../../api/hooks';

// Placeholder content for the right-hand pane. Shows the selected
// project's details and the list of registered agent members, so we can
// confirm end-to-end that the data layer works.
export function ProjectDetail({ projectId }: { projectId: string | null }) {
  const { data: projects } = useProjects();
  const { data: agents } = useAgentMembers();
  const del = useDeleteProject();
  const project = projects?.find((p) => p.id === projectId) ?? null;

  if (!project) {
    return (
      <div className="flex h-full items-center justify-center text-sm text-muted">
        select a project or create one from the sidebar
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <header className="flex items-start justify-between">
        <div>
          <h2 className="text-lg font-semibold">{project.name}</h2>
          <p className="font-mono text-xs text-muted">{project.repo_local}</p>
        </div>
        <button
          onClick={() => {
            if (confirm(`Delete project ${project.name}?`)) {
              del.mutate(project.id);
            }
          }}
          className="rounded-md border border-border px-2 py-1 text-xs text-muted hover:border-danger hover:text-danger"
        >
          delete
        </button>
      </header>

      <section className="space-y-2">
        <h3 className="text-[11px] font-semibold uppercase tracking-wide text-muted">
          Metadata
        </h3>
        <dl className="grid grid-cols-[120px_1fr] gap-x-3 gap-y-1 text-xs">
          <dt className="text-muted">id</dt>
          <dd className="font-mono">{project.id}</dd>
          <dt className="text-muted">default branch</dt>
          <dd className="font-mono">{project.default_branch}</dd>
          {project.repo_url ? (
            <>
              <dt className="text-muted">repo url</dt>
              <dd className="font-mono">{project.repo_url}</dd>
            </>
          ) : null}
          <dt className="text-muted">created</dt>
          <dd>{new Date(project.created_at).toLocaleString()}</dd>
        </dl>
      </section>

      <section className="space-y-2">
        <h3 className="text-[11px] font-semibold uppercase tracking-wide text-muted">
          Registered agents
        </h3>
        {!agents || agents.length === 0 ? (
          <p className="text-xs text-muted">none</p>
        ) : (
          <ul className="divide-y divide-border rounded-md border border-border bg-canvas">
            {agents.map((a) => (
              <li
                key={a.name}
                className="flex items-center justify-between px-3 py-2 text-xs"
              >
                <div className="flex items-center gap-2">
                  <span className="rounded bg-canvas-subtle px-1.5 py-0.5 font-mono text-[10px] uppercase text-muted">
                    {a.kind}
                  </span>
                  <span className="font-semibold">{a.name}</span>
                </div>
                <span className="font-mono text-[10px] text-muted">
                  {a.manifest_path}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
