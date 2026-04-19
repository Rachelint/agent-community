// Thin fetch wrapper. All responses are expected to be JSON; non-2xx
// results are surfaced as Error with the server's `error` field when
// available.
export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
    ...init,
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  const body = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const msg = body?.error ?? `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return body as T;
}
