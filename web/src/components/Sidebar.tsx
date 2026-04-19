import { ProjectSwitcher } from '../features/projects/ProjectSwitcher';

type Props = {
  activeProjectId: string | null;
  onSelectProject: (id: string) => void;
};

// Left column of the app shell. Phase 1 only surfaces the project
// switcher; later phases add Chat / Issues / Mailbox section nav.
export function Sidebar({ activeProjectId, onSelectProject }: Props) {
  return (
    <aside className="flex h-full w-64 flex-col border-r border-border bg-canvas-subtle">
      <div className="flex items-center gap-2 border-b border-border px-3 py-3">
        <div className="h-5 w-5 rounded bg-fg" />
        <div className="flex-1">
          <h1 className="text-sm font-semibold leading-none">agent-community</h1>
          <p className="mt-0.5 text-[10px] text-muted">phase 1 · data layer</p>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        <ProjectSwitcher
          activeId={activeProjectId}
          onSelect={onSelectProject}
        />
      </div>
    </aside>
  );
}
