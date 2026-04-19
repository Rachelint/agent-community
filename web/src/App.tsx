import { useState } from 'react';
import { Sidebar } from './components/Sidebar';
import { ProjectDetail } from './features/projects/ProjectDetail';

export default function App() {
  const [activeProjectId, setActiveProjectId] = useState<string | null>(null);

  return (
    <div className="flex h-full">
      <Sidebar
        activeProjectId={activeProjectId}
        onSelectProject={setActiveProjectId}
      />
      <main className="flex-1 overflow-y-auto bg-canvas">
        <ProjectDetail projectId={activeProjectId} />
      </main>
    </div>
  );
}
