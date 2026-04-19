import { useState } from 'react';
import type { Project } from '../../api/types';
import { useCreateProject } from '../../api/hooks';

// Inline form for creating a project. Intentionally plain — shadcn
// components arrive with phase 2 forms work.
export function NewProjectForm({ onCreated }: { onCreated: (p: Project) => void }) {
  const [name, setName] = useState('');
  const [repoLocal, setRepoLocal] = useState('');
  const [repoUrl, setRepoUrl] = useState('');
  const [branch, setBranch] = useState('');
  const create = useCreateProject();

  return (
    <form
      className="space-y-2 p-3"
      onSubmit={(e) => {
        e.preventDefault();
        create.mutate(
          {
            name: name.trim(),
            repo_local: repoLocal.trim(),
            repo_url: repoUrl.trim() || undefined,
            default_branch: branch.trim() || undefined,
          },
          {
            onSuccess: (p) => {
              setName('');
              setRepoLocal('');
              setRepoUrl('');
              setBranch('');
              onCreated(p);
            },
          },
        );
      }}
    >
      <Input label="Name" value={name} onChange={setName} required />
      <Input
        label="Repo local path"
        value={repoLocal}
        onChange={setRepoLocal}
        placeholder="/absolute/path/to/clone"
        required
      />
      <Input
        label="Repo URL (optional)"
        value={repoUrl}
        onChange={setRepoUrl}
      />
      <Input
        label="Default branch"
        value={branch}
        onChange={setBranch}
        placeholder="master"
      />
      {create.error ? (
        <p className="text-xs text-danger">{(create.error as Error).message}</p>
      ) : null}
      <button
        type="submit"
        disabled={create.isPending}
        className="w-full rounded-md border border-border bg-canvas-subtle px-3 py-1.5 text-xs font-semibold hover:bg-border/30 disabled:opacity-60"
      >
        {create.isPending ? 'Creating…' : 'Create project'}
      </button>
    </form>
  );
}

function Input({
  label,
  value,
  onChange,
  placeholder,
  required,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  required?: boolean;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-[11px] font-medium text-muted">{label}</span>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        required={required}
        className="w-full rounded-md border border-border bg-canvas px-2 py-1 text-xs outline-none focus:border-accent"
      />
    </label>
  );
}
