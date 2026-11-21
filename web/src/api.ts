export interface Channel {
  id: number
  name: string
  base_url: string
  api_key_masked: string
  models: string[]
  enabled: boolean
  created_at: string
}

export interface Member {
  id: number
  name: string
  role: string
  status: string
  enterprise_limit: number
  enterprise_used: number
  enterprise_reserved: number
  remaining: number
  created_at: string
}

export interface Audit {
  request_id: string
  created_at: string
  member_id: number
  member_name: string
  channel_id: number
  quota_source: string
  model: string
  result: string
  error_code: string
  http_status: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  latency_ms: number
}

const TOKEN_KEY = 'onemore.admin_token'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? ''
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Authorization', `Bearer ${getToken()}`)
  if (init.body) headers.set('Content-Type', 'application/json')
  const res = await fetch(path, { ...init, headers })
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => null)
  if (!res.ok) {
    const message = data?.error?.message ?? `HTTP ${res.status}`
    throw new Error(message)
  }
  return data as T
}

export const api = {
  channels: () => request<Channel[]>('/api/channels'),
  createChannel: (body: {
    name: string
    base_url: string
    api_key: string
    models: string[]
  }) => request<{ id: number }>('/api/channels', { method: 'POST', body: JSON.stringify(body) }),
  testChannel: (id: number) =>
    request<{ ok: boolean }>(`/api/channels/${id}/test`, { method: 'POST' }),
  setChannelEnabled: (id: number, enabled: boolean) =>
    request<void>(`/api/channels/${id}`, { method: 'PATCH', body: JSON.stringify({ enabled }) }),

  members: () => request<Member[]>('/api/members'),
  createMember: (body: { name: string; enterprise_limit: number }) =>
    request<{ id: number; name: string; token: string; enterprise_limit: number }>('/api/members', {
      method: 'POST',
      body: JSON.stringify(body),
    }),
  setMemberQuota: (id: number, limit: number) =>
    request<void>(`/api/members/${id}/quota`, { method: 'PUT', body: JSON.stringify({ limit }) }),

  audits: () => request<Audit[]>('/api/audits?limit=100'),
}
