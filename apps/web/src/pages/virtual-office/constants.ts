export const SVG_WIDTH = 1200
export const SVG_HEIGHT = 700

export const OFFICE = {
  x: 30,
  y: 20,
  width: SVG_WIDTH - 60,
  height: SVG_HEIGHT - 40,
  wallThickness: 6,
  cornerRadius: 18,
  corridorWidth: 28,
} as const

const halfW = (OFFICE.width - OFFICE.corridorWidth) / 2
const halfH = (OFFICE.height - OFFICE.corridorWidth) / 2
const rightX = OFFICE.x + halfW + OFFICE.corridorWidth
const bottomY = OFFICE.y + halfH + OFFICE.corridorWidth

export const ZONES = {
  desk: { x: OFFICE.x, y: OFFICE.y, width: halfW, height: halfH, label: '固定工位区' },
  meeting: { x: rightX, y: OFFICE.y, width: halfW, height: halfH, label: '会议区' },
  hotDesk: { x: OFFICE.x, y: bottomY, width: halfW, height: halfH, label: '热工位区' },
  lounge: { x: rightX, y: bottomY, width: halfW, height: halfH, label: '休息区' },
} as const

export type ZoneKey = keyof typeof ZONES

export const CORRIDOR_CENTER = {
  x: OFFICE.x + OFFICE.width / 2,
  y: OFFICE.y + OFFICE.height / 2,
} as const

export const ZONE_COLORS = {
  desk: '#f4f6f9',
  meeting: '#eef3fa',
  hotDesk: '#f1f3f7',
  lounge: '#f3f1f7',
  corridor: '#e8ecf1',
  wall: '#8b9bb0',
} as const

export const ZONE_COLORS_DARK = {
  desk: '#1e293b',
  meeting: '#1a2744',
  hotDesk: '#1e2433',
  lounge: '#231e33',
  corridor: '#0f172a',
  wall: '#475569',
} as const

export type AgentVisualStatus = 'idle' | 'thinking' | 'tool_calling' | 'speaking' | 'spawning' | 'error' | 'offline'

export const STATUS_COLORS: Record<AgentVisualStatus, string> = {
  idle: '#22c55e',
  thinking: '#3b82f6',
  tool_calling: '#f97316',
  speaking: '#a855f7',
  spawning: '#06b6d4',
  error: '#ef4444',
  offline: '#6b7280',
}

export interface VisualAgent {
  id: string
  name: string
  status: AgentVisualStatus
  position: { x: number; y: number }
  zone: ZoneKey | 'corridor'
  isSubAgent: boolean
  confirmed: boolean
  currentTool: string | null
}

export const AVATAR = {
  radius: 20,
  selectedRadius: 24,
  strokeWidth: 3,
} as const
