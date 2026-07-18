import { clsx } from 'clsx'

type Status = 'online' | 'offline' | 'warning' | 'unknown' | 'running' | 'idle' | 'success' | 'failed' | 'pending'

const colors: Record<Status, string> = {
  online:  'bg-emerald-500',
  running: 'bg-emerald-500',
  success: 'bg-emerald-500',
  offline: 'bg-red-500',
  failed:  'bg-red-500',
  warning: 'bg-amber-500',
  unknown: 'bg-slate-500',
  idle:    'bg-slate-500',
  pending: 'bg-blue-500',
}

interface Props {
  status: Status | string
  pulse?: boolean
  size?: 'sm' | 'md'
}

export default function StatusDot({ status, pulse, size = 'sm' }: Props) {
  const color = colors[status as Status] ?? 'bg-slate-500'
  const sz = size === 'sm' ? 'w-2 h-2' : 'w-2.5 h-2.5'
  return (
    <span className={clsx('inline-block rounded-full shrink-0', sz, color, pulse && 'animate-pulse')} />
  )
}

export function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    online:  'badge-online',
    running: 'badge-online',
    success: 'badge-online',
    active:  'badge-online',
    offline: 'badge-offline',
    failed:  'badge-offline',
    inactive:'badge-offline',
    warning: 'badge-warning',
    pending: 'badge-info',
    idle:    'badge-neutral',
    unknown: 'badge-neutral',
  }
  const cls = map[status.toLowerCase()] ?? 'badge-neutral'
  return (
    <span className={cls}>
      <StatusDot status={status as Status} />
      {status}
    </span>
  )
}
