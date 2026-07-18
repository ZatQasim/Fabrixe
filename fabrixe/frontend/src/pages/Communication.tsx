import { useState } from 'react'
import { Network, Plus, Trash2, Shield, ShieldOff, Zap, Copy, Check, Loader, Server } from 'lucide-react'
import { communication } from '../lib/api'
import { useApi } from '../hooks/useApi'
import { formatRelative } from '../lib/format'
import { StatusBadge } from '../components/ui/StatusDot'
import Modal from '../components/ui/Modal'
import { PageLoader } from '../components/ui/Spinner'
import EmptyState from '../components/ui/EmptyState'
import type { CommunicationNode, NodeIdentity } from '../types'

type AddForm = { display_name: string; endpoint: string; public_key: string }
const emptyForm: AddForm = { display_name: '', endpoint: '', public_key: '' }

export default function Communication() {
  const nodesApi = useApi<CommunicationNode[]>(() => communication.nodes.list() as Promise<CommunicationNode[]>, [], { pollInterval: 20000 })
  const identityApi = useApi<NodeIdentity>(() => communication.identity() as Promise<NodeIdentity>, [])

  const [tab, setTab] = useState<'nodes' | 'identity'>('nodes')
  const [addModal, setAddModal] = useState(false)
  const [form, setForm] = useState<AddForm>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [pingLoading, setPingLoading] = useState<number | null>(null)
  const [pingResults, setPingResults] = useState<Record<number, { status: string; latency_ms: number }>>({})
  const [error, setError] = useState('')
  const [copied, setCopied] = useState(false)

  const addNode = async () => {
    if (!form.display_name || !form.endpoint || !form.public_key) return
    setSaving(true)
    try {
      await communication.nodes.add(form)
      setAddModal(false)
      setForm(emptyForm)
      nodesApi.refetch()
    } catch (e) { setError(e instanceof Error ? e.message : 'Error') }
    finally { setSaving(false) }
  }

  const trust = async (id: number) => {
    await communication.nodes.trust(id)
    nodesApi.refetch()
  }

  const revoke = async (id: number) => {
    await communication.nodes.revoke(id)
    nodesApi.refetch()
  }

  const del = async (id: number) => {
    if (!confirm('Remove this node?')) return
    await communication.nodes.delete(id)
    nodesApi.refetch()
  }

  const ping = async (id: number) => {
    setPingLoading(id)
    try {
      const res = await communication.nodes.ping(id) as { status: string; latency_ms: number }
      setPingResults(p => ({ ...p, [id]: res }))
      nodesApi.refetch()
    } catch (e) {
      setPingResults(p => ({ ...p, [id]: { status: 'offline', latency_ms: 0 } }))
    } finally { setPingLoading(null) }
  }

  const copyKey = () => {
    if (!identityApi.data?.public_key) return
    navigator.clipboard.writeText(identityApi.data.public_key)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <Network className="w-6 h-6 text-cyan-400" />
            Protected Communication
          </h1>
          <p className="text-sm text-slate-500 mt-0.5">Encrypted peer-to-peer Fabrixe node network</p>
        </div>
        {tab === 'nodes' && (
          <button className="btn-primary" onClick={() => setAddModal(true)}>
            <Plus className="w-4 h-4" />Add Node
          </button>
        )}
      </div>

      {/* Architecture diagram text */}
      <div className="card bg-slate-900/50 border-fabrixe-500/10">
        <div className="flex items-center gap-3 text-sm text-slate-400">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-fabrixe-600/20 border border-fabrixe-500/30 flex items-center justify-center">
              <Server className="w-4 h-4 text-fabrixe-400" />
            </div>
            <span className="font-medium text-slate-300">This Node</span>
          </div>
          <div className="flex-1 border-t border-dashed border-fabrixe-500/30 relative">
            <span className="absolute left-1/2 -translate-x-1/2 -top-3 text-xs text-fabrixe-500 bg-slate-900 px-2">TLS Encrypted</span>
          </div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-cyan-600/20 border border-cyan-500/30 flex items-center justify-center">
              <Server className="w-4 h-4 text-cyan-400" />
            </div>
            <span className="font-medium text-slate-300">Remote Nodes</span>
          </div>
        </div>
      </div>

      {error && (
        <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          {error} <button onClick={() => setError('')} className="ml-2 underline">Dismiss</button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 bg-slate-900 border border-slate-800 rounded-xl p-1 w-fit">
        {[
          { id: 'nodes' as const, label: 'Nodes' },
          { id: 'identity' as const, label: 'Node Identity' },
        ].map(t => (
          <button key={t.id} onClick={() => setTab(t.id)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              tab === t.id ? 'bg-fabrixe-600/20 text-fabrixe-300 border border-fabrixe-500/20'
                          : 'text-slate-500 hover:text-slate-300'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* ── NODES ── */}
      {tab === 'nodes' && (
        <div className="space-y-3">
          {nodesApi.loading ? <PageLoader /> :
            (nodesApi.data ?? []).length === 0 ? (
              <EmptyState
                icon={Network}
                title="No nodes connected"
                description="Add remote Fabrixe nodes to build your encrypted network."
                action={<button className="btn-primary" onClick={() => setAddModal(true)}><Plus className="w-4 h-4" />Add First Node</button>}
              />
            ) : (
              (nodesApi.data ?? []).map(n => {
                const pingResult = pingResults[n.id]
                return (
                  <div key={n.id} className="card-sm">
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex items-start gap-3">
                        <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${
                          n.status === 'online' ? 'bg-emerald-500/10' :
                          n.status === 'offline' ? 'bg-red-500/10' : 'bg-slate-800'
                        }`}>
                          <Server className={`w-5 h-5 ${
                            n.status === 'online' ? 'text-emerald-400' :
                            n.status === 'offline' ? 'text-red-400' : 'text-slate-500'
                          }`} />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-slate-200">{n.display_name}</span>
                            <StatusBadge status={n.status} />
                            {n.is_trusted
                              ? <span className="badge-online"><Shield className="w-3 h-3" />Trusted</span>
                              : <span className="badge-warning">Untrusted</span>
                            }
                          </div>
                          <p className="text-xs font-mono text-slate-500 mt-0.5">{n.endpoint}</p>
                          <p className="text-xs text-slate-600 mt-0.5">
                            Fingerprint: <span className="font-mono">{n.fingerprint}</span>
                          </p>
                          {n.last_seen && <p className="text-xs text-slate-600">Last seen: {formatRelative(n.last_seen)}</p>}
                          {pingResult && (
                            <p className={`text-xs mt-1 ${pingResult.status === 'online' ? 'text-emerald-400' : 'text-red-400'}`}>
                              Ping: {pingResult.status} {pingResult.latency_ms > 0 ? `(${pingResult.latency_ms}ms)` : ''}
                            </p>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <button onClick={() => ping(n.id)} disabled={pingLoading === n.id} className="btn-secondary">
                          {pingLoading === n.id ? <Loader className="w-4 h-4 animate-spin" /> : <Zap className="w-4 h-4" />}
                          Ping
                        </button>
                        {n.is_trusted
                          ? <button onClick={() => revoke(n.id)} className="btn-secondary"><ShieldOff className="w-4 h-4" />Revoke</button>
                          : <button onClick={() => trust(n.id)} className="btn-success"><Shield className="w-4 h-4" />Trust</button>
                        }
                        <button onClick={() => del(n.id)} className="btn-danger p-2"><Trash2 className="w-4 h-4" /></button>
                      </div>
                    </div>
                  </div>
                )
              })
            )
          }
        </div>
      )}

      {/* ── IDENTITY ── */}
      {tab === 'identity' && (
        <div className="space-y-4">
          {identityApi.loading ? <PageLoader /> : !identityApi.data ? null : (
            <>
              <div className="card">
                <h3 className="section-title">This Node's Identity</h3>
                <div className="space-y-3 text-sm">
                  {[
                    ['Node ID',   identityApi.data.node_id],
                    ['Hostname',  identityApi.data.hostname],
                    ['Version',   identityApi.data.version],
                    ['Fingerprint', identityApi.data.fingerprint],
                  ].map(([k, v]) => (
                    <div key={k} className="flex flex-col sm:flex-row sm:justify-between py-2 border-b border-slate-800">
                      <span className="text-slate-500">{k}</span>
                      <span className="text-slate-200 font-mono text-xs mt-1 sm:mt-0">{v}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="card">
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-sm font-medium text-slate-400">Public Key (Share with peer nodes)</h3>
                  <button onClick={copyKey} className="btn-secondary text-xs">
                    {copied ? <><Check className="w-3.5 h-3.5 text-emerald-400" />Copied!</> : <><Copy className="w-3.5 h-3.5" />Copy</>}
                  </button>
                </div>
                <pre className="text-xs font-mono text-slate-400 bg-slate-950 rounded-xl p-4 overflow-x-auto whitespace-pre-wrap break-all">
                  {identityApi.data.public_key}
                </pre>
                <p className="text-xs text-slate-600 mt-2">
                  Share this public key with administrators of other Fabrixe nodes when adding this node to their trusted list.
                </p>
              </div>
            </>
          )}
        </div>
      )}

      {/* Add Node Modal */}
      <Modal open={addModal} onClose={() => setAddModal(false)} title="Add Fabrixe Node">
        <div className="space-y-4">
          <div>
            <label className="label">Display Name *</label>
            <input className="input" placeholder="Production Server A" value={form.display_name}
              onChange={e => setForm(f => ({ ...f, display_name: e.target.value }))} />
          </div>
          <div>
            <label className="label">Endpoint (HTTPS URL) *</label>
            <input className="input font-mono text-xs" placeholder="https://192.168.1.50:8443" value={form.endpoint}
              onChange={e => setForm(f => ({ ...f, endpoint: e.target.value }))} />
          </div>
          <div>
            <label className="label">Public Key (from peer's Node Identity page) *</label>
            <textarea className="input min-h-[120px] font-mono text-xs resize-y" placeholder="-----BEGIN PUBLIC KEY-----&#10;...&#10;-----END PUBLIC KEY-----"
              value={form.public_key} onChange={e => setForm(f => ({ ...f, public_key: e.target.value }))} />
          </div>
          <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400 text-xs">
            After adding the node, an administrator must explicitly trust it before encrypted communication is established.
          </div>
          <div className="flex justify-end gap-2">
            <button className="btn-secondary" onClick={() => setAddModal(false)}>Cancel</button>
            <button className="btn-primary" onClick={addNode} disabled={saving}>
              {saving ? <Loader className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
              Add Node
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}
