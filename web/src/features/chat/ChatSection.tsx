import { useState } from 'react';
import { ChatPane } from './ChatPane';
import { TopicList } from './TopicList';

type View = { kind: 'list' } | { kind: 'chat'; id: string };

export function ChatSection({ projectId }: { projectId: string }) {
  const [view, setView] = useState<View>({ kind: 'list' });

  if (view.kind === 'chat') {
    return (
      <ChatPane
        topicId={view.id}
        onBack={() => setView({ kind: 'list' })}
      />
    );
  }

  return (
    <TopicList
      projectId={projectId}
      onOpenTopic={(id) => setView({ kind: 'chat', id })}
    />
  );
}
