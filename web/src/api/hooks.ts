import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from './client';
import type {
  AgentMember,
  Issue,
  IssueComment,
  IssueFilter,
  Label,
  Project,
} from './types';

export const qk = {
  projects: ['projects'] as const,
  project: (id: string) => ['projects', id] as const,
  agentMembers: (kind?: string) =>
    ['agent_members', { kind: kind ?? null }] as const,
  labels: (pid: string) => ['labels', pid] as const,
  issues: (pid: string, filter: IssueFilter) =>
    ['issues', pid, filter] as const,
  issue: (id: string) => ['issue', id] as const,
  issueComments: (id: string) => ['issue', id, 'comments'] as const,
};

// ---- projects -------------------------------------------------------------
export function useProjects() {
  return useQuery({
    queryKey: qk.projects,
    queryFn: () => apiFetch<Project[]>('/api/projects'),
  });
}

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      name: string;
      repo_local: string;
      repo_url?: string;
      default_branch?: string;
    }) =>
      apiFetch<Project>('/api/projects', {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.projects }),
  });
}

export function useDeleteProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/projects/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.projects }),
  });
}

// ---- agent members --------------------------------------------------------
export function useAgentMembers(kind?: 'chat' | 'worker') {
  const query = kind ? `?kind=${kind}` : '';
  return useQuery({
    queryKey: qk.agentMembers(kind),
    queryFn: () => apiFetch<AgentMember[]>(`/api/agent_members${query}`),
  });
}

// ---- labels ---------------------------------------------------------------
export function useLabels(projectId: string | null) {
  return useQuery({
    enabled: !!projectId,
    queryKey: qk.labels(projectId ?? ''),
    queryFn: () => apiFetch<Label[]>(`/api/projects/${projectId}/labels`),
  });
}

export function useCreateLabel(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { name: string; color: string; description?: string }) =>
      apiFetch<Label>(`/api/projects/${projectId}/labels`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.labels(projectId) }),
  });
}

export function useDeleteLabel(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/labels/${id}`, { method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.labels(projectId) }),
  });
}

// ---- issues ---------------------------------------------------------------
export function useIssues(projectId: string | null, filter: IssueFilter) {
  const params = new URLSearchParams();
  if (filter.status) params.set('status', filter.status);
  if (filter.assignee) params.set('assignee', filter.assignee);
  if (filter.label) params.set('label', filter.label);
  if (filter.parent !== undefined && filter.parent !== '')
    params.set('parent', filter.parent);
  if (filter.q) params.set('q', filter.q);
  const qs = params.toString();
  return useQuery({
    enabled: !!projectId,
    queryKey: qk.issues(projectId ?? '', filter),
    queryFn: () =>
      apiFetch<Issue[]>(
        `/api/projects/${projectId}/issues${qs ? '?' + qs : ''}`,
      ),
  });
}

export function useIssue(issueId: string | null) {
  return useQuery({
    enabled: !!issueId,
    queryKey: qk.issue(issueId ?? ''),
    queryFn: () => apiFetch<Issue>(`/api/issues/${issueId}`),
  });
}

export function useCreateIssue(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: {
      title: string;
      body?: string;
      parent_id?: string;
      assignee?: string;
      labels?: string[];
    }) =>
      apiFetch<Issue>(`/api/projects/${projectId}/issues`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['issues', projectId] });
    },
  });
}

export type IssuePatchInput = {
  title?: string;
  body?: string;
  status?: 'open' | 'closed';
  assignee?: string;
  parent_id?: string;
  labels?: string[];
};

export function usePatchIssue(issueId: string, projectId?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: IssuePatchInput) =>
      apiFetch<Issue>(`/api/issues/${issueId}`, {
        method: 'PATCH',
        body: JSON.stringify(input),
      }),
    onSuccess: (iss) => {
      qc.setQueryData(qk.issue(issueId), iss);
      if (projectId) {
        qc.invalidateQueries({ queryKey: ['issues', projectId] });
      }
    },
  });
}

export function useDeleteIssue(projectId?: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/issues/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      if (projectId) {
        qc.invalidateQueries({ queryKey: ['issues', projectId] });
      }
    },
  });
}

// ---- comments -------------------------------------------------------------
export function useIssueComments(issueId: string | null) {
  return useQuery({
    enabled: !!issueId,
    queryKey: qk.issueComments(issueId ?? ''),
    queryFn: () =>
      apiFetch<IssueComment[]>(`/api/issues/${issueId}/comments`),
  });
}

export function useCreateIssueComment(issueId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { body: string; author?: string }) =>
      apiFetch<IssueComment>(`/api/issues/${issueId}/comments`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.issueComments(issueId) }),
  });
}
