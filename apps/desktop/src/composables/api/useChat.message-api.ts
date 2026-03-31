import type {
  ChatAttachment,
  FetchMessagesOptions,
  Message,
  MessageStreamEvent,
  StreamEventHandler,
} from './useChat.types'
import { parseStreamPayload, readSSEStream } from './useChat.sse'
import { useAppStore } from '@/stores/app'

const isDev = import.meta.env.DEV

function getApiBase(): { baseUrl: string; token: string } {
  const store = useAppStore()
  if (isDev) {
    // In dev mode, use Vite proxy (relative URL — proxy strips /api prefix)
    return { baseUrl: '/api', token: store.token }
  }
  // In production (Wails embedded), call the server directly (no /api prefix)
  const baseUrl = store.serverURL.replace(/\/+$/, '')
  return { baseUrl, token: store.token }
}

function logChatApi(event: string, data: Record<string, unknown>) {
  console.log(`[desktop-chat:${event}]`, data)
}

/**
 * Notify the Vite dev proxy of the actual server URL (dev mode only).
 */
export async function setProxyTarget(serverURL: string): Promise<void> {
  if (!isDev) return
  try {
    await fetch('/__set_proxy_target', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target: serverURL }),
    })
    logChatApi('proxy-target-set', { serverURL })
  } catch (error) {
    logChatApi('proxy-target-set-failed', {
      serverURL,
      error: error instanceof Error ? error.message : String(error),
    })
  }
}

export async function fetchMessages(
  botId: string,
  _chatId: string,
  options?: FetchMessagesOptions,
): Promise<Message[]> {
  const { baseUrl, token } = getApiBase()
  const params = new URLSearchParams()
  params.set('limit', String(options?.limit ?? 30))
  if (options?.before?.trim()) params.set('before', options.before.trim())

  const res = await fetch(`${baseUrl}/bots/${encodeURIComponent(botId)}/messages?${params}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`fetchMessages failed: ${res.status}`)
  const data = await res.json()
  return data?.items ?? []
}

export async function sendLocalChannelMessage(
  botId: string,
  text: string,
  attachments?: ChatAttachment[],
): Promise<void> {
  const store = useAppStore()
  const { baseUrl, token } = getApiBase()
  const msg: Record<string, unknown> = {}
  const trimmedText = text.trim()
  if (trimmedText) msg.text = trimmedText
  if (attachments?.length) {
    msg.attachments = attachments.map((item) => ({
      type: item.type,
      base64: item.base64,
      mime: item.mime ?? '',
      name: item.name ?? '',
    }))
  }

  const url = `${baseUrl}/bots/${encodeURIComponent(botId)}/web/messages`
  logChatApi('send-request', {
    url,
    botId,
    baseUrl,
    isDev,
    serverURL: store.serverURL,
    tokenPrefix: token ? token.slice(0, 16) : '',
    hasAttachments: !!attachments?.length,
  })

  const res = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ message: msg }),
  })

  if (!res.ok) {
    const body = await res.text().catch(() => '')
    logChatApi('send-response-error', {
      url,
      status: res.status,
      statusText: res.statusText,
      body,
    })
    throw new Error(`sendLocalChannelMessage failed: ${res.status}${body ? ` ${body}` : ''}`)
  }

  logChatApi('send-response-ok', {
    url,
    status: res.status,
  })
}

export async function streamLocalChannel(
  botId: string,
  signal: AbortSignal,
  onEvent: StreamEventHandler,
): Promise<void> {
  const id = botId.trim()
  if (!id) throw new Error('bot id is required')

  const { baseUrl, token } = getApiBase()
  const res = await fetch(`${baseUrl}/bots/${encodeURIComponent(id)}/web/stream`, {
    signal,
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`streamLocalChannel failed: ${res.status}`)
  const body = res.body
  if (!body) throw new Error('No response body')

  await readSSEStream(body, (payload) => {
    const event = parseStreamPayload(payload)
    if (event) onEvent(event)
  })
}

export async function streamMessageEvents(
  botId: string,
  signal: AbortSignal,
  onEvent: (event: MessageStreamEvent) => void,
  since?: string,
): Promise<void> {
  const id = botId.trim()
  if (!id) throw new Error('bot id is required')

  const { baseUrl, token } = getApiBase()
  const params = new URLSearchParams()
  if (since?.trim()) params.set('since', since.trim())
  const qs = params.toString() ? `?${params}` : ''

  const res = await fetch(`${baseUrl}/bots/${encodeURIComponent(id)}/messages/events${qs}`, {
    signal,
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`streamMessageEvents failed: ${res.status}`)
  const body = res.body
  if (!body) throw new Error('No response body')

  await readSSEStream(body, (payload) => {
    const parsed = parseStreamPayload(payload)
    if (!parsed || typeof parsed !== 'object' || !('type' in parsed)) return
    if (typeof parsed.type !== 'string' || !parsed.type.trim()) return
    onEvent(parsed as MessageStreamEvent)
  })
}
