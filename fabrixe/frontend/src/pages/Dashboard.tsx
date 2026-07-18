import { useEffect, useState, useRef } from 'react'
import { Link } from 'react-router-dom'
import {
  Cpu, MemoryStick, HardDrive, Network, Activity,
  Monitor, Rocket, ShieldCheck, Wifi, Clock, Server,
  AlertTriangle, CheckCircle, ArrowRight
} from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import { useLiveSnapshot } from '../hooks/useWebSocket'
import { useApi } from '../hooks/useApi'
import { system } from '../lib/api'
import { formatBytes, formatUptime, formatPercent, usageColor, formatRelative } from '../lib/format'
import ProgressBar from '../components/ui/ProgressBar'
import MetricCard from '../components/ui/MetricCard'
import { PageLoader } from '../components/ui/Spinner'
import type { Alert, SystemSnapshot } from '../types'

const MAX_HISTORY = 60

interface HistoryPoint {
  t: string
  cpu: number
  mem: number
}

export default function Dashboard() {
  const { snapshot, connected } = useLiveSnapshot(true)
  const alertsApi = useApi<Alert[]>(() => system.alerts() as Promise<Alert[]>, [], { pollInterval: 30000 })
  const history = useRef<HistoryPoint[]>([])
  const [chartData, setChartData] = useState<HistoryPoint[]>([])

  useEffect(() => {
    if (!snapshot) return
    const point: HistoryPoint = {
      t: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
      cpu: snapshot.cpu.usage_total_percent,
      mem: snapshot.memory.usage_percent,
    }
    history.current = [...history.current.slice(-MAX_HISTORY + 1), point]
    setChartData([...history.current])
  }, [snapshot])

  if (!snapshot) return <PageLoader />

  const { system: sys, cpu, memory, disks } = snapshot
  const alerts = alertsApi.data ?? []
  const criticalAlerts = alerts.filter(a => a.level === 'critical' && !a.is_resolved)
  const warningAlerts = alerts.filter(a => a.level === 'warning' && !a.is_resolved)

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-100">Dashboard</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            {sys.hostname} · {sys.os} {sys.os_version} · {sys.kernel}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {connected ? (
            <span className="badge-online">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
              Live
            </span>
          ) : (
            <span className="badge-offline">Reconnecting…</span>
          )}
        </div>
      </div>

      {/* Alerts banner */}
      {(criticalAlerts.length > 0 || warningAlerts.length > 0) && (
        <div className="flex flex-wrap gap-3">
          {criticalAlerts.length > 0 && (
            <div className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span><strong>{criticalAlerts.length}</strong> critical alert{criticalAlerts.length !== 1 ? 's' : ''} — {criticalAlerts[0].message}</span>
            </div>
          )}
          {warningAlerts.length > 0 && (
            <div className="flex items-center gap-2 px-4 py-2.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-400 text-sm">
              <AlertTriangle className="w-4 h-4 shrink-0" />
              <span><strong>{warningAlerts.length}</strong> warning{warningAlerts.length !== 1 ? 's' : ''}</span>
            </div>
          )}
        </div>
      )}

      {/* Metric cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard
          label="CPU Usage"
          value={formatPercent(cpu.usage_total_percent)}
          sub={`${cpu.cores} cores · ${cpu.frequency_mhz.toFixed(0)} MHz`}
          icon={Cpu}
          iconColor="text-fabrixe-400"
          alert={cpu.usage_total_percent > 90}
        />
        <MetricCard
          label="Memory"
          value={formatPercent(memory.usage_percent)}
          sub={`${formatBytes(memory.used_bytes)} / ${formatBytes(memory.total_bytes)}`}
          icon={MemoryStick}
          iconColor="text-purple-400"
          alert={memory.usage_percent > 90}
        />
        <MetricCard
          label="Uptime"
          value={formatUptime(sys.uptime_seconds)}
          sub={`${sys.process_count} processes`}
          icon={Clock}
          iconColor="text-cyan-400"
        />
        <MetricCard
          label="Load Average"
          value={sys.load_avg_1.toFixed(2)}
          sub={`5m: ${sys.load_avg_5.toFixed(2)} · 15m: ${sys.load_avg_15.toFixed(2)}`}
          icon={Activity}
          iconColor="text-emerald-400"
          alert={sys.load_avg_1 > cpu.cores}
        />
      </div>

      {/* Charts + Disk row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* CPU + Memory time chart */}
        <div className="card lg:col-span-2">
          <h3 className="text-sm font-medium text-slate-400 mb-4">CPU & Memory — Last {MAX_HISTORY}s</h3>
          <ResponsiveContainer width="100%" height={180}>
            <AreaChart data={chartData} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="gCpu" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#6366f1" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="gMem" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#a855f7" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#a855f7" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="t" tick={{ fontSize: 10, fill: '#475569' }} tickLine={false} axisLine={false} interval="preserveStartEnd" />
              <YAxis domain={[0, 100]} tick={{ fontSize: 10, fill: '#475569' }} tickLine={false} axisLine={false} tickFormatter={v => `${v}%`} />
              <Tooltip
                contentStyle={{ background: '#1e293b', border: '1px solid #334155', borderRadius: 8, fontSize: 12 }}
                labelStyle={{ color: '#94a3b8' }}
                formatter={(val: number, name: string) => [`${val.toFixed(1)}%`, name === 'cpu' ? 'CPU' : 'Memory']}
              />
              <Area type="monotone" dataKey="cpu" stroke="#6366f1" fill="url(#gCpu)" strokeWidth={1.5} dot={false} />
              <Area type="monotone" dataKey="mem" stroke="#a855f7" fill="url(#gMem)" strokeWidth={1.5} dot={false} />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Disk */}
        <div className="card">
          <h3 className="text-sm font-medium text-slate-400 mb-4">Storage</h3>
          <div className="space-y-4">
            {disks.slice(0, 5).map(d => (
              <div key={d.mountpoint}>
                <div className="flex justify-between text-xs mb-1">
                  <span className="text-slate-400 font-mono truncate max-w-[120px]">{d.mountpoint}</span>
                  <span className={usageColor(d.use_percent)}>{formatPercent(d.use_percent)}</span>
                </div>
                <ProgressBar value={d.use_percent} size="sm" />
                <p className="text-xs text-slate-600 mt-0.5">{formatBytes(d.used_bytes)} / {formatBytes(d.total_bytes)} · {d.fstype}</p>
              </div>
            ))}
            {disks.length === 0 && <p className="text-sm text-slate-600">No disks found.</p>}
          </div>
        </div>
      </div>

      {/* Per-core CPU + Modules quick nav */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Per-core */}
        <div className="card">
          <h3 className="text-sm font-medium text-slate-400 mb-4 flex items-center gap-2">
            <Cpu className="w-4 h-4" /> CPU Cores ({cpu.model})
          </h3>
          <div className="grid grid-cols-2 gap-x-4 gap-y-2">
            {(cpu.per_core_percent ?? []).map((pct, i) => (
              <div key={i} className="flex items-center gap-2">
                <span className="text-xs text-slate-600 w-10 shrink-0 font-mono">Core {i}</span>
                <div className="flex-1">
                  <ProgressBar value={pct} size="sm" />
                </div>
                <span className={`text-xs w-10 text-right ${usageColor(pct)}`}>{pct.toFixed(0)}%</span>
              </div>
            ))}
          </div>
        </div>

        {/* Quick nav */}
        <div className="card">
          <h3 className="text-sm font-medium text-slate-400 mb-4 flex items-center gap-2">
            <Server className="w-4 h-4" /> Modules
          </h3>
          <div className="space-y-2">
            {[
              { to: '/system',        icon: Monitor,    label: 'System Management',        sub: 'Monitor & control servers' },
              { to: '/deployment',    icon: Rocket,     label: 'Deployment & Automation',  sub: 'Deploy, schedule, automate' },
              { to: '/security',      icon: ShieldCheck,label: 'Internal Security',        sub: 'Users, audit, devices' },
              { to: '/communication', icon: Wifi,       label: 'Protected Communication',  sub: 'Encrypted node network' },
            ].map(m => (
              <Link
                key={m.to}
                to={m.to}
                className="flex items-center gap-3 p-3 rounded-xl hover:bg-slate-800 transition-colors group"
              >
                <div className="w-9 h-9 rounded-lg bg-slate-800 group-hover:bg-fabrixe-600/20 flex items-center justify-center transition-colors shrink-0">
                  <m.icon className="w-4 h-4 text-slate-500 group-hover:text-fabrixe-400 transition-colors" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-slate-200">{m.label}</p>
                  <p className="text-xs text-slate-500">{m.sub}</p>
                </div>
                <ArrowRight className="w-4 h-4 text-slate-600 group-hover:text-slate-400 transition-colors" />
              </Link>
            ))}
          </div>
        </div>
      </div>

      {/* Network */}
      {snapshot.network.filter(n => n.name !== 'lo').length > 0 && (
        <div className="card">
          <h3 className="text-sm font-medium text-slate-400 mb-4 flex items-center gap-2">
            <Network className="w-4 h-4" /> Network Interfaces
          </h3>
          <div className="overflow-x-auto">
            <table className="table-root">
              <thead><tr>
                <th>Interface</th>
                <th>IP Address</th>
                <th>Bytes Recv</th>
                <th>Bytes Sent</th>
                <th>Errors In/Out</th>
              </tr></thead>
              <tbody>
                {snapshot.network.filter(n => n.name !== 'lo').map(n => (
                  <tr key={n.name}>
                    <td className="font-mono text-slate-200">{n.name}</td>
                    <td className="text-slate-400">{n.ip_addresses?.join(', ') || '—'}</td>
                    <td>{formatBytes(n.bytes_recv)}</td>
                    <td>{formatBytes(n.bytes_sent)}</td>
                    <td className={n.errors_in + n.errors_out > 0 ? 'text-red-400' : 'text-slate-500'}>
                      {n.errors_in}/{n.errors_out}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Recent Alerts */}
      {alerts.length > 0 && (
        <div className="card">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-slate-400">Recent Alerts</h3>
            <Link to="/system" className="text-xs text-fabrixe-400 hover:text-fabrixe-300">View all →</Link>
          </div>
          <div className="space-y-2">
            {alerts.slice(0, 5).map(a => (
              <div key={a.id} className="flex items-start gap-3 p-3 rounded-xl bg-slate-800/50">
                {a.is_resolved
                  ? <CheckCircle className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                  : <AlertTriangle className={`w-4 h-4 shrink-0 mt-0.5 ${a.level === 'critical' ? 'text-red-400' : 'text-amber-400'}`} />
                }
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-slate-200">{a.message}</p>
                  <p className="text-xs text-slate-500 mt-0.5">{a.source} · {formatRelative(a.created_at)}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
