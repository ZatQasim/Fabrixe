import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Monitor,
  Rocket,
  ShieldCheck,
  Network,
  Server,
} from 'lucide-react'
import { clsx } from 'clsx'

const nav = [
  { to: '/',             icon: LayoutDashboard, label: 'Dashboard',               exact: true },
  { to: '/system',       icon: Monitor,         label: 'System Management' },
  { to: '/deployment',   icon: Rocket,          label: 'Deployment & Automation' },
  { to: '/security',     icon: ShieldCheck,     label: 'Internal Security' },
  { to: '/communication',icon: Network,         label: 'Protected Communication' },
]

export default function Sidebar() {
  return (
    <aside className="flex flex-col w-64 bg-slate-900 border-r border-slate-800 shrink-0">
      {/* Logo */}
      <div className="flex items-center gap-3 px-5 py-5 border-b border-slate-800">
        <div className="w-9 h-9 rounded-lg bg-fabrixe-600 flex items-center justify-center shrink-0">
          <Server className="w-5 h-5 text-white" />
        </div>
        <div>
          <div className="text-base font-bold text-white tracking-tight">Fabrixe</div>
          <div className="text-[10px] text-slate-500 font-medium uppercase tracking-widest">Infrastructure</div>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto scrollbar-none">
        <p className="px-2 pb-2 text-[10px] font-semibold text-slate-600 uppercase tracking-widest">
          Navigation
        </p>
        {nav.map(({ to, icon: Icon, label, exact }) => (
          <NavLink
            key={to}
            to={to}
            end={exact}
            className={({ isActive }) =>
              clsx('sidebar-link', isActive && 'active')
            }
          >
            <Icon className="icon" />
            <span>{label}</span>
          </NavLink>
        ))}
      </nav>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-slate-800">
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
          <span className="text-xs text-slate-500">fabrixe.local</span>
        </div>
        <p className="text-[10px] text-slate-700 mt-1">v1.0.0 — Encrypted</p>
      </div>
    </aside>
  )
}
