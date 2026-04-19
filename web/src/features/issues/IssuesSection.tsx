import { useEffect, useState } from 'react';
import { RunDetail } from '../runs/RunDetail';
import { IssueDetail } from './IssueDetail';
import { IssuesList } from './IssuesList';

type View =
  | { kind: 'list' }
  | { kind: 'issue'; id: string }
  | { kind: 'run'; id: string; fromIssueId: string };

// Phase-3 issues section. Routes between list / issue detail / run
// detail purely via local state. URL routing comes in a later phase.
export function IssuesSection({
  projectId,
  initialIssueId,
  onConsumeInitialIssue,
}: {
  projectId: string;
  initialIssueId?: string | null;
  onConsumeInitialIssue?: () => void;
}) {
  const [view, setView] = useState<View>({ kind: 'list' });

  useEffect(() => {
    if (initialIssueId) {
      setView({ kind: 'issue', id: initialIssueId });
      onConsumeInitialIssue?.();
    }
  }, [initialIssueId, onConsumeInitialIssue]);

  if (view.kind === 'issue') {
    return (
      <IssueDetail
        issueId={view.id}
        onBack={() => setView({ kind: 'list' })}
        onOpenIssue={(id) => setView({ kind: 'issue', id })}
        onOpenRun={(runId) =>
          setView({ kind: 'run', id: runId, fromIssueId: view.id })
        }
      />
    );
  }
  if (view.kind === 'run') {
    return (
      <RunDetail
        runId={view.id}
        onBack={() => setView({ kind: 'issue', id: view.fromIssueId })}
      />
    );
  }
  return (
    <IssuesList
      projectId={projectId}
      onOpenIssue={(id) => setView({ kind: 'issue', id })}
    />
  );
}
