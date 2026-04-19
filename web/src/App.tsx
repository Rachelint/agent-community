import { useEffect, useState } from 'react';

type Health = { status: string } | null;

export default function App() {
  const [health, setHealth] = useState<Health>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/healthz')
      .then((r) => r.json())
      .then((data) => setHealth(data))
      .catch((e) => setError(String(e)));
  }, []);

  return (
    <div className="min-h-full flex flex-col">
      <header className="border-b border-border bg-canvas-subtle px-6 py-3">
        <div className="flex items-center gap-2">
          <div className="h-6 w-6 rounded bg-fg" />
          <h1 className="text-base font-semibold">agent-community</h1>
          <span className="ml-2 text-xs text-muted">phase 0 · skeleton</span>
        </div>
      </header>
      <main className="flex-1 p-8">
        <div className="mx-auto max-w-2xl space-y-6">
          <section className="rounded-md border border-border bg-canvas p-4">
            <h2 className="mb-2 text-sm font-semibold">Server</h2>
            {health ? (
              <p className="font-mono text-xs text-success">
                /healthz → {JSON.stringify(health)}
              </p>
            ) : error ? (
              <p className="font-mono text-xs text-danger">{error}</p>
            ) : (
              <p className="font-mono text-xs text-muted">checking…</p>
            )}
          </section>

          <section className="rounded-md border border-border bg-canvas p-4">
            <h2 className="mb-2 text-sm font-semibold">Up next</h2>
            <ul className="list-disc space-y-1 pl-5 text-muted">
              <li>Phase 1 — projects + data layer</li>
              <li>Phase 2 — issues</li>
              <li>Phase 3 — plugin system + worker dispatch</li>
            </ul>
          </section>
        </div>
      </main>
    </div>
  );
}
