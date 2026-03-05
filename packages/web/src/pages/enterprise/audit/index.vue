<template>
  <div class="p-6 space-y-6 mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">
        {{ $t('enterprise.audit.title') }}
      </h1>
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

    <div class="flex flex-wrap items-end gap-4">
      <div class="space-y-1.5">
        <Label>{{ $t('enterprise.audit.selectBot') }}</Label>
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-56">
            <SelectValue :placeholder="$t('enterprise.audit.selectBotPlaceholder')" />
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

      <div class="space-y-1.5">
        <Label>{{ $t('enterprise.audit.filterUser') }}</Label>
        <Input
          v-model="filterUserId"
          class="w-48"
          :placeholder="$t('enterprise.audit.userIdPlaceholder')"
        />
      </div>

      <div class="space-y-1.5">
        <Label>{{ $t('enterprise.audit.filterAction') }}</Label>
        <Input
          v-model="filterAction"
          class="w-48"
          :placeholder="$t('enterprise.audit.actionPlaceholder')"
        />
      </div>
    </div>

    <template v-if="!selectedBotId">
      <div class="text-muted-foreground text-center py-20">
        {{ $t('enterprise.audit.selectBotPlaceholder') }}
      </div>
    </template>

    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <template v-else-if="logs.length > 0">
      <div class="rounded-md border">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b bg-muted/50">
              <th class="p-3 text-left font-medium">{{ $t('enterprise.audit.timestamp') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.audit.user') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.audit.action') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.audit.resource') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.audit.details') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="log in logs"
              :key="log.id"
              class="border-b"
            >
              <td class="p-3 tabular-nums text-muted-foreground whitespace-nowrap">
                {{ formatTime(log.timestamp) }}
              </td>
              <td class="p-3">{{ log.user_id }}</td>
              <td class="p-3">
                <Badge variant="outline">
                  {{ log.action }}
                </Badge>
              </td>
              <td class="p-3 text-muted-foreground">{{ log.resource_type }}/{{ log.resource_id }}</td>
              <td class="p-3 text-muted-foreground max-w-xs truncate">{{ log.details }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else>
      <div class="text-muted-foreground text-center py-12">
        {{ $t('enterprise.audit.noData') }}
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
const filterAction = ref('')

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) {
    selectedBotId.value = list[0].id
  }
}, { immediate: true })

const { data: logs, status, refetch } = useQuery({
  key: () => ['audit-logs', selectedBotId.value, filterUserId.value, filterAction.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/audit-logs', {
      params: {
        path: { bot_id: selectedBotId.value },
        query: {
          limit: 100,
          offset: 0,
          user_id: filterUserId.value || undefined,
          action: filterAction.value || undefined,
        },
      },
    })
    if (res.error) throw new Error('Failed to load audit logs')
    return res.data ?? []
  },
  enabled: () => !!selectedBotId.value,
})

const isLoading = computed(() => status.value === 'loading')

function formatTime(ts: string | undefined): string {
  if (!ts) return '-'
  return new Date(ts).toLocaleString()
}
</script>
