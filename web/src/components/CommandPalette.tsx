import { useEffect, useRef, useState } from 'react';
import { useIssues, useTopics } from '../api/hooks';

type Result = {
  kind: 'issue' | 'topic';
  id: string;
  label: string;
  detail?: string;
};

type Props = {
  open: boolean;
  onClose: () => void;
  projectId: string | null;
  onSelectIssue: (issueId: string) => void;
  onSelectTopic: (topicId: string) => void;
};

export function CommandPalette({
  open,
  onClose,
  projectId,
  onSelectIssue,
  onSelectTopic,
}: Props) {
  const [query, setQuery] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);

  const { data: issues } = useIssues(open ? projectId : null, { q: query || undefined });
  const { data: topics } = useTopics(open ? projectId : null);

  // Reset query and focus input when opened.
  useEffect(() => {
    if (open) {
      setQuery('');
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open]);

  // Close on Escape.
  useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      }
    }
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [open, onClose]);

  if (!open) return null;

  const lowerQ = query.toLowerCase();

  const results: Result[] = [];

  // Issues
  if (issues) {
    for (const iss of issues.slice(0, 10)) {
      results.push({
        kind: 'issue',
        id: iss.id,
        label: `#${iss.number} ${iss.title}`,
        detail: iss.status,
      });
    }
  }

  // Topics (client-side filter since the API doesn't support q)
  if (topics) {
    for (const t of topics) {
      if (lowerQ && !t.title.toLowerCase().includes(lowerQ)) continue;
      results.push({
        kind: 'topic',
        id: t.id,
        label: t.title,
        detail: t.status,
      });
      if (results.filter((r) => r.kind === 'topic').length >= 10) break;
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/40 pt-[15vh]"
      onClick={onClose}
    >
      <div
        className="w-full max-w-md rounded-lg border border-border bg-canvas shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <input
          ref={inputRef}
          type="text"
          placeholder="Search issues and topics..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="w-full border-b border-border bg-transparent px-4 py-3 text-sm outline-none placeholder:text-muted"
        />
        <div className="max-h-72 overflow-y-auto">
          {results.length === 0 ? (
            <p className="px-4 py-6 text-center text-xs text-muted">
              {projectId ? 'No results' : 'Select a project first'}
            </p>
          ) : (
            <ul>
              {results.map((r) => (
                <li key={`${r.kind}-${r.id}`}>
                  <button
                    type="button"
                    className="flex w-full items-center gap-2 px-4 py-2 text-left text-xs hover:bg-canvas-subtle"
                    onClick={() => {
                      if (r.kind === 'issue') onSelectIssue(r.id);
                      else onSelectTopic(r.id);
                      onClose();
                    }}
                  >
                    <span
                      className={`inline-block rounded px-1 py-0.5 text-[10px] font-semibold leading-none ${
                        r.kind === 'issue'
                          ? 'bg-accent/10 text-accent'
                          : 'bg-success/10 text-success'
                      }`}
                    >
                      {r.kind}
                    </span>
                    <span className="flex-1 truncate">{r.label}</span>
                    {r.detail && (
                      <span className="text-[10px] text-muted">{r.detail}</span>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="border-t border-border px-4 py-2 text-[10px] text-muted">
          <kbd className="rounded border border-border bg-canvas-subtle px-1">esc</kbd>{' '}
          to close
        </div>
      </div>
    </div>
  );
}
