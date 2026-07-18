import type { WsMessage } from '../types'

type MessageHandler = (msg: WsMessage) => void
type StatusHandler = (connected: boolean) => void

export class FabrixeWS {
  private ws: WebSocket | null = null
  private token: string
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 10
  private listeners: MessageHandler[] = []
  private statusListeners: StatusHandler[] = []
  private destroyed = false

  constructor(token: string) {
    this.token = token
  }

  connect(): void {
    if (this.destroyed) return
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${proto}://${location.host}/api/system/ws?token=${encodeURIComponent(this.token)}`

    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      this.reconnectAttempts = 0
      this.notifyStatus(true)
    }

    this.ws.onmessage = (event) => {
      try {
        const msg: WsMessage = JSON.parse(event.data)
        this.listeners.forEach(fn => fn(msg))
      } catch { /* ignore malformed frames */ }
    }

    this.ws.onerror = () => {
      // onclose will handle reconnect
    }

    this.ws.onclose = () => {
      this.notifyStatus(false)
      this.scheduleReconnect()
    }
  }

  private scheduleReconnect(): void {
    if (this.destroyed) return
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return
    const delay = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), 30000)
    this.reconnectAttempts++
    this.reconnectTimeout = setTimeout(() => this.connect(), delay)
  }

  onMessage(fn: MessageHandler): () => void {
    this.listeners.push(fn)
    return () => { this.listeners = this.listeners.filter(l => l !== fn) }
  }

  onStatus(fn: StatusHandler): () => void {
    this.statusListeners.push(fn)
    return () => { this.statusListeners = this.statusListeners.filter(l => l !== fn) }
  }

  private notifyStatus(connected: boolean): void {
    this.statusListeners.forEach(fn => fn(connected))
  }

  destroy(): void {
    this.destroyed = true
    if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout)
    this.ws?.close()
    this.ws = null
    this.listeners = []
    this.statusListeners = []
  }
}
