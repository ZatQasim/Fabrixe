import { clsx } from 'clsx'

export default function Spinner({ size = 'md', className }: { size?: 'sm' | 'md' | 'lg'; className?: string }) {
  const sz = { sm: 'w-4 h-4', md: 'w-6 h-6', lg: 'w-8 h-8' }
  return (
    <div className={clsx('border-2 border-fabrixe-600 border-t-transparent rounded-full animate-spin', sz[size], className)} />
  )
}

export function PageLoader() {
  return (
    <div className="flex items-center justify-center py-24">
      <Spinner size="lg" />
    </div>
  )
}
