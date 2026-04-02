export interface SessionTypeMeta {
  labelKey: string
  icon: 'message' | 'heartbeat' | 'calendar' | 'bot'
}

export function getSessionTypeMeta(type?: string | null): SessionTypeMeta {
  switch ((type ?? '').trim()) {
    case 'heartbeat':
      return { labelKey: 'chat.sessionType.heartbeat', icon: 'heartbeat' }
    case 'schedule':
      return { labelKey: 'chat.sessionType.schedule', icon: 'calendar' }
    case 'subagent':
      return { labelKey: 'chat.sessionType.subagent', icon: 'bot' }
    default:
      return { labelKey: 'chat.sessionType.chat', icon: 'message' }
  }
}
