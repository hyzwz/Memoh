<template>
  <div class="glass-panel p-5">
    <h3 class="text-sm font-medium text-text-secondary mb-4">
      {{ $t('cockpit.userRanking') }}
    </h3>
    <div class="space-y-3">
      <div
        v-for="(user, index) in data"
        :key="user.userId"
        class="flex items-center gap-3"
      >
        <span
          class="w-5 text-center text-xs font-bold tabular-nums"
          :class="index < 3 ? 'text-accent-primary' : 'text-text-muted'"
        >
          {{ index + 1 }}
        </span>
        <div class="flex-1 min-w-0">
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm text-text-primary truncate">
              {{ user.userName }}
            </span>
            <span class="text-xs text-text-secondary tabular-nums">
              {{ user.savedHours.toFixed(1) }}h
            </span>
          </div>
          <div class="h-1.5 rounded-full bg-glass-light overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500"
              :style="{ width: barWidth(user.savedHours), background: barColor(index) }"
            />
          </div>
        </div>
        <span class="text-xs text-text-muted tabular-nums w-10 text-right">
          {{ user.avgMultiplier.toFixed(1) }}x
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  data: Array<{
    userId: string
    userName: string
    department: string
    totalTasks: number
    savedHours: number
    avgMultiplier: number
  }>
}>()

const maxHours = computed(() => {
  if (props.data.length === 0) return 1
  return Math.max(...props.data.map(u => u.savedHours))
})

function barWidth(hours: number) {
  return `${(hours / maxHours.value) * 100}%`
}

function barColor(index: number) {
  const colors = ['#06b6d4', '#8b5cf6', '#10b981', '#f59e0b', '#ec4899', '#3b82f6', '#f97316', '#14b8a6']
  return colors[index % colors.length]
}
</script>
