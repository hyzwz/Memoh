import { describe, expect, it } from 'vitest'
import { isSessionReadOnly } from './session-readonly'

describe('isSessionReadOnly', () => {
  it('treats heartbeat, schedule, and subagent sessions as read-only', () => {
    expect(isSessionReadOnly({ type: 'heartbeat' })).toBe(true)
    expect(isSessionReadOnly({ type: 'schedule' })).toBe(true)
    expect(isSessionReadOnly({ type: 'subagent' })).toBe(true)
  })

  it('treats non-web sessions as read-only', () => {
    expect(isSessionReadOnly({ type: 'chat', channel_type: 'telegram' })).toBe(true)
  })

  it('keeps normal web chat sessions writable', () => {
    expect(isSessionReadOnly({ type: 'chat', channel_type: 'web' })).toBe(false)
    expect(isSessionReadOnly({})).toBe(false)
  })
})
