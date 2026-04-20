import { useEffect, useState } from 'react';
import { Sidebar, type Section } from './components/Sidebar';
import { ProjectDetail } from './features/projects/ProjectDetail';
import { IssuesSection } from './features/issues/IssuesSection';
import { ChatSection } from './features/chat/ChatSection';
import { MailboxSection } from './features/mailbox/MailboxSection';

export default function App() {
  const [activeProjectId, setActiveProjectId] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<Section>('issues');
  const [selectedIssueId, setSelectedIssueId] = useState<string | null>(null);

  // When switching projects, snap back to the issues view.
  useEffect(() => {
    setActiveSection('issues');
  }, [activeProjectId]);

  return (
    <div className="flex h-full">
      <Sidebar
        activeProjectId={activeProjectId}
        onSelectProject={setActiveProjectId}
        activeSection={activeSection}
        onSelectSection={setActiveSection}
      />
      <main className="flex-1 overflow-hidden bg-canvas">
        {!activeProjectId ? (
          <ProjectDetail projectId={null} />
        ) : activeSection === 'issues' ? (
          <IssuesSection
            projectId={activeProjectId}
            initialIssueId={selectedIssueId}
            onConsumeInitialIssue={() => setSelectedIssueId(null)}
          />
        ) : activeSection === 'chat' ? (
          <ChatSection projectId={activeProjectId} />
        ) : activeSection === 'mailbox' ? (
          <MailboxSection
            projectId={activeProjectId}
            onSelectIssue={(issueId) => {
              setSelectedIssueId(issueId);
              setActiveSection('issues');
            }}
          />
        ) : (
          <ProjectDetail projectId={activeProjectId} />
        )}
      </main>
    </div>
  );
}
