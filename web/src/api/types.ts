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
