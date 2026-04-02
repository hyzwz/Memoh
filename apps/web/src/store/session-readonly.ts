export interface SessionLike {
  type?: string
  channel_type?: string
  access_mode?: string
}

const READ_ONLY_TYPES = new Set(['heartbeat', 'schedule', 'subagent'])

export function isSessionReadOnly(session: SessionLike | null | undefined): boolean {
  if (!session) return false
  if (session.access_mode === 'channel_identity_observed') return true
  if (READ_ONLY_TYPES.has((session.type ?? '').trim())) return true
  const channelType = (session.channel_type ?? '').trim()
  if (channelType && channelType !== 'web') return true
  return false
}
