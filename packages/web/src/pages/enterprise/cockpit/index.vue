<template>
  <div class="p-6 space-y-6 mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">AI效能驾驶舱</h1>
        <p class="text-sm text-muted-foreground mt-1">实时监控AI助手的效能指标与投资回报</p>
      </div>
      <div class="flex items-center gap-3">
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-56">
            <SelectValue placeholder="选择Bot" />
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
        <div class="flex items-center rounded-lg border p-1 gap-0.5">
          <button
            v-for="opt in dayOptions"
            :key="opt.value"
            class="rounded-md px-3 py-1.5 text-sm transition-colors"
            :class="days === opt.value
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:text-foreground hover:bg-muted'"
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
          <Spinner v-if="isLoading" class="mr-2 size-4" />
          刷新
        </Button>
      </div>
    </div>

    <!-- No bot selected -->
    <template v-if="!selectedBotId">
      <div class="flex flex-col items-center justify-center py-20 text-muted-foreground">
        <svg xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M10.5 6a7.5 7.5 0 1 0 7.5 7.5h-7.5V6Z" /><path stroke-linecap="round" stroke-linejoin="round" d="M13.5 10.5H21A7.5 7.5 0 0 0 13.5 3v7.5Z" /></svg>
        <p class="text-sm">请先选择一个Bot查看效能数据</p>
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
      <!-- Stat Cards Row -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        <Card v-for="stat in statCards" :key="stat.label" class="relative overflow-hidden">
          <CardContent class="p-5">
            <div class="flex items-center justify-between mb-3">
              <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ stat.label }}</span>
              <div class="size-8 rounded-lg flex items-center justify-center" :class="stat.iconBg">
                <component :is="stat.icon" class="size-4" :class="stat.iconColor" />
              </div>
            </div>
            <p class="text-2xl font-bold tabular-nums">{{ stat.value }}</p>
            <p v-if="stat.desc" class="text-xs text-muted-foreground mt-1">{{ stat.desc }}</p>
          </CardContent>
        </Card>
      </div>

      <!-- Footer stats -->
      <div class="flex flex-wrap gap-6 text-sm text-muted-foreground border-t pt-4">
        <span>报告周期: 近{{ days }}天</span>
        <span>更新时间: {{ new Date().toLocaleString('zh-CN') }}</span>
      </div>
    </template>

    <!-- No Data -->
    <template v-else>
      <div class="flex flex-col items-center justify-center py-20 text-muted-foreground">
        <svg xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" /></svg>
        <p class="text-sm">暂无效能数据</p>
        <p class="text-xs mt-1">Bot使用后将自动生成报告</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, h } from 'vue'
import { useQuery } from '@pinia/colada'
import {
  Button,
  Card,
  CardContent,
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

const selectedBotId = useSyncedQueryParam('bot', '')
const days = useSyncedQueryParam('days', '7')

const dayOptions = [
  { value: '7', label: '近7天' },
  { value: '30', label: '近30天' },
  { value: '90', label: '近90天' },
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

// SVG icon components rendered inline
const ClockIcon = { render: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', 'stroke-width': '1.5', stroke: 'currentColor' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z' })]) }
const BoltIcon = { render: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', 'stroke-width': '1.5', stroke: 'currentColor' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'm3.75 13.5 10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75Z' })]) }
const ChartIcon = { render: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', 'stroke-width': '1.5', stroke: 'currentColor' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z' })]) }
const SparklesIcon = { render: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', 'stroke-width': '1.5', stroke: 'currentColor' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.455 2.456L21.75 6l-1.036.259a3.375 3.375 0 0 0-2.455 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z' })]) }
const CoinIcon = { render: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', 'stroke-width': '1.5', stroke: 'currentColor' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M12 6v12m-3-2.818.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z' })]) }
const DocIcon = { render: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', fill: 'none', viewBox: '0 0 24 24', 'stroke-width': '1.5', stroke: 'currentColor' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z' })]) }

const statCards = computed(() => {
  const s = summary.value
  if (!s) return []
  return [
    {
      label: '节省工时',
      value: `${(s.total_saved_hours ?? 0).toFixed(1)}h`,
      desc: '等效人工工时',
      icon: ClockIcon,
      iconBg: 'bg-cyan-500/10',
      iconColor: 'text-cyan-500',
    },
    {
      label: 'AI用时',
      value: `${s.total_ai_time_minutes ?? 0}min`,
      desc: 'AI处理总时长',
      icon: BoltIcon,
      iconBg: 'bg-violet-500/10',
      iconColor: 'text-violet-500',
    },
    {
      label: '效率倍数',
      value: `${(s.average_efficiency_multiple ?? 0).toFixed(1)}x`,
      desc: '人工/AI时间比',
      icon: ChartIcon,
      iconBg: 'bg-emerald-500/10',
      iconColor: 'text-emerald-500',
    },
    {
      label: '报告数',
      value: `${s.total_reports ?? 0}`,
      desc: '自动生成报告',
      icon: DocIcon,
      iconBg: 'bg-blue-500/10',
      iconColor: 'text-blue-500',
    },
    {
      label: '创新应用',
      value: `${s.total_innovations ?? 0}`,
      desc: '创新场景发现',
      icon: SparklesIcon,
      iconBg: 'bg-amber-500/10',
      iconColor: 'text-amber-500',
    },
    {
      label: '节省成本',
      value: formatCNY(s.labor_cost_cny ?? 0),
      desc: '等效人力成本',
      icon: CoinIcon,
      iconBg: 'bg-rose-500/10',
      iconColor: 'text-rose-500',
    },
  ]
})

function formatCNY(n: number): string {
  if (n >= 10000) return `\u00A5${(n / 10000).toFixed(1)}万`
  return `\u00A5${n.toLocaleString('zh-CN', { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`
}
</script>
