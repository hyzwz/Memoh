<template>
  <div class="h-full flex flex-col">
    <!-- Header bar -->
    <div class="flex items-center justify-between px-6 py-3 border-b border-glass-border">
      <div class="flex items-center gap-3">
        <h2 class="text-lg font-semibold text-text-primary">
          {{ $t('virtualOffice.title') }}
        </h2>
        <span class="text-xs text-text-muted">
          {{ agents.length }} {{ $t('virtualOffice.agentsOnline') }}
        </span>
      </div>
      <div class="flex items-center gap-2">
        <!-- Status legend -->
        <div class="flex items-center gap-3 mr-4">
          <div
            v-for="(color, status) in visibleStatuses"
            :key="status"
            class="flex items-center gap-1"
          >
            <span
              class="size-2 rounded-full"
              :style="{ background: color }"
            />
            <span class="text-[10px] text-text-muted">{{ $t(`virtualOffice.status.${status}`) }}</span>
          </div>
        </div>
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-48">
            <SelectValue :placeholder="$t('virtualOffice.selectBot')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="bot in botList"
              :key="bot.id"
              :value="bot.id!"
            >
              {{ bot.display_name || bot.id }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <!-- Floor plan -->
    <div class="flex-1 relative overflow-hidden bg-surface-base">
      <FloorPlan
        :agents="agents"
        :is-dark="isDark"
        @select-agent="selectedAgentId = $event"
      />

      <!-- Agent detail panel -->
      <div
        v-if="selectedAgent"
        class="absolute top-4 right-4 w-72 glass-panel p-4 glow-border-cyan"
      >
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-sm font-semibold text-text-primary">
            {{ selectedAgent.name }}
          </h3>
          <button
            class="text-text-muted hover:text-text-primary"
            @click="selectedAgentId = null"
          >
            <FontAwesomeIcon :icon="['fas', 'xmark']" />
          </button>
        </div>
        <div class="space-y-2 text-sm">
          <div class="flex items-center gap-2">
            <span
              class="size-2 rounded-full"
              :style="{ background: STATUS_COLORS[selectedAgent.status] }"
            />
            <span class="text-text-secondary">
              {{ $t(`virtualOffice.status.${selectedAgent.status}`) }}
            </span>
          </div>
          <div
            v-if="selectedAgent.currentTool"
            class="flex items-center gap-2"
          >
            <FontAwesomeIcon
              :icon="['fas', 'wrench']"
              class="size-3 text-text-muted"
            />
            <span class="text-text-secondary">{{ selectedAgent.currentTool }}</span>
          </div>
          <div class="flex items-center gap-2">
            <FontAwesomeIcon
              :icon="['fas', 'location-dot']"
              class="size-3 text-text-muted"
            />
            <span class="text-text-secondary">{{ selectedAgent.zone }}</span>
          </div>
          <div
            v-if="selectedAgent.isSubAgent"
            class="inline-flex items-center rounded-md bg-violet-500/15 px-2 py-0.5 text-[10px] font-medium text-violet-400"
          >
            Sub-Agent
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useQuery } from '@pinia/colada'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@memoh/ui'
import { getBotsQuery } from '@memoh/sdk/colada'
import { useSettingsStore } from '@/store/settings'
import { useSyncedQueryParam } from '@/composables/useSyncedQueryParam'
import { STATUS_COLORS, type VisualAgent } from './constants'
import FloorPlan from './components/FloorPlan.vue'

const { theme } = storeToRefs(useSettingsStore())
const isDark = computed(() => theme.value === 'dark' || theme.value === 'deep-space')

const selectedBotId = useSyncedQueryParam('bot', '')
const selectedAgentId = ref<string | null>(null)

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) {
    selectedBotId.value = list[0].id
  }
}, { immediate: true })

const visibleStatuses = {
  idle: '#22c55e',
  thinking: '#3b82f6',
  tool_calling: '#f97316',
  speaking: '#a855f7',
  offline: '#6b7280',
}

// Mock agents for demonstration (will be replaced with real data from WebSocket)
const agents = computed<VisualAgent[]>(() => {
  if (!selectedBotId.value) return []
  return [
    { id: 'agent-1', name: '数据分析师', status: 'idle', position: { x: 0, y: 0 }, zone: 'desk', isSubAgent: false, confirmed: true, currentTool: null },
    { id: 'agent-2', name: '内容创作者', status: 'thinking', position: { x: 0, y: 0 }, zone: 'desk', isSubAgent: false, confirmed: true, currentTool: null },
    { id: 'agent-3', name: '代码助手', status: 'tool_calling', position: { x: 0, y: 0 }, zone: 'desk', isSubAgent: false, confirmed: true, currentTool: 'execute_code' },
    { id: 'agent-4', name: '邮件助手', status: 'speaking', position: { x: 0, y: 0 }, zone: 'desk', isSubAgent: false, confirmed: true, currentTool: null },
    { id: 'agent-5', name: '项目经理', status: 'idle', position: { x: 0, y: 0 }, zone: 'meeting', isSubAgent: false, confirmed: true, currentTool: null },
    { id: 'agent-6', name: '设计顾问', status: 'thinking', position: { x: 0, y: 0 }, zone: 'meeting', isSubAgent: false, confirmed: true, currentTool: null },
    { id: 'agent-7', name: '搜索子代理', status: 'tool_calling', position: { x: 0, y: 0 }, zone: 'hotDesk', isSubAgent: true, confirmed: true, currentTool: 'web_search' },
    { id: 'agent-8', name: '翻译子代理', status: 'idle', position: { x: 0, y: 0 }, zone: 'hotDesk', isSubAgent: true, confirmed: true, currentTool: null },
    { id: 'agent-9', name: '休息中', status: 'offline', position: { x: 780, y: 470 }, zone: 'lounge', isSubAgent: false, confirmed: true, currentTool: null },
  ]
})

const selectedAgent = computed(() => {
  if (!selectedAgentId.value) return null
  return agents.value.find(a => a.id === selectedAgentId.value) ?? null
})
</script>
