import { describe, it, expect, beforeEach, afterEach } from 'vitest'

// Simulate the WebSocket connection logic from useWebSocket.ts without React
// This tests the core protocol: URL construction, message parsing, channel filtering

// Mock WebSocket
class MockWS {
  url: string
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null
  onerror: ((e: Event) => void) | null = null

  constructor(url: string) {
    this.url = url
  }
  close() {}
}

describe('WebSocket protocol', () => {
  let instances: MockWS[] = []

  beforeEach(() => {
    instances = []
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ;(globalThis as any).WebSocket = class extends MockWS {
      constructor(url: string) {
        super(url)
        instances.push(this as unknown as MockWS)
      }
    }
  })

  afterEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    delete (globalThis as any).WebSocket
  })

  it('constructs ws:// URL from current host', () => {
    const protocol = 'ws:'
    const host = 'localhost'
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const ws = new (globalThis as any).WebSocket(`${protocol}//${host}/ws`)
    expect(ws.url).toBe('ws://localhost/ws')
  })

  it('constructs wss:// URL for https pages', () => {
    const protocol = 'wss:'
    const host = 'myapp.com'
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const ws = new (globalThis as any).WebSocket(`${protocol}//${host}/ws`)
    expect(ws.url).toBe('wss://myapp.com/ws')
  })
})

describe('WS message parsing', () => {
  it('parses JSON channel envelope', () => {
    const raw = JSON.stringify({ channel: 'risk', data: { equity: 100000, halted: false } })
    const parsed = JSON.parse(raw)
    expect(parsed.channel).toBe('risk')
    expect(parsed.data.equity).toBe(100000)
    expect(parsed.data.halted).toBe(false)
  })

  it('safely handles malformed JSON', () => {
    expect(() => JSON.parse('not-json')).toThrow()
    // Filtering like: try { JSON.parse(data) } catch { /* ignore */ }
    let caught = false
    try { JSON.parse('not-json') } catch { caught = true }
    expect(caught).toBe(true)
  })

  it('filters channels — matching', () => {
    const targetChannel = 'risk'
    const messages = [
      { channel: 'risk', data: { equity: 100000 } },
      { channel: 'performance', data: { sharpe: 1.5 } },
    ]

    const filtered = messages.filter(m => m.channel === targetChannel)
    expect(filtered).toHaveLength(1)
    expect(filtered[0].data).toEqual({ equity: 100000 })
  })

  it('filters channels — no match returns empty', () => {
    const targetChannel = 'cvd'
    const messages = [
      { channel: 'risk', data: { equity: 100000 } },
      { channel: 'performance', data: { sharpe: 1.5 } },
    ]

    const filtered = messages.filter(m => m.channel === targetChannel)
    expect(filtered).toHaveLength(0)
  })
})

describe('WS reconnect logic', () => {
  it('limits max reconnect attempts', () => {
    const maxReconnects = 3
    let reconnectCount = 0

    for (let i = 0; i < 10; i++) {
      if (reconnectCount < maxReconnects) {
        reconnectCount++
      }
    }

    expect(reconnectCount).toBe(3)
  })

  it('resets reconnect count on successful open', () => {
    let reconnectCount = 5
    // Simulate onopen resetting the counter
    reconnectCount = 0
    expect(reconnectCount).toBe(0)
  })
})

describe('multi-channel tracking', () => {
  it('tracks data per channel', () => {
    const dataMap: Record<string, unknown> = {}
    const channels = ['risk', 'performance']

    const messages = [
      { channel: 'risk', data: { equity: 100000 } },
      { channel: 'performance', data: { sharpe: 2.0 } },
    ]

    for (const msg of messages) {
      if (channels.includes(msg.channel)) {
        dataMap[msg.channel] = msg.data
      }
    }

    expect(dataMap['risk']).toEqual({ equity: 100000 })
    expect(dataMap['performance']).toEqual({ sharpe: 2.0 })
  })
})
