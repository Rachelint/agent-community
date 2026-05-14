import { useCallback, useEffect, useState } from 'react';
import { Sidebar, type Section } from './components/Sidebar';
import { ProjectDetail } from './features/projects/ProjectDetail';
import { IssuesSection } from './features/issues/IssuesSection';
import { ChatSection } from './features/chat/ChatSection';
import { MailboxSection } from './features/mailbox/MailboxSection';
import { ToastProvider } from './components/Toast';
import { CommandPalette } from './components/CommandPalette';
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts';

export default function App() {
  const [activeProjectId, setActiveProjectId] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<Section>('issues');
  const [selectedIssueId, setSelectedIssueId] = useState<string | null>(null);
  const [selectedRunId, setSelectedRunId] = useState<string | null>(null);
  const [paletteOpen, setPaletteOpen] = useState(false);

  // When switching projects, snap back to the issues view.
  useEffect(() => {
    setActiveSection('issues');
  }, [activeProjectId]);

  const openPalette = useCallback(() => setPaletteOpen(true), []);

  useKeyboardShortcuts({ onCommandPalette: openPalette });

  return (
    <ToastProvider>
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
              initialRunId={selectedRunId}
              onConsumeInitialNavigation={() => {
                setSelectedIssueId(null);
                setSelectedRunId(null);
              }}
            />
          ) : activeSection === 'chat' ? (
            <ChatSection projectId={activeProjectId} />
          ) : activeSection === 'mailbox' ? (
            <MailboxSection
              projectId={activeProjectId}
              onSelectIssue={(issueId) => {
                setSelectedIssueId(issueId);
                setSelectedRunId(null);
                setActiveSection('issues');
              }}
              onSelectRun={(issueId, runId) => {
                setSelectedIssueId(issueId);
                setSelectedRunId(runId);
                setActiveSection('issues');
              }}
            />
          ) : (
            <ProjectDetail projectId={activeProjectId} />
          )}
        </main>
      </div>
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        projectId={activeProjectId}
        onSelectIssue={(issueId) => {
          setSelectedIssueId(issueId);
          setSelectedRunId(null);
          setActiveSection('issues');
        }}
        onSelectTopic={() => {
          setActiveSection('chat');
        }}
      />
    </ToastProvider>
  );
}
