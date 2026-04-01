import { describe, expect, it } from 'vitest'
import type { Message } from '@/composables/api/useChat.types'
import { convertMessagesToChats } from './chat-history'

describe('chat-list store message conversion', () => {
  it('keeps tool results when persisted under result field', () => {
    const rows = [
      {
        id: 'assistant-tool',
        bot_id: 'bot-1',
        role: 'assistant',
        content: {
          content: [
            {
              type: 'tool-call',
              toolCallId: 'call-1',
              toolName: 'exec',
              input: { command: 'pwd' },
            },
          ],
        },
        created_at: '2026-04-01T00:00:00Z',
      },
      {
        id: 'tool-result',
        bot_id: 'bot-1',
        role: 'tool',
        content: {
          content: [
            {
              type: 'tool-result',
              toolCallId: 'call-1',
              result: { stdout: '/tmp', exit_code: 0 },
            },
          ],
        },
        created_at: '2026-04-01T00:00:01Z',
      },
      {
        id: 'assistant-final',
        bot_id: 'bot-1',
        role: 'assistant',
        content: {
          content: [
            {
              type: 'text',
              text: 'done',
            },
          ],
        },
        created_at: '2026-04-01T00:00:02Z',
      },
    ] as Message[]

    const chats = convertMessagesToChats(rows)
    expect(chats).toHaveLength(1)
    expect(chats[0]?.blocks).toContainEqual({
      type: 'tool_call',
      toolCallId: 'call-1',
      toolName: 'exec',
      input: { command: 'pwd' },
      result: { stdout: '/tmp', exit_code: 0 },
      done: true,
    })
  })
})
