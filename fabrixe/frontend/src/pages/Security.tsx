import { useState } from 'react'
import { ShieldCheck, Users, FileText, Monitor, Settings, Plus, Trash2, Unlock, Loader, Shield } from 'lucide-react'
import { security } from '../lib/api'
import { useApi } from '../hooks/useApi'
import { formatDate, formatRelative } from '../lib/format'
import { StatusBadge } from '../components/ui/StatusDot'
import Modal from '../components/ui/Modal'
import { PageLoader } from '../components/ui/Spinner'
import EmptyState from '../components/ui/EmptyState'
import type { UserRecord, AuditLog, DeviceRecord, Session } from '../types'

type Tab = 'users' | 'audit' | 'devices' | 'sessions'

type UserForm = { username: string; email: string; password: string; role: string; full_name: string }
const emptyUser: UserForm = { username: '', email: '', password: '', role: 'viewer', full_name: '' }

export default function Security() {
  const [tab, setTab] = useState<Tab>('users')
  const usersApi = useApi<UserRecord[]>(() => security.users.list() as Promise<UserRecord[]>, [])
  const auditApi = useApi<AuditLog[]>(() => security.auditLogs({ limit: 200 }) as Promise<AuditLog[]>, [], { pollInterval: 30000 })
  const devicesApi = useApi<DeviceRecord[]>(() => security.devices.list() as Promise<DeviceRecord[]>, [], { pollInterval: 30000 })
  const sessionsApi = useApi<Session[]>(() => security.sessions.list() as Promise<Session[]>, [], { pollInterval: 30000 })

  const [userModal, setUserModal] = useState(false)
  const [userForm, setUserForm] = useState<UserForm>(emptyUser)
  const [saving, setSaving] = useState(false)
  const [actionId, setActionId] = useState<number | null>(null)
  const [error, setError] = useState('')
  const [auditFilter, setAuditFilter] = useState('')

  const createUser = async () => {
    if (!userForm.username || !userForm.email || !userForm.password) return
    setSaving(true)
    try {
      await security.users.create(userForm)
      setUserModal(false)
      setUserForm(emptyUser)
      usersApi.refetch()
    } catch (e) { setError(e instanceof Error ? e.message : 'Error') }
    finally { setSaving(false) }
  }

  const deleteUser = async (id: number) => {
    if (!confirm('Delete this user?')) return
    await security.users.delete(id)
    usersApi.refetch()
  }

  const unlockUser = async (id: number) => {
    setActionId(id)
    await security.users.unlock(id)
    await usersApi.refetch()
    setActionId(null)
  }

  const toggleActive = async (u: UserRecord) => {
    await security.users.update(u.id, { is_active: !u.is_active })
    usersApi.refetch()
  }

  const trustDevice = async (id: number) => {
    await security.devices.trust(id)
    devicesApi.refetch()
  }

  const revokeDevice = async (id: number) => {
    await security.devices.revoke(id)
    devicesApi.refetch()
  }

  const deleteDevice = async (id: number) => {
    if (!confirm('Remove this device?')) return
    await security.devices.delete(id)
    devicesApi.refetch()
  }

  const revokeSession = async (id: string) => {
    if (!confirm('Revoke this session?')) return
    await security.sessions.revoke(id)
    sessionsApi.refetch()
  }

  const roleColor: Record<string, string> = {
    administrator: 'text-fabrixe-400',
    operator: 'text-amber-400',
    viewer: 'text-slate-400',
  }

  const tabs: { id: Tab; label: string; icon: typeof Users }[] = [
    { id: 'users',    label: 'Users',        icon: Users },
    { id: 'audit',    label: 'Audit Logs',   icon: FileText },
    { id: 'devices',  label: 'Devices',      icon: Monitor },
    { id: 'sessions', label: 'Sessions',     icon: Shield },
  ]

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <ShieldCheck className="w-6 h-6 text-emerald-400" />
            Internal Security
          </h1>
          <p className="text-sm text-slate-500 mt-0.5">Users, audit logs, device trust, and active sessions</p>
        </div>
        {tab === 'users' && (
          <button className="btn-primary" onClick={() => setUserModal(true)}>
            <Plus className="w-4 h-4" />New User
          </button>
        )}
      </div>

      {error && (
        <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          {error} <button onClick={() => setError('')} className="ml-2 underline">Dismiss</button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 bg-slate-900 border border-slate-800 rounded-xl p-1 w-fit">
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              tab === t.id ? 'bg-fabrixe-600/20 text-fabrixe-300 border border-fabrixe-500/20'
                          : 'text-slate-500 hover:text-slate-300'
            }`}
          >
            <t.icon className="w-3.5 h-3.5" />{t.label}
          </button>
        ))}
      </div>

      {/* ── USERS ── */}
      {tab === 'users' && (
        <div className="card overflow-x-auto">
          {usersApi.loading ? <PageLoader /> : (
            <table className="table-root">
              <thead><tr>
                <th>Username</th><th>Full Name</th><th>Email</th><th>Role</th><th>Status</th><th>Last Login</th><th>Actions</th>
              </tr></thead>
              <tbody>
                {(usersApi.data ?? []).map(u => (
                  <tr key={u.id}>
                    <td className="font-mono text-sm font-semibold text-slate-200">{u.username}</td>
                    <td>{u.full_name || '—'}</td>
                    <td className="text-slate-500">{u.email}</td>
                    <td className={`font-medium ${roleColor[u.role] ?? ''}`}>{u.role}</td>
                    <td>
                      <div className="flex items-center gap-1.5">
                        <StatusBadge status={u.is_active ? 'online' : 'offline'} />
                        {u.locked_until && <span className="badge-warning text-xs">Locked</span>}
                      </div>
                    </td>
                    <td className="text-slate-500 text-xs">{u.last_login ? formatRelative(u.last_login) : 'Never'}</td>
                    <td>
                      <div className="flex items-center gap-1">
                        {u.locked_until && (
                          <button onClick={() => unlockUser(u.id)} disabled={actionId === u.id}
                            title="Unlock" className="p-1.5 rounded-lg text-amber-400 hover:bg-amber-500/10 transition-colors">
                            {actionId === u.id ? <Loader className="w-3.5 h-3.5 animate-spin" /> : <Unlock className="w-3.5 h-3.5" />}
                          </button>
                        )}
                        <button onClick={() => toggleActive(u)} className="p-1.5 rounded-lg text-slate-400 hover:bg-slate-700 transition-colors text-xs px-2">
                          {u.is_active ? 'Disable' : 'Enable'}
                        </button>
                        <button onClick={() => deleteUser(u.id)} className="p-1.5 rounded-lg text-red-500 hover:bg-red-500/10 transition-colors">
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {!usersApi.loading && (usersApi.data ?? []).length === 0 && (
            <EmptyState icon={Users} title="No users" description="Create your first user." />
          )}
        </div>
      )}

      {/* ── AUDIT LOGS ── */}
      {tab === 'audit' && (
        <div className="space-y-3">
          <div className="flex gap-3">
            <input className="input max-w-sm" placeholder="Filter by event type…" value={auditFilter}
              onChange={e => setAuditFilter(e.target.value)} />
          </div>
          <div className="card overflow-x-auto">
            {auditApi.loading ? <PageLoader /> : (
              <table className="table-root">
                <thead><tr>
                  <th>Timestamp</th><th>Event</th><th>Description</th><th>User</th><th>IP</th><th>Outcome</th>
                </tr></thead>
                <tbody>
                  {(auditApi.data ?? [])
                    .filter(l => auditFilter ? l.event_type.toLowerCase().includes(auditFilter.toLowerCase()) : true)
                    .map(l => (
                    <tr key={l.id}>
                      <td className="text-xs text-slate-500 font-mono whitespace-nowrap">{formatDate(l.created_at)}</td>
                      <td className="font-mono text-xs text-fabrixe-400">{l.event_type}</td>
                      <td className="text-xs text-slate-300 max-w-xs truncate">{l.description}</td>
                      <td className="text-xs">{l.username || '—'}</td>
                      <td className="font-mono text-xs text-slate-500">{l.ip_address || '—'}</td>
                      <td><StatusBadge status={l.outcome} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {!auditApi.loading && (auditApi.data ?? []).length === 0 && (
              <EmptyState icon={FileText} title="No audit logs" description="Events will appear here as they occur." />
            )}
          </div>
        </div>
      )}

      {/* ── DEVICES ── */}
      {tab === 'devices' && (
        <div className="card overflow-x-auto">
          {devicesApi.loading ? <PageLoader /> : (
            <table className="table-root">
              <thead><tr>
                <th>Name</th><th>IP</th><th>Type</th><th>Trust</th><th>Last Seen</th><th>Actions</th>
              </tr></thead>
              <tbody>
                {(devicesApi.data ?? []).map(d => (
                  <tr key={d.id}>
                    <td className="font-medium text-slate-200">{d.name}</td>
                    <td className="font-mono text-xs">{d.ip_address || '—'}</td>
                    <td className="text-slate-500">{d.device_type}</td>
                    <td><StatusBadge status={d.is_trusted ? 'online' : 'offline'} /></td>
                    <td className="text-xs text-slate-500">{formatRelative(d.last_seen)}</td>
                    <td>
                      <div className="flex items-center gap-1">
                        {d.is_trusted
                          ? <button onClick={() => revokeDevice(d.id)} className="text-xs px-2 py-1 rounded-lg text-amber-400 hover:bg-amber-500/10 border border-amber-500/20">Revoke</button>
                          : <button onClick={() => trustDevice(d.id)} className="text-xs px-2 py-1 rounded-lg text-emerald-400 hover:bg-emerald-500/10 border border-emerald-500/20">Trust</button>
                        }
                        <button onClick={() => deleteDevice(d.id)} className="p-1.5 rounded-lg text-red-500 hover:bg-red-500/10">
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {!devicesApi.loading && (devicesApi.data ?? []).length === 0 && (
            <EmptyState icon={Monitor} title="No devices recorded" description="Devices are tracked when users log in." />
          )}
        </div>
      )}

      {/* ── SESSIONS ── */}
      {tab === 'sessions' && (
        <div className="card overflow-x-auto">
          {sessionsApi.loading ? <PageLoader /> : (
            <table className="table-root">
              <thead><tr>
                <th>User</th><th>IP</th><th>User Agent</th><th>Created</th><th>Last Seen</th><th>Expires</th><th></th>
              </tr></thead>
              <tbody>
                {(sessionsApi.data ?? []).map(s => (
                  <tr key={s.id}>
                    <td className="font-medium">{s.username}</td>
                    <td className="font-mono text-xs">{s.ip_address}</td>
                    <td className="text-xs text-slate-500 max-w-[200px] truncate">{s.user_agent}</td>
                    <td className="text-xs text-slate-500">{formatRelative(s.created_at)}</td>
                    <td className="text-xs text-slate-500">{formatRelative(s.last_seen)}</td>
                    <td className="text-xs text-slate-500">{formatDate(s.expires_at)}</td>
                    <td>
                      <button onClick={() => revokeSession(s.id)} className="btn-danger py-1 px-2 text-xs">
                        Revoke
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {!sessionsApi.loading && (sessionsApi.data ?? []).length === 0 && (
            <EmptyState icon={Shield} title="No active sessions" description="Active login sessions appear here." />
          )}
        </div>
      )}

      {/* Create User Modal */}
      <Modal open={userModal} onClose={() => setUserModal(false)} title="Create User">
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Username *</label>
              <input className="input" value={userForm.username} onChange={e => setUserForm(f => ({ ...f, username: e.target.value }))} />
            </div>
            <div>
              <label className="label">Full Name</label>
              <input className="input" value={userForm.full_name} onChange={e => setUserForm(f => ({ ...f, full_name: e.target.value }))} />
            </div>
          </div>
          <div>
            <label className="label">Email *</label>
            <input className="input" type="email" value={userForm.email} onChange={e => setUserForm(f => ({ ...f, email: e.target.value }))} />
          </div>
          <div>
            <label className="label">Password * (min 10 chars)</label>
            <input className="input" type="password" value={userForm.password} onChange={e => setUserForm(f => ({ ...f, password: e.target.value }))} />
          </div>
          <div>
            <label className="label">Role</label>
            <select className="input" value={userForm.role} onChange={e => setUserForm(f => ({ ...f, role: e.target.value }))}>
              <option value="viewer">Viewer — read-only</option>
              <option value="operator">Operator — manage resources</option>
              <option value="administrator">Administrator — full access</option>
            </select>
          </div>
          <div className="flex justify-end gap-2">
            <button className="btn-secondary" onClick={() => setUserModal(false)}>Cancel</button>
            <button className="btn-primary" onClick={createUser} disabled={saving}>
              {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
              Create User
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
