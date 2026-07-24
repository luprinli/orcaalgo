import { useState, useEffect, useRef, useCallback } from 'react'
import type { WSEnvelope } from '../types/ws'

export type ConnectionStatus = 'connecting' | 'connected' | 'reconnecting' | 'disconnected'

interface UseWebSocketOptions {
  channels?: string | string[]
  onMessage?: (data: unknown, channel: string) => void
  onError?: (err: Event) => void
  maxReconnects?: number
  reconnectInterval?: number
}

interface UseWebSocketReturn {
  status: ConnectionStatus
  connected: boolean
  lastMessage: unknown
  getData: (channel: string) => unknown
  send: (type: string, payload?: Record<string, unknown>) => void
}

interface Listener {
  id: number
  channels: string[]
  onMessage?: (data: unknown, channel: string) => void
}

function computeDelay(retryCount: number): number {
  const base = Math.min(1000 * Math.pow(2, retryCount - 1), 30000)
  const jitter = base * (Math.random() * 0.5 - 0.25)
  return Math.max(100, base + jitter)
}

let nextListenerId = 0

class WSConnectionManager {
  private ws: WebSocket | null = null
  private listeners: Map<number, Listener> = new Map()
  private retryCount = 0
  private connecting = false
  maxReconnects = 10
  private dataMap: Record<string, unknown> = {}
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private pingTimer: ReturnType<typeof setInterval> | null = null
  private statusListeners: Set<(status: ConnectionStatus) => void> = new Set()
  private status: ConnectionStatus = 'disconnected'
  private messageListeners: Set<(msg: unknown) => void> = new Set()

  private setStatus(status: ConnectionStatus) {
    this.status = status
    for (const fn of this.statusListeners) fn(status)
  }

  getStatus(): ConnectionStatus { return this.status }

  getData(channel: string): unknown { return this.dataMap[channel] ?? null }

  getActiveChannels(): string[] {
    const channels = new Set<string>()
    for (const listener of this.listeners.values()) {
      for (const ch of listener.channels) channels.add(ch)
    }
    return Array.from(channels)
  }

  register(listener: Listener): () => void {
    const id = listener.id || ++nextListenerId
    listener.id = id
    this.listeners.set(id, listener)
    this.ensureConnection()
    return () => this.unregister(id)
  }

  private unregister(id: number) {
    this.listeners.delete(id)
    if (this.listeners.size === 0) {
      this.close()
    } else {
      this.syncSubscriptions()
    }
  }

  onStatusChange(fn: (status: ConnectionStatus) => void): () => void {
    this.statusListeners.add(fn)
    fn(this.status)
    return () => { this.statusListeners.delete(fn) }
  }

  onLastMessage(fn: (msg: unknown) => void): () => void {
    this.messageListeners.add(fn)
    return () => { this.messageListeners.delete(fn) }
  }

  send(type: string, payload?: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, ...payload }))
    }
  }

  private syncSubscriptions() {
    if (this.ws?.readyState !== WebSocket.OPEN) return
    const active = this.getActiveChannels()
    for (const ch of active) {
      this.ws.send(JSON.stringify({ type: 'subscribe', channel: ch }))
    }
  }

  private ensureConnection() {
    if (this.listeners.size === 0) return
    if (this.connecting) return
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.syncSubscriptions()
      return
    }
    this.connect()
  }

  private connect() {
    if (this.connecting) return
    this.connecting = true
    const isRetry = this.retryCount > 0
    this.setStatus(isRetry ? 'reconnecting' : 'connecting')

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = import.meta.env.DEV ? 'localhost:8080' : window.location.host
    const raw = localStorage.getItem('orca_auth') || ''
    let token = ''
    try {
      const parsed = JSON.parse(raw)
      token = parsed.token || parsed.access_token || raw
    } catch {
      token = raw
    }
    const wsURL = `${protocol}//${host}/ws${token ? '?token=' + token : ''}`
    const ws = new WebSocket(wsURL)

    ws.onopen = () => {
      this.connecting = false
      this.retryCount = 0
      this.setStatus('connected')
      const active = this.getActiveChannels()
      for (const ch of active) {
        ws.send(JSON.stringify({ type: 'subscribe', channel: ch }))
      }
      this.pingTimer = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }))
        }
      }, 30000)
    }

    ws.onclose = () => {
      if (this.pingTimer) { clearInterval(this.pingTimer); this.pingTimer = null }
      this.connecting = false
      this.ws = null
      if (this.listeners.size > 0 && this.retryCount < this.maxReconnects) {
        this.retryCount++
        this.setStatus('reconnecting')
        this.reconnectTimer = setTimeout(() => this.ensureConnection(), computeDelay(this.retryCount))
      } else {
        this.setStatus('disconnected')
      }
    }

    ws.onerror = () => {
      this.connecting = false
    }

    ws.onmessage = (e) => {
      try {
        const envelope: WSEnvelope = JSON.parse(e.data)
        this.dataMap[envelope.channel] = envelope.data
        for (const listener of this.listeners.values()) {
          if (listener.channels.includes(envelope.channel)) {
            listener.onMessage?.(envelope.data, envelope.channel)
          }
        }
        for (const fn of this.messageListeners) fn(envelope.data)
      } catch { /* ignore malformed */ }
    }

    this.ws = ws
  }

  private close() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
    this.connecting = false
    this.retryCount = 0
    this.dataMap = {}
    this.setStatus('disconnected')
  }
}

const manager = new WSConnectionManager()

export function useWebSocket(channel: string, options?: UseWebSocketOptions): UseWebSocketReturn
export function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn
export function useWebSocket(
  channelOrOptions: string | UseWebSocketOptions,
  maybeOptions?: UseWebSocketOptions,
): UseWebSocketReturn {
  const resolvedOptions: UseWebSocketOptions & { channels: string[] } =
    typeof channelOrOptions === 'string'
      ? { ...maybeOptions, channels: [channelOrOptions] }
      : {
          ...channelOrOptions,
          channels: channelOrOptions.channels
            ? (Array.isArray(channelOrOptions.channels)
                ? channelOrOptions.channels
                : [channelOrOptions.channels])
            : [],
        }

  return useWS(resolvedOptions)
}

export function useWebSocketMulti(
  channels: string[],
  options?: UseWebSocketOptions,
): UseWebSocketReturn {
  return useWS({ ...options, channels })
}

function useWS(options: UseWebSocketOptions & { channels: string[] }): UseWebSocketReturn {
  const { channels, onMessage, maxReconnects = 10 } = options

  const [status, setStatus] = useState<ConnectionStatus>(() => manager.getStatus())
  const [lastMessage, setLastMessage] = useState<unknown>(null)
  const channelsRef = useRef(channels)
  channelsRef.current = channels
  const unregisterRef = useRef<(() => void) | null>(null)

  useEffect(() => {
    manager.maxReconnects = maxReconnects

    const listener: Listener = {
      id: 0,
      channels,
      onMessage: (data, channel) => {
        if (channelsRef.current.includes(channel)) {
          onMessage?.(data, channel)
        }
      },
    }

    const unregister = manager.register(listener)
    unregisterRef.current = unregister
    const unsubStatus = manager.onStatusChange(setStatus)
    const unsubMsg = manager.onLastMessage(setLastMessage)

    return () => {
      unsubStatus()
      unsubMsg()
      unregister()
      unregisterRef.current = null
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const send = useCallback((type: string, payload?: Record<string, unknown>) => {
    manager.send(type, payload)
  }, [])

  const getData = useCallback((ch: string) => manager.getData(ch), [])

  return {
    status,
    connected: status === 'connected',
    lastMessage,
    getData,
    send,
  }
}
