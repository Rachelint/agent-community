import { useEffect, useState } from 'react';
import { Sidebar, type Section } from './components/Sidebar';
import { ProjectDetail } from './features/projects/ProjectDetail';
import { IssuesSection } from './features/issues/IssuesSection';

export default function App() {
  const [activeProjectId, setActiveProjectId] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<Section>('issues');

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
          <IssuesSection projectId={activeProjectId} />
        ) : (
          <ProjectDetail projectId={activeProjectId} />
        )}
      </main>
    </div>
  );
}
