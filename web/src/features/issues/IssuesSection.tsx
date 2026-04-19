import { useState } from 'react';
import { IssueDetail } from './IssueDetail';
import { IssuesList } from './IssuesList';

// Phase-2 issues section. Keeps its own state for "which issue is open",
// driven by local state rather than URL — we'll graduate to real router
// URLs in a later phase.
export function IssuesSection({ projectId }: { projectId: string }) {
  const [openIssueId, setOpenIssueId] = useState<string | null>(null);

  if (openIssueId) {
    return (
      <IssueDetail
        issueId={openIssueId}
        onBack={() => setOpenIssueId(null)}
        onOpenIssue={(id) => setOpenIssueId(id)}
      />
    );
  }

  return (
    <IssuesList
      projectId={projectId}
      onOpenIssue={(id) => setOpenIssueId(id)}
    />
  );
}
