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
