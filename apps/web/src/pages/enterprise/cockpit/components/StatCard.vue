<template>
  <div
    class="glass-panel-hover p-5 relative overflow-hidden"
    :class="glowClass"
  >
    <div class="flex items-center justify-between mb-3">
      <span class="text-xs font-medium text-text-secondary uppercase tracking-wider">
        {{ title }}
      </span>
      <div
        class="size-8 rounded-lg flex items-center justify-center"
        :class="iconBgClass"
      >
        <FontAwesomeIcon
          :icon="icon"
          class="size-4"
          :class="iconColorClass"
        />
      </div>
    </div>
    <p class="text-2xl font-bold tabular-nums text-text-primary">
      {{ value }}
    </p>
    <div
      v-if="trend !== undefined"
      class="flex items-center gap-1 mt-1"
    >
      <FontAwesomeIcon
        :icon="trend >= 0 ? ['fas', 'arrow-up'] : ['fas', 'arrow-down']"
        class="size-3"
        :class="trend >= 0 ? 'text-accent-success' : 'text-accent-danger'"
      />
      <span
        class="text-xs font-medium"
        :class="trend >= 0 ? 'text-accent-success' : 'text-accent-danger'"
      >
        {{ trend >= 0 ? '+' : '' }}{{ trend.toFixed(1) }}%
      </span>
    </div>
    <p
      v-else-if="desc"
      class="text-xs text-text-muted mt-1"
    >
      {{ desc }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  title: string
  value: string
  icon: string[]
  glowColor: 'cyan' | 'purple' | 'green' | 'amber' | 'orange'
  trend?: number
  desc?: string
}>()

const glowClass = computed(() => `glow-border-${props.glowColor}`)

const colorMap = {
  cyan: { bg: 'bg-cyan-500/10', text: 'text-cyan-500' },
  purple: { bg: 'bg-violet-500/10', text: 'text-violet-500' },
  green: { bg: 'bg-emerald-500/10', text: 'text-emerald-500' },
  amber: { bg: 'bg-amber-500/10', text: 'text-amber-500' },
  orange: { bg: 'bg-orange-500/10', text: 'text-orange-500' },
}

const iconBgClass = computed(() => colorMap[props.glowColor].bg)
const iconColorClass = computed(() => colorMap[props.glowColor].text)
</script>
