import { describe, expect, it } from 'vitest'
import { extractAllToolResults } from './useChat.content'
import type { Message } from './useChat.types'

describe('useChat.content', () => {
  it('reads tool result payloads from result field when output is absent', () => {
    const message = {
      id: 'tool-1',
      role: 'tool',
      content: {
        content: [
          {
            type: 'tool-result',
            toolCallId: 'call-1',
            result: {
              stdout: 'ok',
              exit_code: 0,
            },
          },
        ],
      },
    } as Message

    expect(extractAllToolResults(message)).toEqual([
      {
        toolCallId: 'call-1',
        output: {
          stdout: 'ok',
          exit_code: 0,
        },
      },
    ])
  })
})
