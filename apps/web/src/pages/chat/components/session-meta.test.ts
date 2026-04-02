import { describe, expect, it } from 'vitest'
import { getSessionTypeMeta } from './session-meta'

describe('getSessionTypeMeta', () => {
  it('maps known session types to stable labels and icons', () => {
    expect(getSessionTypeMeta('heartbeat')).toMatchObject({ labelKey: 'chat.sessionType.heartbeat', icon: 'heartbeat' })
    expect(getSessionTypeMeta('schedule')).toMatchObject({ labelKey: 'chat.sessionType.schedule', icon: 'calendar' })
    expect(getSessionTypeMeta('subagent')).toMatchObject({ labelKey: 'chat.sessionType.subagent', icon: 'bot' })
  })

  it('falls back to chat defaults', () => {
    expect(getSessionTypeMeta('chat')).toMatchObject({ labelKey: 'chat.sessionType.chat', icon: 'message' })
    expect(getSessionTypeMeta('')).toMatchObject({ labelKey: 'chat.sessionType.chat', icon: 'message' })
  })
})
