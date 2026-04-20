import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiFetch } from './client';
import type {
  AgentMember,
  ChatMessage,
  ChatTopic,
  Issue,
  IssueComment,
  IssueFilter,
  Label,
  LogChunk,
  Notification,
  Project,
  TopicStatus,
  WorkerRun,
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
  issueRuns: (id: string) => ['issue', id, 'runs'] as const,
  run: (id: string) => ['run', id] as const,
  notifications: (pid: string, unread?: boolean) =>
    ['notifications', pid, { unread: unread ?? false }] as const,
  notificationCount: (pid: string) => ['notification_count', pid] as const,
  topics: (pid: string) => ['topics', pid] as const,
  topic: (id: string) => ['topic', id] as const,
  messages: (topicId: string) => ['messages', topicId] as const,
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

// ---- runs -----------------------------------------------------------------
// Runs auto-refresh while any run is in a non-terminal state so the UI
// reflects worker progress without websocket support yet.
const RUN_TERMINAL = new Set([
  'completed',
  'failed',
  'cancelled',
  'needs_review',
  'orphan',
]);

export function useIssueRuns(issueId: string | null) {
  return useQuery({
    enabled: !!issueId,
    queryKey: qk.issueRuns(issueId ?? ''),
    queryFn: () => apiFetch<WorkerRun[]>(`/api/issues/${issueId}/runs`),
    refetchInterval: (q) => {
      const data = q.state.data as WorkerRun[] | undefined;
      if (!data) return 4000;
      const active = data.some((r) => !RUN_TERMINAL.has(r.status));
      return active ? 2000 : false;
    },
  });
}

export function useRun(runId: string | null) {
  return useQuery({
    enabled: !!runId,
    queryKey: qk.run(runId ?? ''),
    queryFn: () => apiFetch<WorkerRun>(`/api/runs/${runId}`),
    refetchInterval: (q) => {
      const data = q.state.data as WorkerRun | undefined;
      if (!data) return 2000;
      return RUN_TERMINAL.has(data.status) ? false : 2000;
    },
  });
}

export function useDispatch(issueId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { plugin: string; prompt?: string }) =>
      apiFetch<WorkerRun>(`/api/issues/${issueId}/dispatch`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.issueRuns(issueId) });
    },
  });
}

export function useCancelRun(issueId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (runId: string) =>
      apiFetch<void>(`/api/runs/${runId}/cancel`, { method: 'POST' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.issueRuns(issueId) }),
  });
}

export function useProbeRun() {
  return useMutation({
    mutationFn: (runId: string) =>
      apiFetch<{ alive: boolean; pid: number }>(
        `/api/runs/${runId}/probe`,
        { method: 'POST' },
      ),
  });
}

export function useMarkOrphan(issueId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (runId: string) =>
      apiFetch<void>(`/api/runs/${runId}/mark_orphan`, { method: 'POST' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.issueRuns(issueId) }),
  });
}

export function fetchLogChunk(
  runId: string,
  stream: 'stdout' | 'stderr' | 'events',
  from: number,
): Promise<LogChunk> {
  return apiFetch<LogChunk>(
    `/api/runs/${runId}/logs?stream=${stream}&from=${from}`,
  );
}

// ---- notifications --------------------------------------------------------
export function useNotifications(
  projectId: string | null,
  unreadOnly = false,
) {
  const params = unreadOnly ? '?unread=true' : '';
  return useQuery({
    enabled: !!projectId,
    queryKey: qk.notifications(projectId ?? '', unreadOnly),
    queryFn: () =>
      apiFetch<Notification[]>(
        `/api/projects/${projectId}/notifications${params}`,
      ),
    refetchInterval: 30000,
  });
}

export function useNotificationCount(projectId: string | null) {
  return useQuery({
    enabled: !!projectId,
    queryKey: qk.notificationCount(projectId ?? ''),
    queryFn: () =>
      apiFetch<{ count: number }>(
        `/api/projects/${projectId}/notifications/count`,
      ),
    refetchInterval: 15000,
  });
}

export function useMarkNotificationRead(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/notifications/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ read: true }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications', projectId] });
      qc.invalidateQueries({ queryKey: qk.notificationCount(projectId) });
    },
  });
}

export function useArchiveNotification(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/notifications/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ archived: true }),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notifications', projectId] });
      qc.invalidateQueries({ queryKey: qk.notificationCount(projectId) });
    },
  });
}

// ---- chat topics ----------------------------------------------------------
export function useTopics(projectId: string | null) {
  return useQuery({
    enabled: !!projectId,
    queryKey: qk.topics(projectId ?? ''),
    queryFn: () => apiFetch<ChatTopic[]>(`/api/projects/${projectId}/topics`),
  });
}

export function useTopic(topicId: string | null) {
  return useQuery({
    enabled: !!topicId,
    queryKey: qk.topic(topicId ?? ''),
    queryFn: () => apiFetch<ChatTopic>(`/api/topics/${topicId}`),
    refetchInterval: (q) => {
      const data = q.state.data as ChatTopic | undefined;
      if (!data) return 5000;
      return data.status === 'open' ? 5000 : false;
    },
  });
}

export function useMessages(topicId: string | null, topicStatus?: TopicStatus) {
  return useQuery({
    enabled: !!topicId,
    queryKey: qk.messages(topicId ?? ''),
    queryFn: () => apiFetch<ChatMessage[]>(`/api/topics/${topicId}/messages`),
    refetchInterval: topicStatus === 'open' ? 2000 : false,
  });
}

export function useCreateTopic(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; plugin: string }) =>
      apiFetch<ChatTopic>(`/api/projects/${projectId}/topics`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.topics(projectId) }),
  });
}

export function useSendMessage(topicId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { content: string }) =>
      apiFetch<ChatMessage>(`/api/topics/${topicId}/messages`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: qk.messages(topicId) }),
  });
}

export function useCloseTopic() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<void>(`/api/topics/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['topics'] });
      qc.invalidateQueries({ queryKey: ['topic'] });
    },
  });
}

export function useRestartTopic() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      apiFetch<ChatTopic>(`/api/topics/${id}/restart`, { method: 'POST' }),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: qk.topic(id) });
    },
  });
}

export function useDraftIssue(topicId: string) {
  return useMutation({
    mutationFn: () =>
      apiFetch<{ title: string; body: string }>(
        `/api/topics/${topicId}/draft_issue`,
        { method: 'POST' },
      ),
  });
}

export function usePublishIssue(topicId: string, projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; body: string }) =>
      apiFetch<Issue>(`/api/topics/${topicId}/publish_issue`, {
        method: 'POST',
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['issues', projectId] });
      qc.invalidateQueries({ queryKey: qk.topic(topicId) });
    },
  });
}

// ---- agents ---------------------------------------------------------------
export function useReloadAgents() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () =>
      apiFetch<{ loaded: number; disabled: string[] | null }>(
        `/api/agent_members/reload`,
        { method: 'POST' },
      ),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agent_members'] });
    },
  });
}
