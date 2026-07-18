import { type LucideIcon } from 'lucide-react'
import { clsx } from 'clsx'

interface Props {
  label: string
  value: string
  sub?: string
  icon: LucideIcon
  iconColor?: string
  trend?: 'up' | 'down' | 'neutral'
  alert?: boolean
}

export default function MetricCard({ label, value, sub, icon: Icon, iconColor = 'text-fabrixe-400', alert }: Props) {
  return (
    <div className={clsx('card-sm flex flex-col gap-3', alert && 'border-red-500/30 bg-red-500/5')}>
      <div className="flex items-start justify-between">
        <span className="text-xs font-medium text-slate-500 uppercase tracking-wider">{label}</span>
        <div className={clsx('w-8 h-8 rounded-lg bg-slate-800 flex items-center justify-center shrink-0', alert && 'bg-red-500/10')}>
          <Icon className={clsx('w-4 h-4', alert ? 'text-red-400' : iconColor)} />
        </div>
      </div>
      <div>
        <p className={clsx('text-2xl font-bold', alert ? 'text-red-300' : 'text-slate-100')}>{value}</p>
        {sub && <p className="text-xs text-slate-500 mt-0.5">{sub}</p>}
      </div>
    </div>
  )
}
