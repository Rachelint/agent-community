import { useNotificationCount } from '../api/hooks';
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
  const { data: countData } = useNotificationCount(activeProjectId);
  const unreadCount = countData?.count ?? 0;
  return (
    <aside className="flex h-full w-64 flex-col border-r border-border bg-canvas-subtle">
      <div className="flex items-center gap-2 border-b border-border px-3 py-3">
        <div className="h-5 w-5 rounded bg-fg" />
        <div className="flex-1">
          <h1 className="text-sm font-semibold leading-none">agent-community</h1>
          <p className="mt-0.5 text-[10px] text-muted">phase 6</p>
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
            />
            <SectionButton
              label="Mailbox"
              active={activeSection === 'mailbox'}
              onClick={() => onSelectSection('mailbox')}
              badge={unreadCount > 0 ? unreadCount : undefined}
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
  badge,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
  disabled?: boolean;
  hint?: string;
  badge?: number;
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
      <span className="flex items-center gap-1.5">
        {badge ? (
          <span className="inline-flex h-4 min-w-[16px] items-center justify-center rounded-full bg-accent px-1 text-[10px] font-semibold leading-none text-white">
            {badge > 99 ? '99+' : badge}
          </span>
        ) : null}
        {hint ? (
          <span className="text-[10px] font-normal text-muted">{hint}</span>
        ) : null}
      </span>
    </button>
  );
}
