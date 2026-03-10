<template>
  <div class="relative h-full w-full">
    <svg
      :viewBox="`0 0 ${SVG_WIDTH} ${SVG_HEIGHT}`"
      class="h-full w-full"
      preserveAspectRatio="xMidYMid meet"
    >
      <defs>
        <filter
          id="building-shadow"
          x="-3%"
          y="-3%"
          width="106%"
          height="106%"
        >
          <feDropShadow
            dx="0"
            dy="3"
            stdDeviation="6"
            :flood-opacity="isDark ? 0.5 : 0.12"
          />
        </filter>
        <pattern
          id="corridor-tiles"
          width="28"
          height="28"
          patternUnits="userSpaceOnUse"
        >
          <rect
            width="28"
            height="28"
            :fill="colors.corridor"
          />
          <rect
            x="0.5"
            y="0.5"
            width="27"
            height="27"
            fill="none"
            :stroke="isDark ? '#1f2937' : '#d5dbe3'"
            stroke-width="0.3"
            rx="1"
          />
        </pattern>
        <pattern
          id="lounge-carpet"
          width="6"
          height="6"
          patternUnits="userSpaceOnUse"
        >
          <rect
            width="6"
            height="6"
            :fill="colors.lounge"
          />
          <circle
            cx="3"
            cy="3"
            r="0.5"
            :fill="isDark ? '#2d2540' : '#e5e0ed'"
            opacity="0.4"
          />
        </pattern>
      </defs>

      <!-- Layer 0: Building shell -->
      <rect
        :x="OFFICE.x"
        :y="OFFICE.y"
        :width="OFFICE.width"
        :height="OFFICE.height"
        :rx="OFFICE.cornerRadius"
        :fill="colors.corridor"
        :stroke="colors.wall"
        :stroke-width="OFFICE.wallThickness"
        filter="url(#building-shadow)"
      />

      <!-- Layer 1: Corridors -->
      <rect
        :x="OFFICE.x"
        :y="midY"
        :width="OFFICE.width"
        :height="cw"
        fill="url(#corridor-tiles)"
      />
      <rect
        :x="midX"
        :y="OFFICE.y"
        :width="cw"
        :height="OFFICE.height"
        fill="url(#corridor-tiles)"
      />
      <!-- Corridor guide lines -->
      <line
        :x1="OFFICE.x"
        :y1="midY + cw / 2"
        :x2="OFFICE.x + OFFICE.width"
        :y2="midY + cw / 2"
        :stroke="isDark ? '#334155' : '#c8d0dc'"
        stroke-width="0.5"
        stroke-dasharray="8 6"
        opacity="0.6"
      />
      <line
        :x1="midX + cw / 2"
        :y1="OFFICE.y"
        :x2="midX + cw / 2"
        :y2="OFFICE.y + OFFICE.height"
        :stroke="isDark ? '#334155' : '#c8d0dc'"
        stroke-width="0.5"
        stroke-dasharray="8 6"
        opacity="0.6"
      />

      <!-- Layer 2: Zone floors -->
      <rect
        v-for="(zone, key) in ZONES"
        :key="`floor-${key}`"
        :x="zone.x"
        :y="zone.y"
        :width="zone.width"
        :height="zone.height"
        :fill="key === 'lounge' ? 'url(#lounge-carpet)' : colors[key as keyof typeof ZONE_COLORS]"
      />

      <!-- Layer 3: Partition walls -->
      <rect
        v-for="(w, i) in walls"
        :key="`wall-${i}`"
        :x="w.x"
        :y="w.y"
        :width="w.w"
        :height="w.h"
        :fill="isDark ? '#334155' : '#c8d0dc'"
        :stroke="isDark ? '#475569' : '#8b9bb0'"
        stroke-width="0.5"
      />

      <!-- Layer 4: Door openings -->
      <g
        v-for="(d, i) in doors"
        :key="`door-${i}`"
      >
        <template v-if="d.horizontal">
          <rect
            :x="d.cx - 20"
            :y="d.cy - 3"
            width="40"
            height="6"
            :fill="doorColor"
          />
          <path
            :d="`M ${d.cx - 20} ${d.cy} A 20 20 0 0 1 ${d.cx + 20} ${d.cy}`"
            fill="none"
            :stroke="arcColor"
            stroke-width="0.8"
            stroke-dasharray="3 2"
            opacity="0.5"
          />
        </template>
        <template v-else>
          <rect
            :x="d.cx - 3"
            :y="d.cy - 20"
            width="6"
            height="40"
            :fill="doorColor"
          />
          <path
            :d="`M ${d.cx} ${d.cy - 20} A 20 20 0 0 1 ${d.cx} ${d.cy + 20}`"
            fill="none"
            :stroke="arcColor"
            stroke-width="0.8"
            stroke-dasharray="3 2"
            opacity="0.5"
          />
        </template>
      </g>

      <!-- Zone labels -->
      <text
        v-for="(zone, key) in ZONES"
        :key="`label-${key}`"
        :x="zone.x + 14"
        :y="zone.y + 22"
        :fill="isDark ? '#64748b' : '#94a3b8'"
        font-size="11"
        font-weight="600"
        font-family="system-ui, sans-serif"
      >
        {{ zone.label }}
      </text>

      <!-- Layer 5: Desk zone furniture -->
      <g
        v-for="(slot, i) in deskSlots"
        :key="`desk-${i}`"
      >
        <Furniture
          type="desk"
          :x="slot.x"
          :y="slot.y + 30"
          :is-dark="isDark"
        />
        <Furniture
          type="chair"
          :x="slot.x"
          :y="slot.y - 12"
          :is-dark="isDark"
        />
        <AgentAvatar
          v-if="deskAgentMap.get(i)"
          :agent="deskAgentMap.get(i)!"
          @select="$emit('selectAgent', $event)"
        />
      </g>

      <!-- Layer 5: Meeting zone -->
      <Furniture
        type="meetingTable"
        :x="meetingCenter.x"
        :y="meetingCenter.y"
        :radius="meetingTableRadius"
        :is-dark="isDark"
      />
      <Furniture
        v-for="(seat, i) in meetingChairs"
        :key="`mc-${i}`"
        type="chair"
        :x="seat.x"
        :y="seat.y"
        :is-dark="isDark"
      />

      <!-- Layer 5: Hot desk zone -->
      <g
        v-for="(slot, i) in hotDeskSlots"
        :key="`hotdesk-${i}`"
      >
        <Furniture
          type="desk"
          :x="slot.x"
          :y="slot.y + 30"
          :is-dark="isDark"
        />
        <Furniture
          type="chair"
          :x="slot.x"
          :y="slot.y - 12"
          :is-dark="isDark"
        />
        <AgentAvatar
          v-if="hotDeskAgents[i]"
          :agent="hotDeskAgents[i]"
          @select="$emit('selectAgent', $event)"
        />
      </g>

      <!-- Layer 5: Lounge decor -->
      <Furniture
        type="sofa"
        :x="ZONES.lounge.x + 100"
        :y="ZONES.lounge.y + 60"
        :rotation="0"
        :is-dark="isDark"
      />
      <Furniture
        type="sofa"
        :x="ZONES.lounge.x + 280"
        :y="ZONES.lounge.y + 60"
        :rotation="0"
        :is-dark="isDark"
      />
      <Furniture
        type="sofa"
        :x="ZONES.lounge.x + 100"
        :y="ZONES.lounge.y + 140"
        :rotation="180"
        :is-dark="isDark"
      />
      <Furniture
        type="coffeeCup"
        :x="ZONES.lounge.x + 190"
        :y="ZONES.lounge.y + 100"
      />
      <Furniture
        type="coffeeCup"
        :x="ZONES.lounge.x + 100"
        :y="ZONES.lounge.y + 100"
      />
      <Furniture
        type="sofa"
        :x="ZONES.lounge.x + 440"
        :y="ZONES.lounge.y + 100"
        :rotation="90"
        :is-dark="isDark"
      />

      <!-- Logo wall -->
      <rect
        :x="loungeCenter - 100"
        :y="logoWallY"
        width="200"
        height="36"
        rx="4"
        :fill="isDark ? '#1e293b' : '#3b4f6b'"
      />
      <text
        :x="loungeCenter"
        :y="logoWallY + 23"
        text-anchor="middle"
        :fill="isDark ? '#94a3b8' : '#ffffff'"
        font-size="14"
        font-weight="700"
        font-family="system-ui, sans-serif"
        letter-spacing="0.12em"
      >
        Memoh
      </text>

      <!-- Reception desk -->
      <rect
        :x="loungeCenter - 80"
        :y="logoWallY + 50"
        width="160"
        height="24"
        rx="12"
        :fill="isDark ? '#475569' : '#8494a7'"
        :stroke="isDark ? '#334155' : '#5a6878'"
        stroke-width="1"
      />

      <!-- Plants -->
      <Furniture
        type="plant"
        :x="loungeCenter - 130"
        :y="logoWallY + 18"
      />
      <Furniture
        type="plant"
        :x="loungeCenter + 130"
        :y="logoWallY + 18"
      />
      <Furniture
        type="plant"
        :x="ZONES.lounge.x + 40"
        :y="ZONES.lounge.y + ZONES.lounge.height - 50"
      />
      <Furniture
        type="plant"
        :x="ZONES.lounge.x + ZONES.lounge.width - 40"
        :y="ZONES.lounge.y + ZONES.lounge.height - 50"
      />

      <!-- Layer 5a: Lounge agents -->
      <AgentAvatar
        v-for="agent in loungeAgents"
        :key="`lounge-${agent.id}`"
        :agent="agent"
        @select="$emit('selectAgent', $event)"
      />

      <!-- Layer 5b: Entrance door -->
      <rect
        :x="entranceCX - 37"
        :y="OFFICE.y + OFFICE.height - OFFICE.wallThickness - 1"
        width="74"
        :height="OFFICE.wallThickness + 4"
        :fill="isDark ? ZONE_COLORS_DARK.lounge : ZONE_COLORS.lounge"
      />
      <path
        :d="`M ${entranceCX - 35} ${OFFICE.y + OFFICE.height} A 35 35 0 0 0 ${entranceCX} ${OFFICE.y + OFFICE.height - 35}`"
        fill="none"
        :stroke="arcColor"
        stroke-width="0.8"
        stroke-dasharray="4 3"
        opacity="0.5"
      />
      <path
        :d="`M ${entranceCX + 35} ${OFFICE.y + OFFICE.height} A 35 35 0 0 1 ${entranceCX} ${OFFICE.y + OFFICE.height - 35}`"
        fill="none"
        :stroke="arcColor"
        stroke-width="0.8"
        stroke-dasharray="4 3"
        opacity="0.5"
      />
      <text
        :x="entranceCX"
        :y="OFFICE.y + OFFICE.height + 14"
        text-anchor="middle"
        :fill="isDark ? '#64748b' : '#94a3b8'"
        font-size="9"
        font-weight="600"
        font-family="system-ui, sans-serif"
        letter-spacing="0.15em"
      >
        ENTRANCE
      </text>

      <!-- Layer 7: Meeting agents -->
      <AgentAvatar
        v-for="(agent, i) in meetingAgents"
        :key="agent.id"
        :agent="{ ...agent, position: meetingSeats[i] ?? agent.position }"
        @select="$emit('selectAgent', $event)"
      />
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  SVG_WIDTH,
  SVG_HEIGHT,
  OFFICE,
  ZONES,
  ZONE_COLORS,
  ZONE_COLORS_DARK,
  type VisualAgent,
} from '../constants'
import AgentAvatar from './AgentAvatar.vue'
import Furniture from './Furniture.vue'

const props = defineProps<{
  agents: VisualAgent[]
  isDark: boolean
}>()

defineEmits<{
  selectAgent: [id: string]
}>()

const colors = computed(() => props.isDark ? ZONE_COLORS_DARK : ZONE_COLORS)

const cw = OFFICE.corridorWidth
const midX = OFFICE.x + (OFFICE.width - cw) / 2
const midY = OFFICE.y + (OFFICE.height - cw) / 2

const wallW = 4
const walls = [
  { x: midX - wallW / 2, y: OFFICE.y, w: wallW, h: midY - OFFICE.y },
  { x: midX - wallW / 2, y: midY + cw, w: wallW, h: OFFICE.y + OFFICE.height - midY - cw },
  { x: midX + cw - wallW / 2, y: OFFICE.y, w: wallW, h: midY - OFFICE.y },
  { x: midX + cw - wallW / 2, y: midY + cw, w: wallW, h: OFFICE.y + OFFICE.height - midY - cw },
  { x: OFFICE.x, y: midY - wallW / 2, w: midX - OFFICE.x, h: wallW },
  { x: midX + cw, y: midY - wallW / 2, w: OFFICE.x + OFFICE.width - midX - cw, h: wallW },
  { x: OFFICE.x, y: midY + cw - wallW / 2, w: midX - OFFICE.x, h: wallW },
  { x: midX + cw, y: midY + cw - wallW / 2, w: OFFICE.x + OFFICE.width - midX - cw, h: wallW },
]

const doorColor = computed(() => props.isDark ? ZONE_COLORS_DARK.corridor : ZONE_COLORS.corridor)
const arcColor = computed(() => props.isDark ? '#64748b' : '#94a3b8')

const doors = [
  { cx: (OFFICE.x + midX) / 2, cy: midY, horizontal: true },
  { cx: (midX + cw + OFFICE.x + OFFICE.width) / 2, cy: midY, horizontal: true },
  { cx: (OFFICE.x + midX) / 2, cy: midY + cw, horizontal: true },
  { cx: (midX + cw + OFFICE.x + OFFICE.width) / 2, cy: midY + cw, horizontal: true },
  { cx: midX, cy: (OFFICE.y + midY) / 2, horizontal: false },
  { cx: midX + cw, cy: (OFFICE.y + midY) / 2, horizontal: false },
  { cx: midX, cy: (midY + cw + OFFICE.y + OFFICE.height) / 2, horizontal: false },
  { cx: midX + cw, cy: (midY + cw + OFFICE.y + OFFICE.height) / 2, horizontal: false },
]

// Agent zone filtering
const deskAgents = computed(() =>
  props.agents.filter(a => a.zone === 'desk' && !a.isSubAgent && a.confirmed),
)
const hotDeskAgents = computed(() =>
  props.agents.filter(a => a.zone === 'hotDesk'),
)
const meetingAgents = computed(() =>
  props.agents.filter(a => a.zone === 'meeting'),
)
const loungeAgents = computed(() =>
  props.agents.filter(a => a.zone === 'lounge'),
)

// Desk slot calculation
function calculateSlots(zone: typeof ZONES.desk, count: number) {
  const cols = 4
  const rows = Math.ceil(Math.max(count, 4) / cols)
  const paddingX = 40
  const paddingY = 50
  const cellW = (zone.width - paddingX * 2) / cols
  const cellH = (zone.height - paddingY * 2) / rows
  const slots: Array<{ x: number; y: number }> = []
  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < cols; c++) {
      if (slots.length >= Math.max(count, 4)) break
      slots.push({
        x: zone.x + paddingX + cellW * (c + 0.5),
        y: zone.y + paddingY + cellH * (r + 0.5),
      })
    }
  }
  return slots
}

const deskSlots = computed(() => calculateSlots(ZONES.desk, deskAgents.value.length))
const hotDeskSlots = computed(() => calculateSlots(ZONES.hotDesk, hotDeskAgents.value.length))

// Desk agent assignment by hash
const deskAgentMap = computed(() => {
  const map = new Map<number, VisualAgent>()
  for (const agent of deskAgents.value) {
    let hash = 0
    for (let i = 0; i < agent.id.length; i++) {
      hash = ((hash << 5) - hash + agent.id.charCodeAt(i)) | 0
    }
    const idx = Math.abs(hash) % deskSlots.value.length
    let slot = idx
    while (map.has(slot)) {
      slot = (slot + 1) % deskSlots.value.length
    }
    map.set(slot, {
      ...agent,
      position: { x: deskSlots.value[slot].x, y: deskSlots.value[slot].y - 12 },
    })
  }
  return map
})

// Meeting layout
const meetingCenter = {
  x: ZONES.meeting.x + ZONES.meeting.width / 2,
  y: ZONES.meeting.y + ZONES.meeting.height / 2,
}

const meetingTableRadius = computed(() =>
  Math.min(60 + meetingAgents.value.length * 8, Math.min(ZONES.meeting.width, ZONES.meeting.height) / 2 - 40),
)

const meetingSeats = computed(() => {
  const count = Math.max(meetingAgents.value.length, 6)
  const r = meetingTableRadius.value + 36
  return Array.from({ length: count }, (_, i) => {
    const angle = (2 * Math.PI * i) / count - Math.PI / 2
    return {
      x: Math.round(meetingCenter.x + Math.cos(angle) * r),
      y: Math.round(meetingCenter.y + Math.sin(angle) * r),
    }
  })
})

const meetingChairs = computed(() => {
  if (meetingAgents.value.length > 0) return meetingSeats.value.slice(0, meetingAgents.value.length)
  return meetingSeats.value.slice(0, 6)
})

// Lounge layout
const loungeCenter = ZONES.lounge.x + ZONES.lounge.width / 2
const logoWallY = ZONES.lounge.y + ZONES.lounge.height * 0.52
const entranceCX = loungeCenter
</script>
