import { useState } from 'react';
import {
  useArchiveNotification,
  useMarkNotificationRead,
  useNotifications,
} from '../../api/hooks';
import type { Notification } from '../../api/types';

function relativeTime(epochMs: number): string {
  const diff = Date.now() - epochMs;
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function KindIcon({ kind }: { kind: string }) {
  if (kind.includes('needs_review')) {
    return (
      <svg
        className="h-4 w-4 shrink-0 text-yellow-500"
        viewBox="0 0 16 16"
        fill="currentColor"
      >
        <circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" strokeWidth="1.5" />
        <circle cx="8" cy="8" r="3" />
      </svg>
    );
  }
  if (kind.includes('completed')) {
    return (
      <svg
        className="h-4 w-4 shrink-0 text-success"
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
      >
        <circle cx="8" cy="8" r="7" />
        <path d="M5 8l2 2 4-4" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    );
  }
  if (kind.includes('failed')) {
    return (
      <svg
        className="h-4 w-4 shrink-0 text-danger"
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
      >
        <circle cx="8" cy="8" r="7" />
        <path d="M5.5 5.5l5 5M10.5 5.5l-5 5" strokeLinecap="round" />
      </svg>
    );
  }
  return (
    <svg
      className="h-4 w-4 shrink-0 text-muted"
      viewBox="0 0 16 16"
      fill="currentColor"
    >
      <circle cx="8" cy="8" r="4" />
    </svg>
  );
}

function NotificationRow({
  notification,
  projectId,
  onSelect,
}: {
  notification: Notification;
  projectId: string;
  onSelect: (issueId: string) => void;
}) {
  const markRead = useMarkNotificationRead(projectId);
  const archive = useArchiveNotification(projectId);
  const isUnread = !notification.read_at;

  return (
    <div
      className={`flex items-start gap-3 border-b border-border px-4 py-3 ${
        isUnread ? 'bg-canvas' : 'bg-canvas-subtle'
      } hover:bg-border/30 cursor-pointer`}
      onClick={() => {
        if (notification.issue_id) {
          onSelect(notification.issue_id);
        }
      }}
    >
      <div className="mt-0.5">
        <KindIcon kind={notification.kind} />
      </div>
      <div className="min-w-0 flex-1">
        <p className={`text-xs ${isUnread ? 'font-semibold text-fg' : 'text-fg'}`}>
          {notification.title}
        </p>
        {notification.body ? (
          <p className="mt-0.5 truncate text-xs text-muted">{notification.body}</p>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="text-[10px] text-muted">
          {relativeTime(notification.created_at)}
        </span>
        {isUnread ? (
          <button
            type="button"
            className="rounded px-1.5 py-0.5 text-[10px] text-accent hover:bg-border/40"
            onClick={(e) => {
              e.stopPropagation();
              markRead.mutate(notification.id);
            }}
          >
            Mark read
          </button>
        ) : null}
        <button
          type="button"
          className="rounded px-1.5 py-0.5 text-[10px] text-muted hover:bg-border/40"
          onClick={(e) => {
            e.stopPropagation();
            archive.mutate(notification.id);
          }}
        >
          Archive
        </button>
      </div>
    </div>
  );
}

export function MailboxSection({
  projectId,
  onSelectIssue,
}: {
  projectId: string;
  onSelectIssue: (issueId: string) => void;
}) {
  const [tab, setTab] = useState<'unread' | 'all'>('unread');
  const { data: notifications, isLoading } = useNotifications(
    projectId,
    tab === 'unread',
  );

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-4 border-b border-border px-4 py-2">
        <h2 className="text-sm font-semibold text-fg">Mailbox</h2>
        <div className="flex gap-1">
          <TabButton
            label="Unread"
            active={tab === 'unread'}
            onClick={() => setTab('unread')}
          />
          <TabButton
            label="All"
            active={tab === 'all'}
            onClick={() => setTab('all')}
          />
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        {isLoading ? (
          <p className="px-4 py-8 text-center text-xs text-muted">Loading...</p>
        ) : !notifications?.length ? (
          <p className="px-4 py-8 text-center text-xs text-muted">
            No notifications
          </p>
        ) : (
          notifications.map((n) => (
            <NotificationRow
              key={n.id}
              notification={n}
              projectId={projectId}
              onSelect={onSelectIssue}
            />
          ))
        )}
      </div>
    </div>
  );
}

function TabButton({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-md px-2 py-0.5 text-xs ${
        active
          ? 'bg-border/60 font-semibold text-fg'
          : 'text-muted hover:text-fg'
      }`}
    >
      {label}
    </button>
  );
}
