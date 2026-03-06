<template>
  <div class="p-6 space-y-6 mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">审计日志</h1>
        <p class="text-sm text-muted-foreground mt-1">追踪所有用户操作，确保合规与安全</p>
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

    <!-- Filters -->
    <div class="flex flex-wrap items-end gap-4">
      <div class="space-y-1.5">
        <Label class="text-xs text-muted-foreground">Bot</Label>
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-56">
            <SelectValue placeholder="选择Bot" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="bot in botList" :key="bot.id" :value="bot.id!">
              {{ bot.display_name || bot.id }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="space-y-1.5">
        <Label class="text-xs text-muted-foreground">用户ID</Label>
        <Input v-model="filterUserId" class="w-48" placeholder="按用户筛选" />
      </div>
      <div class="space-y-1.5">
        <Label class="text-xs text-muted-foreground">操作类型</Label>
        <Select v-model="filterAction">
          <SelectTrigger class="w-36">
            <SelectValue placeholder="全部" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">全部</SelectItem>
            <SelectItem value="create">创建</SelectItem>
            <SelectItem value="update">更新</SelectItem>
            <SelectItem value="delete">删除</SelectItem>
            <SelectItem value="execute">执行</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <!-- Summary bar -->
    <div v-if="selectedBotId && logEntries.length > 0" class="flex items-center gap-6 text-sm text-muted-foreground">
      <span>共 <strong class="text-foreground">{{ total }}</strong> 条记录</span>
      <span v-for="(count, action) in actionCounts" :key="action">
        {{ actionLabel(String(action)) }}: <strong class="text-foreground">{{ count }}</strong>
      </span>
    </div>

    <!-- No bot selected -->
    <template v-if="!selectedBotId">
      <div class="flex flex-col items-center justify-center py-20 text-muted-foreground">
        <svg xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 0 0 2.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 0 0-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 0 0 .75-.75 2.25 2.25 0 0 0-.1-.664m-5.8 0A2.251 2.251 0 0 1 13.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25ZM6.75 12h.008v.008H6.75V12Zm0 3h.008v.008H6.75V15Zm0 3h.008v.008H6.75V18Z" /></svg>
        <p class="text-sm">请先选择一个Bot查看审计日志</p>
      </div>
    </template>

    <!-- Loading -->
    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <!-- Table -->
    <template v-else-if="logEntries.length > 0">
      <div class="rounded-lg border overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b bg-muted/50">
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground">时间</th>
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground">用户</th>
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground">操作</th>
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground">资源</th>
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground hidden md:table-cell">详情</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logEntries" :key="log.id" class="border-b hover:bg-muted/30 transition-colors">
              <td class="p-3 tabular-nums text-muted-foreground whitespace-nowrap text-xs">
                {{ formatTime(log.timestamp) }}
              </td>
              <td class="p-3 font-mono text-xs">{{ shortId(log.user_id) }}</td>
              <td class="p-3">
                <Badge :class="actionBadgeClass(log.action)">
                  {{ actionLabel(log.action) }}
                </Badge>
              </td>
              <td class="p-3 text-muted-foreground text-xs">
                <span class="font-medium text-foreground">{{ log.resource_type }}</span>
                <span v-if="log.resource_id" class="text-muted-foreground"> / {{ shortId(log.resource_id) }}</span>
              </td>
              <td class="p-3 text-muted-foreground max-w-xs truncate text-xs hidden md:table-cell">{{ log.details || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <!-- Empty -->
    <template v-else>
      <div class="flex flex-col items-center justify-center py-20 text-muted-foreground">
        <svg xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 0 0 2.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 0 0-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 0 0 .75-.75 2.25 2.25 0 0 0-.1-.664m-5.8 0A2.251 2.251 0 0 1 13.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25ZM6.75 12h.008v.008H6.75V12Zm0 3h.008v.008H6.75V15Zm0 3h.008v.008H6.75V18Z" /></svg>
        <p class="text-sm">暂无审计日志</p>
        <p class="text-xs mt-1">操作记录将在用户执行变更后自动生成</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useQuery } from '@pinia/colada'
import {
  Badge,
  Button,
  Input,
  Label,
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
const filterUserId = ref('')
const filterAction = ref('__all__')

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) {
    selectedBotId.value = list[0].id
  }
}, { immediate: true })

const { data: rawData, status, refetch } = useQuery({
  key: () => ['audit-logs', selectedBotId.value, filterUserId.value, filterAction.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/audit-logs', {
      params: {
        path: { bot_id: selectedBotId.value },
        query: {
          limit: 100,
          offset: 0,
          user_id: filterUserId.value || undefined,
          action: filterAction.value === '__all__' ? undefined : filterAction.value || undefined,
        },
      },
    })
    if (res.error) throw new Error('Failed to load audit logs')
    return res.data
  },
  enabled: () => !!selectedBotId.value,
})

const logEntries = computed(() => (rawData.value as any)?.logs ?? [])
const total = computed(() => (rawData.value as any)?.total ?? 0)
const isLoading = computed(() => status.value === 'loading')

const actionCounts = computed(() => {
  const counts: Record<string, number> = {}
  for (const log of logEntries.value) {
    const a = log.action || 'unknown'
    counts[a] = (counts[a] || 0) + 1
  }
  return counts
})

function actionLabel(action: string | undefined): string {
  const map: Record<string, string> = { create: '创建', update: '更新', delete: '删除', execute: '执行' }
  return map[action ?? ''] ?? action ?? '-'
}

function actionBadgeClass(action: string | undefined): string {
  const map: Record<string, string> = {
    create: 'bg-emerald-500/10 text-emerald-700 border-emerald-200',
    delete: 'bg-rose-500/10 text-rose-700 border-rose-200',
    execute: 'bg-blue-500/10 text-blue-700 border-blue-200',
    update: 'bg-amber-500/10 text-amber-700 border-amber-200',
  }
  return map[action ?? ''] ?? 'bg-muted text-muted-foreground'
}

function shortId(id: string | undefined): string {
  if (!id) return '-'
  if (id.length <= 12) return id
  return id.slice(0, 8) + '...'
}

function formatTime(ts: string | undefined): string {
  if (!ts) return '-'
  return new Date(ts).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}
</script>
