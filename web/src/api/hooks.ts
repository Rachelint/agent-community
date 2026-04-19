import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from './client';
import type { AgentMember, Project } from './types';

export const qk = {
  projects: ['projects'] as const,
  project: (id: string) => ['projects', id] as const,
  agentMembers: (kind?: string) =>
    ['agent_members', { kind: kind ?? null }] as const,
};

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

export function useAgentMembers(kind?: 'chat' | 'worker') {
  const query = kind ? `?kind=${kind}` : '';
  return useQuery({
    queryKey: qk.agentMembers(kind),
    queryFn: () => apiFetch<AgentMember[]>(`/api/agent_members${query}`),
  });
}
