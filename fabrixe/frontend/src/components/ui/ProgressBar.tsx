import { clsx } from 'clsx'
import { usageBarColor } from '../../lib/format'

interface Props {
  value: number   // 0–100
  size?: 'sm' | 'md'
  color?: string  // override tailwind class
  showLabel?: boolean
  label?: string
}

export default function ProgressBar({ value, size = 'md', color, showLabel, label }: Props) {
  const clamped = Math.max(0, Math.min(100, value))
  const barColor = color ?? usageBarColor(clamped)

  return (
    <div className="w-full">
      {(showLabel || label) && (
        <div className="flex justify-between mb-1">
          {label && <span className="text-xs text-slate-400">{label}</span>}
          {showLabel && <span className="text-xs font-medium text-slate-300">{clamped.toFixed(1)}%</span>}
        </div>
      )}
      <div className={clsx('w-full bg-slate-800 rounded-full overflow-hidden', size === 'sm' ? 'h-1.5' : 'h-2')}>
        <div
          className={clsx('h-full rounded-full transition-all duration-500', barColor)}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  )
}
