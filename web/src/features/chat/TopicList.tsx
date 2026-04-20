import { useState } from 'react';
import {
  useAgentMembers,
  useCreateTopic,
  useTopics,
} from '../../api/hooks';
import type { ChatTopic } from '../../api/types';

type Props = {
  projectId: string;
  onOpenTopic: (id: string) => void;
};

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

export function TopicList({ projectId, onOpenTopic }: Props) {
  const { data: topics, isLoading } = useTopics(projectId);
  const [showNew, setShowNew] = useState(false);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b border-border bg-canvas-subtle px-4 py-2">
        <h2 className="text-sm font-semibold text-fg">Chat</h2>
        <div className="flex-1" />
        <button
          onClick={() => setShowNew((v) => !v)}
          className="rounded-md border border-border bg-accent px-3 py-1 text-xs font-semibold text-canvas hover:opacity-90"
        >
          New Chat
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="space-y-3 p-4">
          {showNew ? (
            <NewChatForm
              projectId={projectId}
              onCreated={(id) => {
                setShowNew(false);
                onOpenTopic(id);
              }}
              onCancel={() => setShowNew(false)}
            />
          ) : null}

          {isLoading ? (
            <p className="text-xs text-muted">loading...</p>
          ) : !topics || topics.length === 0 ? (
            <p className="text-xs text-muted">
              No chat topics yet.{' '}
              <button
                className="underline"
                onClick={() => setShowNew(true)}
              >
                Start one
              </button>
              .
            </p>
          ) : (
            <ul className="divide-y divide-border rounded-md border border-border bg-canvas">
              {topics.map((t) => (
                <TopicRow key={t.id} topic={t} onOpen={onOpenTopic} />
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}

function TopicRow({
  topic,
  onOpen,
}: {
  topic: ChatTopic;
  onOpen: (id: string) => void;
}) {
  return (
    <li
      onClick={() => onOpen(topic.id)}
      className="flex cursor-pointer items-center gap-3 px-3 py-2 text-xs hover:bg-border/20"
    >
      <span
        className={`inline-block h-2 w-2 flex-shrink-0 rounded-full ${
          topic.status === 'open' ? 'bg-success' : 'bg-muted'
        }`}
        title={topic.status}
      />
      <span className="min-w-0 flex-1 truncate font-semibold">
        {topic.title}
      </span>
      <span className="flex-shrink-0 rounded bg-canvas-subtle px-1.5 py-0.5 text-[10px] text-muted">
        {topic.plugin}
      </span>
      <span className="flex-shrink-0 text-[10px] text-muted">
        {relativeTime(topic.updated_at)}
      </span>
    </li>
  );
}

function NewChatForm({
  projectId,
  onCreated,
  onCancel,
}: {
  projectId: string;
  onCreated: (id: string) => void;
  onCancel: () => void;
}) {
  const [title, setTitle] = useState('');
  const [plugin, setPlugin] = useState('');
  const { data: agents } = useAgentMembers('chat');
  const create = useCreateTopic(projectId);

  // Auto-select first agent when agents load
  const agentList = agents ?? [];
  const selectedPlugin = plugin || agentList[0]?.name || '';

  return (
    <div className="rounded-md border border-border bg-canvas p-3">
      <div className="space-y-2">
        <input
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="Chat title..."
          className="w-full rounded-md border border-border bg-canvas px-2 py-1 text-xs outline-none focus:border-accent"
          autoFocus
        />
        <select
          value={selectedPlugin}
          onChange={(e) => setPlugin(e.target.value)}
          className="w-full rounded-md border border-border bg-canvas px-2 py-1 text-xs"
        >
          {agentList.length === 0 ? (
            <option value="">no chat agents available</option>
          ) : (
            agentList.map((a) => (
              <option key={a.name} value={a.name}>
                {a.name}
              </option>
            ))
          )}
        </select>
        <div className="flex gap-2">
          <button
            onClick={() => {
              if (!title.trim() || !selectedPlugin) return;
              create.mutate(
                { title: title.trim(), plugin: selectedPlugin },
                { onSuccess: (t) => onCreated(t.id) },
              );
            }}
            disabled={!title.trim() || !selectedPlugin || create.isPending}
            className="rounded-md border border-border bg-accent px-3 py-1 text-xs font-semibold text-canvas hover:opacity-90 disabled:opacity-50"
          >
            {create.isPending ? 'Creating...' : 'Create'}
          </button>
          <button
            onClick={onCancel}
            className="rounded-md border border-border px-3 py-1 text-xs text-muted hover:text-fg"
          >
            Cancel
          </button>
        </div>
        {create.isError ? (
          <p className="text-xs text-danger">{String(create.error)}</p>
        ) : null}
      </div>
    </div>
  );
}
