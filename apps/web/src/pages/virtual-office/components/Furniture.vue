<template>
  <!-- Desk -->
  <g
    v-if="type === 'desk'"
    :transform="`translate(${x}, ${y})`"
  >
    <rect
      x="-50"
      y="0"
      width="100"
      height="60"
      rx="4"
      :fill="isDark ? '#334155' : '#d1d5db'"
    />
    <rect
      x="-50"
      y="55"
      width="100"
      height="8"
      rx="2"
      :fill="isDark ? '#475569' : '#9ca3af'"
    />
    <rect
      x="-20"
      y="8"
      width="40"
      height="25"
      rx="3"
      :fill="isDark ? '#1e293b' : '#e5e7eb'"
    />
    <rect
      x="-12"
      y="38"
      width="24"
      height="4"
      rx="1"
      :fill="isDark ? '#475569' : '#d1d5db'"
    />
  </g>

  <!-- Chair -->
  <g
    v-else-if="type === 'chair'"
    :transform="`translate(${x}, ${y})`"
  >
    <path
      d="M -10 -6 A 12 8 0 0 1 10 -6"
      :fill="isDark ? '#475569' : '#9ca3af'"
    />
    <circle
      r="14"
      :fill="isDark ? '#334155' : '#6b7280'"
    />
  </g>

  <!-- Meeting Table -->
  <g
    v-else-if="type === 'meetingTable'"
    :transform="`translate(${x}, ${y})`"
  >
    <circle
      :r="radius"
      :fill="isDark ? '#334155' : '#9ca3af'"
      :stroke="isDark ? '#475569' : '#6b7280'"
      stroke-width="2"
    />
    <circle
      :r="radius! * 0.7"
      :fill="isDark ? '#3f4f63' : '#b0b8c4'"
      opacity="0.4"
    />
  </g>

  <!-- Sofa -->
  <g
    v-else-if="type === 'sofa'"
    :transform="`translate(${x}, ${y}) rotate(${rotation ?? 0})`"
  >
    <rect
      x="-55"
      y="-22"
      width="110"
      height="44"
      rx="8"
      :fill="isDark ? '#334155' : '#8b9bb0'"
    />
    <rect
      x="-48"
      y="-16"
      width="96"
      height="32"
      rx="6"
      :fill="isDark ? '#475569' : '#a5b4c8'"
    />
    <circle
      cx="-36"
      cy="0"
      r="10"
      :fill="isDark ? '#5a6b80' : '#bac4d0'"
      opacity="0.6"
    />
    <circle
      cx="36"
      cy="0"
      r="10"
      :fill="isDark ? '#5a6b80' : '#bac4d0'"
      opacity="0.6"
    />
  </g>

  <!-- Plant -->
  <g
    v-else-if="type === 'plant'"
    :transform="`translate(${x}, ${y})`"
  >
    <rect
      x="-8"
      y="4"
      width="16"
      height="14"
      rx="3"
      fill="#8b6b47"
    />
    <ellipse
      cx="0"
      cy="-4"
      rx="8"
      ry="10"
      fill="#22c55e"
      opacity="0.8"
    />
    <ellipse
      cx="-6"
      cy="0"
      rx="6"
      ry="8"
      fill="#16a34a"
      opacity="0.6"
    />
    <ellipse
      cx="6"
      cy="0"
      rx="6"
      ry="8"
      fill="#15803d"
      opacity="0.5"
    />
  </g>

  <!-- Coffee Cup -->
  <g
    v-else-if="type === 'coffeeCup'"
    :transform="`translate(${x}, ${y})`"
  >
    <ellipse
      rx="8"
      ry="4"
      fill="#d4c4a8"
      opacity="0.5"
    />
    <ellipse
      cy="-2"
      rx="6"
      ry="3"
      fill="#f5f0e8"
    />
    <ellipse
      cy="-2"
      rx="5"
      ry="2"
      fill="#6b4226"
      opacity="0.8"
    />
    <path
      d="M -2 -6 Q -1 -10 0 -6 Q 1 -10 2 -6"
      fill="none"
      stroke="#94a3b8"
      stroke-width="0.5"
      opacity="0.4"
    />
  </g>
</template>

<script setup lang="ts">
defineProps<{
  type: 'desk' | 'chair' | 'meetingTable' | 'sofa' | 'plant' | 'coffeeCup'
  x: number
  y: number
  isDark?: boolean
  radius?: number
  rotation?: number
}>()
</script>
