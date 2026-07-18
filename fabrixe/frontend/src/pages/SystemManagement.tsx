import { useState } from 'react'
import { Monitor, RefreshCw, Play, Square, RotateCcw, Server, AlertTriangle, CheckCircle, Cpu, HardDrive, MemoryStick } from 'lucide-react'
import { system } from '../lib/api'
import { useApi } from '../hooks/useApi'
import { useLiveSnapshot } from '../hooks/useWebSocket'
import { formatBytes, formatUptime, formatPercent, usageColor } from '../lib/format'
import ProgressBar from '../components/ui/ProgressBar'
import { StatusBadge } from '../components/ui/StatusDot'
import { PageLoader } from '../components/ui/Spinner'
import EmptyState from '../components/ui/EmptyState'
import type { ServiceInfo, Alert } from '../types'

export default function SystemManagement() {
  const { snapshot, connected } = useLiveSnapshot(true)
  const servicesApi = useApi<ServiceInfo[]>(() => system.services() as Promise<ServiceInfo[]>, [], { pollInterval: 10000 })
  const alertsApi = useApi<Alert[]>(() => system.alerts(false) as Promise<Alert[]>, [], { pollInterval: 15000 })
  const hwApi = useApi(() => system.hardware(), [])

  const [serviceFilter, setServiceFilter] = useState('')
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [actionResult, setActionResult] = useState<{ name: string; out: string } | null>(null)
  const [tab, setTab] = useState<'overview' | 'services' | 'alerts' | 'hardware'>('overview')

  const doAction = async (name: string, action: 'start' | 'stop' | 'restart') => {
    setActionLoading(name + action)
    try {
      const res: { output: string } = await system.serviceAction(name, action) as { output: string }
      setActionResult({ name, out: res.output || `${action} succeeded` })
      await servicesApi.refetch()
    } catch (err) {
      setActionResult({ name, out: err instanceof Error ? err.message : 'Error' })
    } finally {
      setActionLoading(null)
    }
  }

  const resolveAlert = async (id: number) => {
    await system.resolveAlert(id)
    alertsApi.refetch()
  }

  const tabs = [
    { id: 'overview' as const, label: 'Overview' },
    { id: 'services' as const, label: 'Services' },
    { id: 'alerts' as const, label: 'Alerts', badge: (alertsApi.data?.length ?? 0) },
    { id: 'hardware' as const, label: 'Hardware' },
  ]

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <Monitor className="w-6 h-6 text-fabrixe-400" />
            System Management
          </h1>
          <p className="text-sm text-slate-500 mt-0.5">Real-time server monitoring and control</p>
        </div>
        <div className="flex items-center gap-2">
          {connected ? <span className="badge-online"><span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />Live</span>
            : <span className="badge-offline">Reconnecting</span>}
          <button onClick={() => servicesApi.refetch()} className="btn-secondary">
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-slate-900 border border-slate-800 rounded-xl p-1">
        {tabs.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              tab === t.id
                ? 'bg-fabrixe-600/20 text-fabrixe-300 border border-fabrixe-500/20'
                : 'text-slate-500 hover:text-slate-300'
            }`}
          >
            {t.label}
            {t.badge ? (
              <span className="px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-400 text-xs">{t.badge}</span>
            ) : null}
          </button>
        ))}
      </div>

      {/* ── OVERVIEW ── */}
      {tab === 'overview' && (
        <div className="space-y-4">
          {!snapshot ? <PageLoader /> : (
            <>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {/* CPU detail */}
                <div className="card">
                  <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2"><Cpu className="w-4 h-4" />CPU</h3>
                  <p className={`text-3xl font-bold mb-1 ${usageColor(snapshot.cpu.usage_total_percent)}`}>
                    {formatPercent(snapshot.cpu.usage_total_percent)}
                  </p>
                  <p className="text-xs text-slate-500 mb-3">{snapshot.cpu.model}</p>
                  <ProgressBar value={snapshot.cpu.usage_total_percent} showLabel />
                  <div className="grid grid-cols-2 gap-2 mt-3 text-xs text-slate-500">
                    <div>Cores: <span className="text-slate-300">{snapshot.cpu.cores}</span></div>
                    <div>Freq: <span className="text-slate-300">{snapshot.cpu.frequency_mhz.toFixed(0)} MHz</span></div>
                  </div>
                  <div className="mt-3 space-y-1">
                    {snapshot.cpu.per_core_percent?.map((pct, i) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <span className="text-slate-600 w-10 font-mono">Core {i}</span>
                        <ProgressBar value={pct} size="sm" />
                        <span className={`w-8 text-right ${usageColor(pct)}`}>{pct.toFixed(0)}%</span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Memory detail */}
                <div className="card">
                  <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2"><MemoryStick className="w-4 h-4" />Memory</h3>
                  <p className={`text-3xl font-bold mb-1 ${usageColor(snapshot.memory.usage_percent)}`}>
                    {formatPercent(snapshot.memory.usage_percent)}
                  </p>
                  <p className="text-xs text-slate-500 mb-3">{formatBytes(snapshot.memory.used_bytes)} used of {formatBytes(snapshot.memory.total_bytes)}</p>
                  <ProgressBar value={snapshot.memory.usage_percent} showLabel />
                  <div className="mt-3 space-y-2 text-xs">
                    {[
                      ['Total',   formatBytes(snapshot.memory.total_bytes)],
                      ['Used',    formatBytes(snapshot.memory.used_bytes)],
                      ['Free',    formatBytes(snapshot.memory.free_bytes)],
                      ['Cached',  formatBytes(snapshot.memory.cached_bytes)],
                      ['Buffers', formatBytes(snapshot.memory.buffers_bytes)],
                      ['Swap',    `${formatBytes(snapshot.memory.swap_used_bytes)} / ${formatBytes(snapshot.memory.swap_total_bytes)}`],
                    ].map(([k, v]) => (
                      <div key={k} className="flex justify-between">
                        <span className="text-slate-500">{k}</span>
                        <span className="text-slate-300 font-medium">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>

                {/* System info */}
                <div className="card">
                  <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2"><Server className="w-4 h-4" />System</h3>
                  <div className="space-y-2 text-xs">
                    {[
                      ['Hostname',  snapshot.system.hostname],
                      ['OS',        `${snapshot.system.os} ${snapshot.system.os_version}`],
                      ['Kernel',    snapshot.system.kernel],
                      ['Arch',      snapshot.system.arch],
                      ['Uptime',    formatUptime(snapshot.system.uptime_seconds)],
                      ['Processes', String(snapshot.system.process_count)],
                      ['Load 1m',   snapshot.system.load_avg_1.toFixed(2)],
                      ['Load 5m',   snapshot.system.load_avg_5.toFixed(2)],
                      ['Load 15m',  snapshot.system.load_avg_15.toFixed(2)],
                    ].map(([k, v]) => (
                      <div key={k} className="flex justify-between">
                        <span className="text-slate-500">{k}</span>
                        <span className="text-slate-300 font-medium font-mono text-right">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              {/* Disks */}
              <div className="card">
                <h3 className="text-sm font-medium text-slate-400 mb-4 flex items-center gap-2"><HardDrive className="w-4 h-4" />Storage</h3>
                <div className="overflow-x-auto">
                  <table className="table-root">
                    <thead><tr>
                      <th>Device</th><th>Mount</th><th>FS</th><th>Total</th><th>Used</th><th>Free</th><th>Usage</th>
                    </tr></thead>
                    <tbody>
                      {snapshot.disks.map(d => (
                        <tr key={d.mountpoint}>
                          <td className="font-mono text-xs">{d.device}</td>
                          <td className="font-mono text-xs">{d.mountpoint}</td>
                          <td>{d.fstype}</td>
                          <td>{formatBytes(d.total_bytes)}</td>
                          <td>{formatBytes(d.used_bytes)}</td>
                          <td>{formatBytes(d.free_bytes)}</td>
                          <td className="min-w-[120px]">
                            <div className="flex items-center gap-2">
                              <ProgressBar value={d.use_percent} size="sm" />
                              <span className={`text-xs w-12 ${usageColor(d.use_percent)}`}>{formatPercent(d.use_percent)}</span>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </>
          )}
        </div>
      )}

      {/* ── SERVICES ── */}
      {tab === 'services' && (
        <div className="space-y-4">
          <div className="flex gap-3">
            <input
              type="text"
              className="input max-w-sm"
              placeholder="Filter services…"
              value={serviceFilter}
              onChange={e => setServiceFilter(e.target.value)}
            />
          </div>

          {actionResult && (
            <div className="card bg-slate-800/50">
              <div className="flex justify-between items-center mb-2">
                <span className="text-sm font-medium text-slate-300">Output: {actionResult.name}</span>
                <button onClick={() => setActionResult(null)} className="text-slate-500 text-xs hover:text-slate-300">Dismiss</button>
              </div>
              <pre className="text-xs font-mono text-slate-400 whitespace-pre-wrap">{actionResult.out}</pre>
            </div>
          )}

          <div className="card overflow-x-auto">
            {servicesApi.loading ? <PageLoader /> : (
              <table className="table-root">
                <thead><tr>
                  <th>Service</th><th>Description</th><th>State</th><th>Sub-state</th><th>Actions</th>
                </tr></thead>
                <tbody>
                  {(servicesApi.data ?? [])
                    .filter(s => s.name.toLowerCase().includes(serviceFilter.toLowerCase()))
                    .map(s => (
                    <tr key={s.name}>
                      <td className="font-mono text-xs text-slate-200">{s.name}</td>
                      <td className="text-slate-500 text-xs max-w-xs truncate">{s.description}</td>
                      <td><StatusBadge status={s.active_state} /></td>
                      <td className="text-xs text-slate-500">{s.sub_state}</td>
                      <td>
                        <div className="flex items-center gap-1">
                          <button
                            onClick={() => doAction(s.name, 'start')}
                            disabled={actionLoading !== null}
                            title="Start"
                            className="p-1.5 rounded-lg text-emerald-500 hover:bg-emerald-500/10 disabled:opacity-40 transition-colors"
                          >
                            <Play className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => doAction(s.name, 'stop')}
                            disabled={actionLoading !== null}
                            title="Stop"
                            className="p-1.5 rounded-lg text-red-500 hover:bg-red-500/10 disabled:opacity-40 transition-colors"
                          >
                            <Square className="w-3.5 h-3.5" />
                          </button>
                          <button
                            onClick={() => doAction(s.name, 'restart')}
                            disabled={actionLoading !== null}
                            title="Restart"
                            className="p-1.5 rounded-lg text-amber-500 hover:bg-amber-500/10 disabled:opacity-40 transition-colors"
                          >
                            <RotateCcw className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {!servicesApi.loading && (servicesApi.data ?? []).length === 0 && (
              <EmptyState icon={Server} title="No services found" description="systemd services will appear here." />
            )}
          </div>
        </div>
      )}

      {/* ── ALERTS ── */}
      {tab === 'alerts' && (
        <div className="space-y-3">
          {alertsApi.loading ? <PageLoader /> :
            (alertsApi.data ?? []).length === 0 ? (
              <EmptyState icon={CheckCircle} title="No active alerts" description="The system is healthy." />
            ) : (
              (alertsApi.data ?? []).map(a => (
                <div key={a.id} className={`card-sm flex items-start gap-3 ${
                  a.level === 'critical' ? 'border-red-500/30' :
                  a.level === 'warning' ? 'border-amber-500/30' : 'border-slate-700'
                }`}>
                  <AlertTriangle className={`w-4 h-4 shrink-0 mt-0.5 ${
                    a.level === 'critical' ? 'text-red-400' :
                    a.level === 'warning' ? 'text-amber-400' : 'text-blue-400'
                  }`} />
                  <div className="flex-1">
                    <p className="text-sm text-slate-200">{a.message}</p>
                    <p className="text-xs text-slate-500 mt-0.5">{a.source} · {a.created_at}</p>
                  </div>
                  <button onClick={() => resolveAlert(a.id)} className="btn-success text-xs py-1 px-2">
                    Resolve
                  </button>
                </div>
              ))
            )
          }
        </div>
      )}

      {/* ── HARDWARE ── */}
      {tab === 'hardware' && (
        <div className="card">
          {hwApi.loading ? <PageLoader /> : !hwApi.data ? null : (
            <div className="space-y-6">
              <div>
                <h3 className="section-title">Hardware Information</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-sm">
                  {[
                    ['Vendor',   (hwApi.data as { vendor?: string }).vendor || '—'],
                    ['Product',  (hwApi.data as { product?: string }).product || '—'],
                    ['Serial',   (hwApi.data as { serial_number?: string }).serial_number || '—'],
                    ['CPU',      (hwApi.data as { cpu?: string }).cpu || '—'],
                    ['CPU Cores',(hwApi.data as { cpu_cores?: number }).cpu_cores ?? '—'],
                    ['RAM',      formatBytes((hwApi.data as { memory_total_bytes?: number }).memory_total_bytes ?? 0)],
                  ].map(([k, v]) => (
                    <div key={String(k)} className="flex justify-between py-2 border-b border-slate-800">
                      <span className="text-slate-500">{k}</span>
                      <span className="text-slate-200 font-mono text-xs">{String(v)}</span>
                    </div>
                  ))}
                </div>
              </div>
              {((hwApi.data as { disk_models?: string[] }).disk_models ?? []).length > 0 && (
                <div>
                  <h4 className="text-sm font-medium text-slate-400 mb-2">Storage Devices</h4>
                  <ul className="space-y-1">
                    {((hwApi.data as { disk_models?: string[] }).disk_models ?? []).map((d: string, i: number) => (
                      <li key={i} className="text-sm text-slate-300 font-mono">{d}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
