import type { Bot, ChatSummary } from './useChat.types'
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

export async function fetchBots(): Promise<Bot[]> {
  const { baseUrl, token } = getApiBase()
  const res = await fetch(`${baseUrl}/bots`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`fetchBots failed: ${res.status}`)
  const data = await res.json()
  return data?.items ?? []
}

export async function fetchChats(botId: string): Promise<ChatSummary[]> {
  const id = botId.trim()
  if (!id) return []
  return [{ id, bot_id: id, kind: 'bot' }]
}

export async function createChat(botId: string): Promise<ChatSummary> {
  const id = botId.trim()
  if (!id) throw new Error('bot id is required')
  return { id, bot_id: id, kind: 'bot' }
}

export async function deleteChat(botId: string, _chatId: string): Promise<void> {
  const { baseUrl, token } = getApiBase()
  const res = await fetch(`${baseUrl}/bots/${encodeURIComponent(botId)}/messages`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`deleteChat failed: ${res.status}`)
}
