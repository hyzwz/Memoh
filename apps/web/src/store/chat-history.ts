import {
  extractAllToolResults,
  extractMessageReasoning,
  extractMessageText,
  extractToolCalls,
} from '../composables/api/useChat.content'
import type { Message } from '../composables/api/useChat.types'
import type { AttachmentBlock, ChatMessage, ContentBlock, ToolCallBlock } from './chat-list'

const nextId = () => `${Date.now()}-${Math.floor(Math.random() * 1000)}`

function mediaTypeFromMime(mime: string): string {
  const m = (mime || '').toLowerCase().trim()
  if (m.startsWith('image/')) return 'image'
  if (m.startsWith('audio/')) return 'audio'
  if (m.startsWith('video/')) return 'video'
  return 'file'
}

function buildAssetBlocks(raw: Message): AttachmentBlock[] {
  if (!raw.assets?.length) return []
  const items: Array<Record<string, unknown>> = raw.assets.map((a) => ({
    type: mediaTypeFromMime(a.mime),
    content_hash: a.content_hash,
    bot_id: raw.bot_id,
    mime: a.mime,
    size: a.size_bytes,
    storage_key: a.storage_key,
  }))
  return [{ type: 'attachment', attachments: items }]
}

function resolveIsSelf(raw: Message): boolean {
  const platform = (raw.platform ?? '').trim().toLowerCase()
  if (!platform || platform === 'web') return true
  return false
}

function messageToChat(raw: Message): ChatMessage | null {
  if (raw.role !== 'user' && raw.role !== 'assistant') return null

  const text = extractMessageText(raw)
  const assetBlocks = buildAssetBlocks(raw)
  const reasoningTexts = extractMessageReasoning(raw)

  if (!text && assetBlocks.length === 0 && reasoningTexts.length === 0) return null

  const blocks: ContentBlock[] = []
  for (const r of reasoningTexts) {
    blocks.push({ type: 'thinking', content: r, done: true })
  }
  if (text) blocks.push({ type: 'text', content: text })
  blocks.push(...assetBlocks)

  const createdAt = raw.created_at ? new Date(raw.created_at) : new Date()
  const timestamp = Number.isNaN(createdAt.getTime()) ? new Date() : createdAt
  const platform = (raw.platform ?? '').trim().toLowerCase()
  const channelTag = platform && platform !== 'web' ? platform : undefined

  if (raw.role === 'user') {
    const isSelf = resolveIsSelf(raw)
    const senderName = (raw.sender_display_name ?? '').trim() || undefined
    const senderAvatar = (raw.sender_avatar_url ?? '').trim() || undefined
    return {
      id: raw.id || nextId(),
      role: 'user',
      blocks,
      timestamp,
      streaming: false,
      isSelf,
      ...(channelTag && { platform: channelTag }),
      ...(channelTag && { senderDisplayName: senderName, senderAvatarUrl: senderAvatar }),
    }
  }

  return {
    id: raw.id || nextId(),
    role: 'assistant',
    blocks,
    timestamp,
    streaming: false,
    ...(channelTag && { platform: channelTag }),
  }
}

export function convertMessagesToChats(rows: Message[]): ChatMessage[] {
  const result: ChatMessage[] = []
  let pendingAssistant: ChatMessage | null = null
  const pendingToolCallMap = new Map<string, ToolCallBlock>()

  function flushPending() {
    if (!pendingAssistant) return
    for (const block of pendingAssistant.blocks) {
      if (block.type === 'tool_call' && !block.done) block.done = true
    }
    result.push(pendingAssistant)
    pendingAssistant = null
    pendingToolCallMap.clear()
  }

  function makeTimestamp(raw: Message): Date {
    const d = raw.created_at ? new Date(raw.created_at) : new Date()
    return Number.isNaN(d.getTime()) ? new Date() : d
  }

  for (const raw of rows) {
    if (raw.role === 'user') {
      flushPending()
      const chat = messageToChat(raw)
      if (chat) result.push(chat)
      continue
    }

    if (raw.role === 'assistant') {
      const toolCalls = extractToolCalls(raw)
      const text = extractMessageText(raw)
      const reasoningTexts = extractMessageReasoning(raw)

      if (toolCalls.length > 0) {
        if (!pendingAssistant) {
          const platform = (raw.platform ?? '').trim().toLowerCase()
          const channelTag = platform && platform !== 'web' ? platform : undefined
          pendingAssistant = {
            id: raw.id || nextId(),
            role: 'assistant',
            blocks: [],
            timestamp: makeTimestamp(raw),
            streaming: false,
            ...(channelTag && { platform: channelTag }),
          }
        }
        for (const r of reasoningTexts) {
          pendingAssistant.blocks.push({ type: 'thinking', content: r, done: true })
        }
        if (text) pendingAssistant.blocks.push({ type: 'text', content: text })
        for (const tc of toolCalls) {
          const block: ToolCallBlock = {
            type: 'tool_call',
            toolCallId: tc.id ?? '',
            toolName: tc.name,
            input: tc.input,
            result: null,
            done: false,
          }
          pendingAssistant.blocks.push(block)
          if (tc.id) pendingToolCallMap.set(tc.id, block)
        }
        pendingAssistant.blocks.push(...buildAssetBlocks(raw))
        continue
      }

      if (pendingAssistant && text) {
        for (const r of reasoningTexts) {
          pendingAssistant.blocks.push({ type: 'thinking', content: r, done: true })
        }
        pendingAssistant.blocks.push({ type: 'text', content: text })
        pendingAssistant.blocks.push(...buildAssetBlocks(raw))
        flushPending()
        continue
      }

      flushPending()
      const chat = messageToChat(raw)
      if (chat) result.push(chat)
      continue
    }

    if (raw.role === 'tool') {
      const results = extractAllToolResults(raw)
      for (const r of results) {
        if (r.toolCallId && pendingToolCallMap.has(r.toolCallId)) {
          const block = pendingToolCallMap.get(r.toolCallId)!
          block.result = r.output
          block.done = true
        }
      }
    }
  }

  flushPending()
  return result
}
