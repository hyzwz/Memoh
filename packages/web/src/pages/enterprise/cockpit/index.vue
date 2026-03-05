<template>
  <div class="p-6 space-y-6 mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">
        {{ $t('enterprise.cockpit.title') }}
      </h1>
      <div class="flex items-center gap-3">
        <div class="space-y-1">
          <Select v-model="selectedBotId">
            <SelectTrigger class="w-56">
              <SelectValue :placeholder="$t('enterprise.cockpit.selectBot')" />
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
        <Select v-model="days">
          <SelectTrigger class="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="7">
              {{ $t('enterprise.cockpit.last7Days') }}
            </SelectItem>
            <SelectItem value="30">
              {{ $t('enterprise.cockpit.last30Days') }}
            </SelectItem>
            <SelectItem value="90">
              {{ $t('enterprise.cockpit.last90Days') }}
            </SelectItem>
          </SelectContent>
        </Select>
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

    <template v-if="!selectedBotId">
      <div class="text-muted-foreground text-center py-20">
        {{ $t('enterprise.cockpit.selectBot') }}
      </div>
    </template>

    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <template v-else-if="summary">
      <div class="grid grid-cols-2 lg:grid-cols-3 gap-4">
        <Card>
          <CardHeader class="pb-2">
            <CardDescription>{{ $t('enterprise.cockpit.savedHours') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <p class="text-2xl font-bold tabular-nums">
              {{ summary.total_saved_hours?.toFixed(1) ?? '0' }}h
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader class="pb-2">
            <CardDescription>{{ $t('enterprise.cockpit.aiTime') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <p class="text-2xl font-bold tabular-nums">
              {{ summary.total_ai_time_minutes ?? 0 }}min
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader class="pb-2">
            <CardDescription>{{ $t('enterprise.cockpit.multiplier') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <p class="text-2xl font-bold tabular-nums">
              {{ summary.average_efficiency_multiple?.toFixed(1) ?? '0' }}x
            </p>
          </CardContent>
        </Card>
      </div>

      <div class="grid grid-cols-2 lg:grid-cols-3 gap-4">
        <Card>
          <CardHeader class="pb-2">
            <CardDescription>{{ $t('enterprise.cockpit.totalReports') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <p class="text-2xl font-bold tabular-nums">
              {{ summary.total_reports ?? 0 }}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader class="pb-2">
            <CardDescription>{{ $t('enterprise.cockpit.totalInnovations') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <p class="text-2xl font-bold tabular-nums">
              {{ summary.total_innovations ?? 0 }}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader class="pb-2">
            <CardDescription>{{ $t('enterprise.cockpit.laborCostSaved') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <p class="text-2xl font-bold tabular-nums">
              {{ formatCNY(summary.labor_cost_cny ?? 0) }}
            </p>
          </CardContent>
        </Card>
      </div>
    </template>

    <template v-else>
      <div class="text-muted-foreground text-center py-12">
        {{ $t('enterprise.cockpit.noData') }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useQuery } from '@pinia/colada'
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
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
const days = useSyncedQueryParam('days', '30')

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

function formatCNY(n: number): string {
  return '\u00A5' + n.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
</script>
