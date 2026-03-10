<template>
  <div class="glass-panel p-5">
    <h3 class="text-sm font-medium text-text-secondary mb-4">
      {{ $t('cockpit.categoryDistribution') }}
    </h3>
    <VChart
      style="height: 280px; width: 100%"
      :option="chartOption"
      autoresize
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import VChart from 'vue-echarts'

const props = defineProps<{
  data: Array<{
    category: string
    count: number
    savedHours: number
  }>
}>()

const { t } = useI18n()

const COLORS = ['#06b6d4', '#8b5cf6', '#10b981', '#f59e0b', '#ec4899', '#3b82f6', '#f97316', '#14b8a6']

const chartOption = computed(() => {
  const total = props.data.reduce((sum, d) => sum + d.count, 0)
  return {
    tooltip: {
      trigger: 'item' as const,
      backgroundColor: 'rgba(26, 32, 64, 0.95)',
      borderColor: 'rgba(255,255,255,0.1)',
      textStyle: { color: '#f1f5f9' },
      formatter: (params: { name: string; value: number; percent: number }) =>
        `${params.name}<br/>${t('cockpit.tasks')}: ${params.value} (${params.percent}%)`,
    },
    legend: {
      orient: 'vertical' as const,
      right: 10,
      top: 'center',
      textStyle: { color: '#94a3b8', fontSize: 11 },
    },
    series: [
      {
        type: 'pie' as const,
        radius: ['40%', '65%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 6,
          borderColor: 'transparent',
          borderWidth: 2,
        },
        label: {
          show: true,
          position: 'center' as const,
          formatter: `{total|${total}}\n{label|${t('cockpit.totalTasksLabel')}}`,
          rich: {
            total: { fontSize: 24, fontWeight: 'bold' as const, color: '#f1f5f9' },
            label: { fontSize: 12, color: '#94a3b8', padding: [4, 0, 0, 0] },
          },
        },
        emphasis: {
          label: { show: true },
        },
        data: props.data.map((d, i) => ({
          name: d.category,
          value: d.count,
          itemStyle: { color: COLORS[i % COLORS.length] },
        })),
      },
    ],
  }
})
</script>
