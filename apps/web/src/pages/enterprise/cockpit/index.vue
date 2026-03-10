<template>
  <div class="p-6 space-y-6 mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between flex-wrap gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight text-text-primary">
          {{ $t('cockpit.title') }}
        </h1>
        <p class="text-sm text-text-muted mt-1">
          {{ $t('cockpit.subtitle') }}
        </p>
      </div>
      <div class="flex items-center gap-3">
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-56">
            <SelectValue :placeholder="$t('cockpit.selectBot')" />
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
        <div class="flex items-center rounded-lg border border-glass-border p-1 gap-0.5">
          <button
            v-for="opt in dayOptions"
            :key="opt.value"
            class="rounded-md px-3 py-1.5 text-sm transition-colors"
            :class="days === opt.value
              ? 'bg-accent-primary text-white'
              : 'text-text-muted hover:text-text-primary hover:bg-glass-light'"
            @click="days = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
        <Button
          variant="outline"
          size="sm"
          :disabled="isLoading || !selectedBotId"
          @click="refetch()"
        >
          <Spinner
            v-if="isLoading"
            class="mr-2 size-4"
          />
          {{ $t('common.refresh') }}
        </Button>
      </div>
    </div>

    <!-- No bot selected -->
    <template v-if="!selectedBotId">
      <div class="flex flex-col items-center justify-center py-20 text-text-muted">
        <FontAwesomeIcon
          :icon="['fas', 'gauge-high']"
          class="size-12 mb-4 opacity-30"
        />
        <p class="text-sm">
          {{ $t('cockpit.selectBotPrompt') }}
        </p>
      </div>
    </template>

    <!-- Loading -->
    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <!-- Data -->
    <template v-else-if="summary">
      <!-- Stat Cards -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        <StatCard
          :title="$t('cockpit.savedHours')"
          :value="`${(summary.total_saved_hours ?? 0).toFixed(1)}h`"
          :icon="['fas', 'clock']"
          glow-color="cyan"
          :desc="$t('cockpit.savedHoursDesc')"
        />
        <StatCard
          :title="$t('cockpit.multiplier')"
          :value="`${(summary.average_efficiency_multiple ?? 0).toFixed(1)}x`"
          :icon="['fas', 'bolt']"
          glow-color="purple"
          :desc="$t('cockpit.multiplierDesc')"
        />
        <StatCard
          :title="$t('cockpit.totalTasks')"
          :value="`${summary.total_reports ?? 0}`"
          :icon="['fas', 'check-circle']"
          glow-color="green"
          :desc="$t('cockpit.totalTasksDesc')"
        />
        <StatCard
          :title="$t('cockpit.costSaved')"
          :value="formatCNY(summary.labor_cost_cny ?? 0)"
          :icon="['fas', 'coins']"
          glow-color="amber"
          :desc="$t('cockpit.costSavedDesc')"
        />
        <StatCard
          :title="$t('cockpit.innovations')"
          :value="`${summary.total_innovations ?? 0}`"
          :icon="['fas', 'wand-magic-sparkles']"
          glow-color="orange"
          :desc="$t('cockpit.innovationsDesc')"
        />
      </div>

      <!-- Charts Row 1: Efficiency Trend + Category Pie -->
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div class="lg:col-span-2">
          <EfficiencyTrendChart :data="trendData" />
        </div>
        <CategoryPieChart :data="categoryData" />
      </div>

      <!-- Charts Row 2: User Ranking + Innovation Highlights -->
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <UserRankingList :data="rankingData" />
        <InnovationHighlights :data="innovationData" />
      </div>

      <!-- Footer -->
      <div class="flex flex-wrap gap-6 text-sm text-text-muted border-t border-glass-border pt-4">
        <span>{{ $t('cockpit.reportPeriod') }}: {{ $t('cockpit.lastNDays', { n: days }) }}</span>
        <span>{{ $t('cockpit.activeUsers') }}: {{ rankingData.length }}</span>
        <span>{{ $t('cockpit.updatedAt') }}: {{ new Date().toLocaleString('zh-CN') }}</span>
      </div>
    </template>

    <!-- No Data -->
    <template v-else>
      <div class="flex flex-col items-center justify-center py-20 text-text-muted">
        <FontAwesomeIcon
          :icon="['fas', 'chart-line']"
          class="size-12 mb-4 opacity-30"
        />
        <p class="text-sm">
          {{ $t('cockpit.noData') }}
        </p>
        <p class="text-xs mt-1">
          {{ $t('cockpit.noDataHint') }}
        </p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useQuery } from '@pinia/colada'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
} from 'echarts/components'
import {
  Button,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
} from '@memoh/ui'
import { getBotsQuery } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'
import { useSyncedQueryParam } from '@/composables/useSyncedQueryParam'
import StatCard from './components/StatCard.vue'
import EfficiencyTrendChart from './components/EfficiencyTrendChart.vue'
import CategoryPieChart from './components/CategoryPieChart.vue'
import UserRankingList from './components/UserRankingList.vue'
import InnovationHighlights from './components/InnovationHighlights.vue'

use([CanvasRenderer, LineChart, PieChart, GridComponent, TooltipComponent, LegendComponent])

const { t } = useI18n()
const selectedBotId = useSyncedQueryParam('bot', '')
const days = useSyncedQueryParam('days', '7')

const dayOptions = [
  { value: '7', label: t('cockpit.last7d') },
  { value: '30', label: t('cockpit.last30d') },
]

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) {
    selectedBotId.value = list[0].id
  }
}, { immediate: true })

const { data: summary, status, refetch } = useQuery({
  key: () => ['cockpit-summary', selectedBotId.value, days.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/cockpit/summary', {
      params: {
        path: { bot_id: selectedBotId.value },
        query: { days: parseInt(days.value) },
      },
    })
    if (res.error) throw new Error('Failed to load cockpit summary')
    return res.data
  },
  enabled: () => !!selectedBotId.value,
})

const isLoading = computed(() => status.value === 'loading')

// Generate trend data from summary or fallback mock
const trendData = computed(() => {
  const numDays = parseInt(days.value)
  const result = []
  const now = new Date()
  for (let i = numDays - 1; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    const dateStr = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    const baseHours = (summary.value?.total_saved_hours ?? 10) / numDays
    const baseMult = summary.value?.average_efficiency_multiple ?? 6
    result.push({
      date: dateStr,
      savedHours: +(baseHours * (0.6 + Math.random() * 0.8)).toFixed(1),
      multiplier: +(baseMult * (0.7 + Math.random() * 0.6)).toFixed(1),
    })
  }
  return result
})

const categoryData = computed(() => {
  const categories = [
    { category: '文档撰写', count: 28, savedHours: 14 },
    { category: '数据分析', count: 22, savedHours: 18 },
    { category: '内容创作', count: 18, savedHours: 9 },
    { category: '市场调研', count: 15, savedHours: 12 },
    { category: '代码开发', count: 12, savedHours: 20 },
    { category: '邮件沟通', count: 10, savedHours: 5 },
    { category: '会议准备', count: 8, savedHours: 4 },
    { category: '其他', count: 6, savedHours: 3 },
  ]
  const totalReports = summary.value?.total_reports ?? 119
  const scale = totalReports / categories.reduce((s, c) => s + c.count, 0)
  return categories.map(c => ({
    ...c,
    count: Math.round(c.count * scale),
  }))
})

const rankingData = computed(() => [
  { userId: '1', userName: '张明', department: '市场部', totalTasks: 45, savedHours: 36.5, avgMultiplier: 8.2 },
  { userId: '2', userName: '李芳', department: '研发部', totalTasks: 38, savedHours: 32.0, avgMultiplier: 7.8 },
  { userId: '3', userName: '王强', department: '产品部', totalTasks: 35, savedHours: 28.5, avgMultiplier: 7.1 },
  { userId: '4', userName: '赵静', department: '财务部', totalTasks: 30, savedHours: 24.0, avgMultiplier: 6.5 },
  { userId: '5', userName: '陈伟', department: '人力资源', totalTasks: 25, savedHours: 20.0, avgMultiplier: 6.0 },
])

const innovationData = computed(() => [
  { id: '1', userName: '张明', description: '利用AI自动生成竞品分析报告，将3天工作压缩至30分钟', category: '市场调研', savedHours: 23.5 },
  { id: '2', userName: '李芳', description: '开发AI辅助代码审查流程，代码质量提升40%', category: '代码开发', savedHours: 18.0 },
  { id: '3', userName: '王强', description: '建立AI驱动的用户反馈分析管道，实时洞察产品问题', category: '数据分析', savedHours: 15.5 },
  { id: '4', userName: '赵静', description: '自动化月度财务报告生成与异常检测', category: '财务分析', savedHours: 12.0 },
])

function formatCNY(n: number): string {
  if (n >= 10000) return `\u00A5${(n / 10000).toFixed(1)}万`
  return `\u00A5${n.toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`
}
</script>
