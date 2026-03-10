<template>
  <div class="glass-panel p-5">
    <h3 class="text-sm font-medium text-text-secondary mb-4">
      {{ $t('cockpit.efficiencyTrend') }}
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
    date: string
    savedHours: number
    multiplier: number
  }>
}>()

const { t } = useI18n()

const chartOption = computed(() => {
  const dates = props.data.map(d => d.date.slice(5))
  return {
    tooltip: {
      trigger: 'axis' as const,
      backgroundColor: 'rgba(26, 32, 64, 0.95)',
      borderColor: 'rgba(255,255,255,0.1)',
      textStyle: { color: '#f1f5f9' },
    },
    grid: { left: 50, right: 50, bottom: 30, top: 20 },
    xAxis: {
      type: 'category' as const,
      data: dates,
      axisLabel: { color: '#94a3b8', fontSize: 11 },
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } },
    },
    yAxis: [
      {
        type: 'value' as const,
        name: t('cockpit.savedHoursUnit'),
        nameTextStyle: { color: '#94a3b8', fontSize: 11 },
        axisLabel: { color: '#94a3b8', fontSize: 11 },
        splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)', type: 'dashed' as const } },
      },
      {
        type: 'value' as const,
        name: t('cockpit.multiplierUnit'),
        nameTextStyle: { color: '#94a3b8', fontSize: 11 },
        axisLabel: { color: '#94a3b8', fontSize: 11, formatter: '{value}x' },
        splitLine: { show: false },
        position: 'right' as const,
      },
    ],
    series: [
      {
        name: t('cockpit.savedHours'),
        type: 'line' as const,
        yAxisIndex: 0,
        areaStyle: { color: 'rgba(6,182,212,0.15)' },
        lineStyle: { color: '#06b6d4', width: 2 },
        itemStyle: { color: '#06b6d4' },
        smooth: true,
        data: props.data.map(d => d.savedHours),
      },
      {
        name: t('cockpit.multiplier'),
        type: 'line' as const,
        yAxisIndex: 1,
        lineStyle: { color: '#8b5cf6', width: 2 },
        itemStyle: { color: '#8b5cf6' },
        smooth: true,
        data: props.data.map(d => d.multiplier),
      },
    ],
  }
})
</script>
