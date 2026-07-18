import { useEffect, useRef, useState } from 'react'
import { FabrixeWS } from '../lib/websocket'
import type { SystemSnapshot } from '../types'
import { auth } from '../lib/api'

export function useLiveSnapshot(enabled = true) {
  const [snapshot, setSnapshot] = useState<SystemSnapshot | null>(null)
  const [connected, setConnected] = useState(false)
  const wsRef = useRef<FabrixeWS | null>(null)

  useEffect(() => {
    if (!enabled) return
    const token = auth.getToken()
    if (!token) return

    const ws = new FabrixeWS(token)
    wsRef.current = ws

    const offMsg = ws.onMessage(msg => {
      if (msg.type === 'snapshot') {
        setSnapshot(msg.payload)
      }
    })
    const offStatus = ws.onStatus(setConnected)

    ws.connect()

    return () => {
      offMsg()
      offStatus()
      ws.destroy()
    }
  }, [enabled])

  return { snapshot, connected }
}
