import type { StreamEvent, MessageStreamEvent, ChatAttachment, StreamEventHandler } from './useChat.types'
import { useAppStore } from '@/stores/app'

export interface WSClientMessage {
  type: 'message' | 'abort'
  text?: string
  attachments?: ChatAttachment[]
}

export interface ChatWebSocket {
  send: (msg: WSClientMessage) => void
  abort: () => void
  close: () => void
  readonly connected: boolean
  onOpen: (() => void) | null
  onClose: (() => void) | null
}

const isDev = import.meta.env.DEV

function logChatWs(event: string, data: Record<string, unknown>) {
  console.log(`[desktop-chat-ws:${event}]`, data)
}

function resolveWebSocketUrl(botId: string): string {
  if (isDev) {
    // In dev mode, use Vite's WebSocket proxy (same host, /api prefix stripped by proxy)
    const path = `/api/bots/${encodeURIComponent(botId)}/web/ws`
    const loc = window.location
    const proto = loc.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${loc.host}${path}`
  }

  // In production (Wails embedded), call the server directly (no /api prefix)
  const path = `/bots/${encodeURIComponent(botId)}/web/ws`
  const store = useAppStore()
  const serverURL = store.serverURL.replace(/\/+$/, '')

  if (!serverURL) {
    const loc = window.location
    const proto = loc.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${loc.host}${path}`
  }

  try {
    const url = new URL(path, serverURL)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    return url.toString()
  } catch {
    const loc = window.location
    const proto = loc.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${loc.host}${path}`
  }
}

export function connectWebSocket(
  botId: string,
  onStreamEvent: StreamEventHandler,
  onMessageEvent?: (event: MessageStreamEvent) => void,
): ChatWebSocket {
  const id = botId.trim()
  if (!id) throw new Error('bot id is required')

  const store = useAppStore()
  const wsUrl = resolveWebSocketUrl(id)
  const token = store.token
  const url = token ? `${wsUrl}?token=${encodeURIComponent(token)}` : wsUrl

  logChatWs('connect-init', {
    botId: id,
    wsUrl,
    finalUrl: url,
    isDev,
    serverURL: store.serverURL,
    tokenPrefix: token ? token.slice(0, 16) : '',
  })

  let ws: WebSocket | null = null
  let isConnected = false
  let closed = false
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectDelay = 1000

  const handle: ChatWebSocket = {
    send(msg: WSClientMessage) {
      logChatWs('send', {
        botId: id,
        connected: isConnected,
        readyState: ws?.readyState ?? null,
        type: msg.type,
      })
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(msg))
      }
    },
    abort() {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'abort' }))
      }
    },
    close() {
      closed = true
      if (reconnectTimer) {
        clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
      if (ws) {
        ws.close()
        ws = null
      }
      isConnected = false
    },
    get connected() {
      return isConnected
    },
    onOpen: null,
    onClose: null,
  }

  function connect() {
    if (closed) return
    logChatWs('connect-attempt', {
      botId: id,
      finalUrl: url,
      reconnectDelay,
    })
    ws = new WebSocket(url)

    ws.onopen = () => {
      isConnected = true
      reconnectDelay = 1000
      logChatWs('open', {
        botId: id,
        finalUrl: url,
      })
      handle.onOpen?.()
    }

    ws.onclose = (event) => {
      isConnected = false
      logChatWs('close', {
        botId: id,
        code: event.code,
        reason: event.reason,
        wasClean: event.wasClean,
      })
      handle.onClose?.()
      scheduleReconnect()
    }

    ws.onerror = () => {
      logChatWs('error', {
        botId: id,
        finalUrl: url,
        readyState: ws?.readyState ?? null,
      })
    }

    ws.onmessage = (event) => {
      if (typeof event.data !== 'string') return
      try {
        const parsed = JSON.parse(event.data)
        if (!parsed || typeof parsed !== 'object') return

        const eventType = String(parsed.type ?? '').trim()
        if (eventType === 'message_created' && onMessageEvent) {
          onMessageEvent(parsed as MessageStreamEvent)
          return
        }

        onStreamEvent(parsed as StreamEvent)
      } catch {
        // Ignore unparsable messages
      }
    }
  }

  function scheduleReconnect() {
    if (closed) return
    logChatWs('schedule-reconnect', {
      botId: id,
      reconnectDelay,
    })
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, reconnectDelay)
    reconnectDelay = Math.min(reconnectDelay * 1.5, 10000)
  }

  connect()
  return handle
}
