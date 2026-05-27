import { useEffect, useRef, useState } from 'react';
import {
  useCloseTopic,
  useMessages,
  useRestartTopic,
  useSendMessage,
  useTopic,
} from '../../api/hooks';
import type { ChatMessage } from '../../api/types';
import { Markdown } from '../../components/Markdown';

type Props = {
  topicId: string;
  onBack: () => void;
};

export function ChatPane({ topicId, onBack }: Props) {
  const { data: topic } = useTopic(topicId);
  const { data: messages } = useMessages(topicId, topic?.status);
  const sendMsg = useSendMessage(topicId);
  const closeTopic = useCloseTopic();
  const restartTopic = useRestartTopic();

  const [input, setInput] = useState('');
  const scrollRef = useRef<HTMLDivElement>(null);
  const prevCountRef = useRef(0);

  // Auto-scroll when new messages arrive
  useEffect(() => {
    const count = messages?.length ?? 0;
    if (count > prevCountRef.current) {
      const el = scrollRef.current;
      if (el) {
        requestAnimationFrame(() => {
          el.scrollTop = el.scrollHeight;
        });
      }
    }
    prevCountRef.current = count;
  }, [messages?.length]);

  const isOpen = topic?.status === 'open';
  const agentAlive = isOpen && topic?.pid != null;
  const agentDead = isOpen && topic?.pid == null;

  // Detect "thinking" state: last message is from user and no reply yet
  const lastMsg = messages?.[messages.length - 1];
  const isThinking = isOpen && agentAlive && lastMsg?.role === 'user';

  const handleSend = () => {
    const text = input.trim();
    if (!text || !isOpen) return;
    sendMsg.mutate({ content: text });
    setInput('');
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (!topic) {
    return (
      <div className="p-6 text-xs text-muted">
        <button onClick={onBack} className="underline">
          &larr; topics
        </button>
        <p className="mt-4">loading...</p>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-border bg-canvas-subtle px-4 py-2 text-xs">
        <button
          onClick={onBack}
          className="text-muted hover:text-fg"
        >
          &larr; topics
        </button>
        <span className="text-muted">/</span>
        <span className="font-semibold">{topic.title}</span>
        <span
          className={`rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase ${
            isOpen
              ? 'bg-[#dcfce7] text-success'
              : 'bg-canvas-subtle text-muted'
          }`}
        >
          {topic.status}
        </span>
        {isOpen ? (
          <span
            className={`inline-block h-2 w-2 rounded-full ${
              agentAlive ? 'bg-success' : 'bg-danger'
            }`}
            title={agentAlive ? 'Agent running' : 'Agent offline'}
          />
        ) : null}
        <div className="flex-1" />
        {agentDead ? (
          <button
            onClick={() => restartTopic.mutate(topicId)}
            disabled={restartTopic.isPending}
            className="rounded-md border border-border px-2 py-1 text-xs text-accent hover:bg-border/40"
          >
            {restartTopic.isPending ? 'Restarting...' : 'Restart'}
          </button>
        ) : null}
        {isOpen ? (
          <button
            onClick={() => {
              if (confirm('Close this topic? The agent will be terminated.')) {
                closeTopic.mutate(topicId);
              }
            }}
            disabled={closeTopic.isPending}
            className="rounded-md border border-border px-2 py-1 text-xs text-danger hover:bg-danger hover:text-canvas"
          >
            Close
          </button>
        ) : null}
      </div>

      {/* Messages */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-4">
        {!messages?.length ? (
          <p className="py-8 text-center text-xs text-muted">
            Send a message to start the conversation
          </p>
        ) : (
          <div className="space-y-3">
            {messages.map((msg) => (
              <MessageBubble key={msg.id} message={msg} />
            ))}
          </div>
        )}
        {isThinking ? (
          <div className="mt-3 flex items-center gap-2 text-xs text-muted">
            <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-accent" />
            Agent is thinking...
          </div>
        ) : null}
      </div>

      {/* Input */}
      <div className="border-t border-border bg-canvas-subtle px-4 py-3">
        {agentDead ? (
          <p className="mb-2 text-xs text-danger">
            Agent is offline.{' '}
            <button
              className="underline"
              onClick={() => restartTopic.mutate(topicId)}
            >
              Restart
            </button>{' '}
            to continue.
          </p>
        ) : null}
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={isOpen ? 'Type a message... (Enter to send, Shift+Enter for newline)' : 'Topic is closed'}
            disabled={!isOpen || agentDead}
            rows={1}
            className="max-h-36 min-h-[32px] flex-1 resize-none rounded-md border border-border bg-canvas px-2 py-1.5 text-xs outline-none focus:border-accent disabled:opacity-50"
            style={{ overflow: 'auto' }}
            onInput={(e) => {
              const el = e.currentTarget;
              el.style.height = 'auto';
              el.style.height = `${Math.min(el.scrollHeight, 144)}px`;
            }}
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || !isOpen || agentDead || sendMsg.isPending}
            className="self-end rounded-md border border-border bg-accent px-3 py-1.5 text-xs font-semibold text-canvas hover:opacity-90 disabled:opacity-50"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
}

function MessageBubble({ message }: { message: ChatMessage }) {
  if (message.role === 'system') {
    return (
      <div className="text-center text-xs italic text-muted">
        {message.content}
      </div>
    );
  }

  if (message.role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[70%] rounded-lg bg-accent px-3 py-2 text-xs text-white">
          <p className="whitespace-pre-wrap">{message.content}</p>
        </div>
      </div>
    );
  }

  // assistant
  return (
    <div className="flex justify-start">
      <div className="max-w-[70%] rounded-lg bg-canvas-subtle px-3 py-2 text-xs">
        <Markdown source={message.content} />
      </div>
    </div>
  );
}
