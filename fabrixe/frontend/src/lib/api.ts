import type { ApiResponse } from '../types'

const BASE = '/api'

function getToken(): string | null {
  return localStorage.getItem('fabrixe_token')
}

function setTokens(access: string, refresh: string): void {
  localStorage.setItem('fabrixe_token', access)
  localStorage.setItem('fabrixe_refresh', refresh)
}

function clearTokens(): void {
  localStorage.removeItem('fabrixe_token')
  localStorage.removeItem('fabrixe_refresh')
}

async function request<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(`${BASE}${path}`, { ...options, headers })

  // Token expired — try refresh
  if (res.status === 401) {
    const refreshToken = localStorage.getItem('fabrixe_refresh')
    if (refreshToken && !path.startsWith('/auth/refresh')) {
      try {
        const refreshRes = await fetch(`${BASE}/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        })
        if (refreshRes.ok) {
          const data: ApiResponse<{ access_token: string; refresh_token: string }> =
            await refreshRes.json()
          if (data.success && data.data) {
            setTokens(data.data.access_token, data.data.refresh_token)
            return request<T>(path, options)
          }
        }
      } catch { /* ignore */ }
    }
    clearTokens()
    window.location.href = '/login'
    throw new Error('Unauthenticated')
  }

  const json: ApiResponse<T> = await res.json()
  if (!json.success) throw new Error(json.error ?? 'Request failed')
  return json.data as T
}

// ─────────────────────────────────────────────
// Auth
// ─────────────────────────────────────────────

export const auth = {
  async login(username: string, password: string) {
    const data = await request<{
      access_token: string
      refresh_token: string
      expires_at: string
      token_type: string
      user: { id: number; username: string; full_name: string; role: string }
    }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
    setTokens(data.access_token, data.refresh_token)
    return data
  },

  async logout(refreshToken?: string) {
    await request('/auth/logout', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    clearTokens()
  },

  async me() {
    return request<{
      id: number; username: string; email: string;
      full_name: string; role: string; last_login?: string
    }>('/auth/me')
  },

  async changePassword(currentPassword: string, newPassword: string) {
    return request('/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    })
  },

  isAuthenticated: () => !!getToken(),
  getToken,
  clearTokens,
}

// ─────────────────────────────────────────────
// System
// ─────────────────────────────────────────────

export const system = {
  snapshot: () => request('/system/snapshot'),
  hardware: () => request('/system/hardware'),
  services: () => request('/system/services'),
  serviceAction: (name: string, action: string) =>
    request(`/system/services/${encodeURIComponent(name)}/${action}`, { method: 'POST' }),
  alerts: (resolved?: boolean) =>
    request(`/system/alerts${resolved !== undefined ? `?resolved=${resolved}` : ''}`),
  resolveAlert: (id: number) =>
    request(`/system/alerts/${id}/resolve`, { method: 'POST' }),
}

// ─────────────────────────────────────────────
// Deployment
// ─────────────────────────────────────────────

export const deployment = {
  list: () => request('/deployment'),
  get: (id: number) => request(`/deployment/${id}`),
  create: (body: object) =>
    request('/deployment', { method: 'POST', body: JSON.stringify(body) }),
  update: (id: number, body: object) =>
    request(`/deployment/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id: number) =>
    request(`/deployment/${id}`, { method: 'DELETE' }),
  run: (id: number) =>
    request(`/deployment/${id}/run`, { method: 'POST' }),

  tasks: {
    list: () => request('/deployment/tasks'),
    get: (id: number) => request(`/deployment/tasks/${id}`),
    create: (body: object) =>
      request('/deployment/tasks', { method: 'POST', body: JSON.stringify(body) }),
    update: (id: number, body: object) =>
      request(`/deployment/tasks/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    delete: (id: number) =>
      request(`/deployment/tasks/${id}`, { method: 'DELETE' }),
    run: (id: number) =>
      request(`/deployment/tasks/${id}/run`, { method: 'POST' }),
  },
}

// ─────────────────────────────────────────────
// Security
// ─────────────────────────────────────────────

export const security = {
  users: {
    list: () => request('/security/users'),
    get: (id: number) => request(`/security/users/${id}`),
    create: (body: object) =>
      request('/security/users', { method: 'POST', body: JSON.stringify(body) }),
    update: (id: number, body: object) =>
      request(`/security/users/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    delete: (id: number) =>
      request(`/security/users/${id}`, { method: 'DELETE' }),
    unlock: (id: number) =>
      request(`/security/users/${id}/unlock`, { method: 'POST' }),
  },
  auditLogs: (params?: { event_type?: string; username?: string; limit?: number; offset?: number }) => {
    const qs = new URLSearchParams()
    if (params?.event_type) qs.set('event_type', params.event_type)
    if (params?.username) qs.set('username', params.username)
    if (params?.limit) qs.set('limit', String(params.limit))
    if (params?.offset) qs.set('offset', String(params.offset))
    return request(`/security/audit-logs?${qs}`)
  },
  devices: {
    list: () => request('/security/devices'),
    trust: (id: number) => request(`/security/devices/${id}/trust`, { method: 'POST' }),
    revoke: (id: number) => request(`/security/devices/${id}/revoke`, { method: 'POST' }),
    delete: (id: number) => request(`/security/devices/${id}`, { method: 'DELETE' }),
  },
  sessions: {
    list: () => request('/security/sessions'),
    revoke: (id: string) => request(`/security/sessions/${id}`, { method: 'DELETE' }),
  },
  settings: {
    get: () => request('/security/settings'),
    update: (body: Record<string, string>) =>
      request('/security/settings', { method: 'PUT', body: JSON.stringify(body) }),
  },
}

// ─────────────────────────────────────────────
// Communication
// ─────────────────────────────────────────────

export const communication = {
  identity: () => request('/comm/identity'),
  nodes: {
    list: () => request('/comm/nodes'),
    get: (id: number) => request(`/comm/nodes/${id}`),
    add: (body: object) =>
      request('/comm/nodes', { method: 'POST', body: JSON.stringify(body) }),
    trust: (id: number) => request(`/comm/nodes/${id}/trust`, { method: 'POST' }),
    revoke: (id: number) => request(`/comm/nodes/${id}/revoke`, { method: 'POST' }),
    delete: (id: number) => request(`/comm/nodes/${id}`, { method: 'DELETE' }),
    ping: (id: number) => request(`/comm/ping/${id}`, { method: 'POST' }),
  },
}
