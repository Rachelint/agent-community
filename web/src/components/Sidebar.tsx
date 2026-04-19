import { ProjectSwitcher } from '../features/projects/ProjectSwitcher';

export type Section = 'issues' | 'chat' | 'mailbox';

type Props = {
  activeProjectId: string | null;
  onSelectProject: (id: string) => void;
  activeSection: Section;
  onSelectSection: (s: Section) => void;
};

// Left column of the app shell. The section nav only becomes active
// once a project is selected.
export function Sidebar({
  activeProjectId,
  onSelectProject,
  activeSection,
  onSelectSection,
}: Props) {
  return (
    <aside className="flex h-full w-64 flex-col border-r border-border bg-canvas-subtle">
      <div className="flex items-center gap-2 border-b border-border px-3 py-3">
        <div className="h-5 w-5 rounded bg-fg" />
        <div className="flex-1">
          <h1 className="text-sm font-semibold leading-none">agent-community</h1>
          <p className="mt-0.5 text-[10px] text-muted">phase 2 · issues</p>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        <ProjectSwitcher
          activeId={activeProjectId}
          onSelect={onSelectProject}
        />
        {activeProjectId ? (
          <nav className="mt-2 border-t border-border py-2">
            <SectionButton
              label="Issues"
              active={activeSection === 'issues'}
              onClick={() => onSelectSection('issues')}
            />
            <SectionButton
              label="Chat"
              active={activeSection === 'chat'}
              onClick={() => onSelectSection('chat')}
              disabled
              hint="phase 5"
            />
            <SectionButton
              label="Mailbox"
              active={activeSection === 'mailbox'}
              onClick={() => onSelectSection('mailbox')}
              disabled
              hint="phase 4"
            />
          </nav>
        ) : null}
      </div>
    </aside>
  );
}

function SectionButton({
  label,
  active,
  onClick,
  disabled,
  hint,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
  disabled?: boolean;
  hint?: string;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={`flex w-full items-center justify-between px-3 py-1.5 text-xs ${
        disabled
          ? 'cursor-not-allowed text-muted'
          : active
            ? 'bg-border/60 font-semibold text-fg'
            : 'text-fg hover:bg-border/40'
      }`}
    >
      <span>{label}</span>
      {hint ? (
        <span className="text-[10px] font-normal text-muted">{hint}</span>
      ) : null}
    </button>
  );
}
