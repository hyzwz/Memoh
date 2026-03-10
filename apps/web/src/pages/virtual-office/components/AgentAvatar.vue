<template>
  <g
    :transform="`translate(${agent.position.x}, ${agent.position.y})`"
    class="cursor-pointer"
    @click="$emit('select', agent.id)"
  >
    <!-- Status ring -->
    <circle
      :r="AVATAR.radius + 2"
      fill="none"
      :stroke="statusColor"
      :stroke-width="AVATAR.strokeWidth"
      :opacity="agent.confirmed ? 1 : 0.5"
      :style="ringAnimStyle"
    />

    <!-- Avatar circle -->
    <circle
      :r="AVATAR.radius"
      :fill="avatarBg"
      :opacity="agent.confirmed ? 1 : 0.5"
    />

    <!-- Face - simple procedural -->
    <circle
      :cx="-5"
      :cy="-4"
      r="2.5"
      :fill="faceColor"
    />
    <circle
      :cx="5"
      :cy="-4"
      r="2.5"
      :fill="faceColor"
    />
    <path
      :d="smilePath"
      fill="none"
      :stroke="faceColor"
      stroke-width="1.5"
      stroke-linecap="round"
    />

    <!-- Sub-agent badge -->
    <g v-if="agent.isSubAgent">
      <circle
        :cx="AVATAR.radius - 4"
        :cy="-AVATAR.radius + 4"
        r="7"
        fill="#8b5cf6"
      />
      <text
        :x="AVATAR.radius - 4"
        :y="-AVATAR.radius + 7"
        text-anchor="middle"
        fill="white"
        font-size="8"
        font-weight="700"
      >
        S
      </text>
    </g>

    <!-- Thinking dots -->
    <g v-if="agent.status === 'thinking'">
      <circle
        cx="-6"
        cy="-30"
        r="2"
        fill="#3b82f6"
        opacity="0.6"
      >
        <animate
          attributeName="opacity"
          values="0.3;1;0.3"
          dur="1.2s"
          repeatCount="indefinite"
        />
      </circle>
      <circle
        cx="0"
        cy="-30"
        r="2"
        fill="#3b82f6"
        opacity="0.6"
      >
        <animate
          attributeName="opacity"
          values="0.3;1;0.3"
          dur="1.2s"
          begin="0.2s"
          repeatCount="indefinite"
        />
      </circle>
      <circle
        cx="6"
        cy="-30"
        r="2"
        fill="#3b82f6"
        opacity="0.6"
      >
        <animate
          attributeName="opacity"
          values="0.3;1;0.3"
          dur="1.2s"
          begin="0.4s"
          repeatCount="indefinite"
        />
      </circle>
    </g>

    <!-- Tool label -->
    <text
      v-if="agent.currentTool"
      y="36"
      text-anchor="middle"
      fill="#94a3b8"
      font-size="8"
      font-family="system-ui, sans-serif"
    >
      {{ agent.currentTool }}
    </text>

    <!-- Name label -->
    <g>
      <rect
        :x="-nameWidth / 2 - 4"
        :y="AVATAR.radius + 6"
        :width="nameWidth + 8"
        height="16"
        rx="4"
        fill="rgba(15,23,42,0.7)"
      />
      <text
        :y="AVATAR.radius + 17"
        text-anchor="middle"
        fill="#e2e8f0"
        font-size="9"
        font-weight="500"
        font-family="system-ui, sans-serif"
      >
        {{ displayName }}
      </text>
    </g>
  </g>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { AVATAR, STATUS_COLORS, type VisualAgent } from '../constants'

const props = defineProps<{
  agent: VisualAgent
}>()

defineEmits<{
  select: [id: string]
}>()

const statusColor = computed(() => STATUS_COLORS[props.agent.status])

// Deterministic avatar colors from agent id
const avatarBg = computed(() => {
  let hash = 0
  for (let i = 0; i < props.agent.id.length; i++) {
    hash = ((hash << 5) - hash + props.agent.id.charCodeAt(i)) | 0
  }
  const hue = Math.abs(hash) % 360
  return `hsl(${hue}, 45%, 65%)`
})

const faceColor = computed(() => {
  let hash = 0
  for (let i = 0; i < props.agent.id.length; i++) {
    hash = ((hash << 5) - hash + props.agent.id.charCodeAt(i)) | 0
  }
  const hue = Math.abs(hash) % 360
  return `hsl(${hue}, 30%, 30%)`
})

const smilePath = computed(() => {
  const statuses = ['idle', 'speaking']
  if (statuses.includes(props.agent.status)) {
    return 'M -4 4 Q 0 8 4 4'
  }
  if (props.agent.status === 'error') {
    return 'M -4 7 Q 0 3 4 7'
  }
  return 'M -4 5 L 4 5'
})

const displayName = computed(() => {
  const n = props.agent.name
  return n.length > 12 ? n.slice(0, 11) + '\u2026' : n
})

const nameWidth = computed(() => displayName.value.length * 6)

const ringAnimStyle = computed(() => {
  switch (props.agent.status) {
    case 'thinking': return { animation: 'agent-pulse 1.5s ease-in-out infinite' }
    case 'tool_calling': return { animation: 'agent-pulse 2s ease-in-out infinite', strokeDasharray: '6 4' }
    case 'speaking': return { animation: 'agent-pulse 1s ease-in-out infinite' }
    case 'error': return { animation: 'agent-blink 0.8s ease-in-out infinite' }
    case 'spawning': return { animation: 'agent-spawn 0.5s ease-out' }
    default: return {}
  }
})
</script>
