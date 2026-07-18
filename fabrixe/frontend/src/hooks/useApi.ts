import { useState, useEffect, useCallback, useRef } from 'react'

interface FetchState<T> {
  data: T | null
  loading: boolean
  error: string | null
}

export function useApi<T>(
  fetcher: () => Promise<T>,
  deps: unknown[] = [],
  options: { pollInterval?: number; enabled?: boolean } = {}
) {
  const { pollInterval, enabled = true } = options
  const [state, setState] = useState<FetchState<T>>({ data: null, loading: true, error: null })
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetch = useCallback(async () => {
    setState(s => ({ ...s, loading: true, error: null }))
    try {
      const data = await fetcher()
      setState({ data, loading: false, error: null })
    } catch (err) {
      setState({ data: null, loading: false, error: err instanceof Error ? err.message : 'Unknown error' })
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  useEffect(() => {
    if (!enabled) return
    fetch()
    if (pollInterval) {
      timerRef.current = setInterval(fetch, pollInterval)
    }
    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [fetch, pollInterval, enabled])

  return { ...state, refetch: fetch }
}
