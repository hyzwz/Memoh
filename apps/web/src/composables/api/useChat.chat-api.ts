import {
  deleteBotsByBotIdSessionsById,
  getBots,
  getBotsByBotIdSessions,
  postBotsByBotIdSessions,
} from '@memoh/sdk'
import type { Bot, ChatSummary } from './useChat.types'

export async function fetchBots(): Promise<Bot[]> {
  const { data } = await getBots({ throwOnError: true })
  return data?.items ?? []
}

export async function fetchChats(botId: string): Promise<ChatSummary[]> {
  const id = botId.trim()
  if (!id) return []
  const { data } = await getBotsByBotIdSessions({
    path: { bot_id: id },
    throwOnError: true,
  })
  return data?.items ?? []
}

export async function createChat(botId: string): Promise<ChatSummary> {
  const id = botId.trim()
  if (!id) throw new Error('bot id is required')
  const { data } = await postBotsByBotIdSessions({
    path: { bot_id: id },
    body: { kind: 'direct' },
    throwOnError: true,
  })
  return data
}

export async function deleteChat(botId: string, chatId: string): Promise<void> {
  const bid = botId.trim()
  const cid = chatId.trim()
  if (!bid) throw new Error('bot id is required')
  if (!cid) throw new Error('chat id is required')
  await deleteBotsByBotIdSessionsById({
    path: { bot_id: bid, id: cid },
    throwOnError: true,
  })
}
