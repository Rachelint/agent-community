export type Project = {
  id: string;
  name: string;
  repo_url?: string;
  repo_local: string;
  default_branch: string;
  created_at: number;
  updated_at: number;
};

export type AgentMember = {
  name: string;
  kind: 'chat' | 'worker';
  manifest_path: string;
  enabled: boolean;
  registered_at: number;
  updated_at: number;
};

export type Label = {
  id: string;
  project_id: string;
  name: string;
  color: string;
  description?: string;
  created_at: number;
  updated_at: number;
};

export type Issue = {
  id: string;
  project_id: string;
  number: number;
  parent_id?: string | null;
  title: string;
  body: string;
  status: 'open' | 'closed';
  assignee?: string | null;
  created_at: number;
  updated_at: number;
  closed_at?: number | null;
  labels: Label[];
  child_count: number;
};

export type IssueComment = {
  id: string;
  issue_id: string;
  author: string;
  body: string;
  created_at: number;
};

export type IssueFilter = {
  status?: 'open' | 'closed' | '';
  assignee?: string;
  label?: string;
  parent?: string; // "" | "null" | issue id
  q?: string;
};

export type RunStatus =
  | 'queued'
  | 'running'
  | 'needs_review'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'orphan';

export type WorkerRun = {
  id: string;
  issue_id: string;
  project_id: string;
  plugin: string;
  status: RunStatus;
  pid?: number | null;
  workspace_dir: string;
  started_at?: number | null;
  finished_at?: number | null;
  exit_code?: number | null;
  mr_url?: string;
  summary?: string;
  created_at: number;
};

export type LogChunk = {
  stream: 'stdout' | 'stderr' | 'events';
  from: number;
  next: number;
  chunk: string;
};

export type Notification = {
  id: string;
  project_id: string;
  kind: string;
  issue_id?: string | null;
  run_id?: string | null;
  title: string;
  body?: string;
  read_at?: number | null;
  archived_at?: number | null;
  created_at: number;
};
