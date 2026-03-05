<template>
  <div class="p-6 space-y-6 mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">
        {{ $t('enterprise.modelRouting.title') }}
      </h1>
      <div class="flex items-center gap-3">
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-56">
            <SelectValue :placeholder="$t('enterprise.modelRouting.selectBot')" />
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
        <Button
          size="sm"
          :disabled="!selectedBotId"
          @click="showCreate = true"
        >
          {{ $t('common.add') }}
        </Button>
      </div>
    </div>

    <template v-if="!selectedBotId">
      <div class="text-muted-foreground text-center py-20">
        {{ $t('enterprise.modelRouting.selectBot') }}
      </div>
    </template>

    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <template v-else-if="routes.length > 0">
      <div class="rounded-md border">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b bg-muted/50">
              <th class="p-3 text-left font-medium">{{ $t('common.name') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.modelRouting.model') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.modelRouting.complexity') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.modelRouting.priority') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('common.status') }}</th>
              <th class="p-3 text-right font-medium">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="route in routes"
              :key="route.id"
              class="border-b"
            >
              <td class="p-3 font-medium">{{ route.name }}</td>
              <td class="p-3 text-muted-foreground">{{ route.model_id }}</td>
              <td class="p-3">
                <Badge variant="outline">
                  {{ route.complexity_tier }}
                </Badge>
              </td>
              <td class="p-3 tabular-nums">{{ route.priority }}</td>
              <td class="p-3">
                <Badge :variant="route.is_enabled ? 'default' : 'secondary'">
                  {{ route.is_enabled ? $t('common.enable') : $t('enterprise.modelRouting.disabled') }}
                </Badge>
              </td>
              <td class="p-3 text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  @click="handleDelete(route.id)"
                >
                  {{ $t('common.delete') }}
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else>
      <div class="text-muted-foreground text-center py-12">
        {{ $t('enterprise.modelRouting.noData') }}
      </div>
    </template>

    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('enterprise.modelRouting.createTitle') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="space-y-1.5">
            <Label>{{ $t('common.name') }}</Label>
            <Input
              v-model="form.name"
              :placeholder="$t('common.namePlaceholder')"
            />
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.modelRouting.model') }}</Label>
            <Input
              v-model="form.model_id"
              placeholder="gpt-4, claude-3, etc."
            />
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.modelRouting.complexity') }}</Label>
            <Select v-model="form.complexity_tier">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="simple">Simple</SelectItem>
                <SelectItem value="medium">Medium</SelectItem>
                <SelectItem value="complex">Complex</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.modelRouting.priority') }}</Label>
            <Input
              v-model.number="form.priority"
              type="number"
            />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <Button
            variant="outline"
            @click="showCreate = false"
          >
            {{ $t('common.cancel') }}
          </Button>
          <Button
            :disabled="!form.name || !form.model_id || creating"
            @click="handleCreate"
          >
            <Spinner
              v-if="creating"
              class="mr-2 size-4"
            />
            {{ $t('common.confirm') }}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, watch } from 'vue'
import { useQuery } from '@pinia/colada'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import {
  Badge,
  Button,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
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

const { t } = useI18n()
const selectedBotId = useSyncedQueryParam('bot', '')
const showCreate = ref(false)
const creating = ref(false)

const form = reactive({
  name: '',
  model_id: '',
  complexity_tier: 'medium',
  priority: 10,
})

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) {
    selectedBotId.value = list[0].id
  }
}, { immediate: true })

const { data: routes, status, refetch } = useQuery({
  key: () => ['model-routes', selectedBotId.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/model-routes', {
      params: { path: { bot_id: selectedBotId.value } },
    })
    if (res.error) throw new Error('Failed to load model routes')
    return res.data ?? []
  },
  enabled: () => !!selectedBotId.value,
})

const isLoading = computed(() => status.value === 'loading')

async function handleCreate() {
  creating.value = true
  try {
    const res = await client.POST('/bots/{bot_id}/model-routes', {
      params: { path: { bot_id: selectedBotId.value } },
      body: { ...form },
    })
    if (res.error) throw res.error
    showCreate.value = false
    form.name = ''
    form.model_id = ''
    form.complexity_tier = 'medium'
    form.priority = 10
    refetch()
    toast.success(t('enterprise.modelRouting.createSuccess'))
  } catch {
    toast.error(t('enterprise.modelRouting.createFailed'))
  } finally {
    creating.value = false
  }
}

async function handleDelete(id: string) {
  try {
    const res = await client.DELETE('/bots/{bot_id}/model-routes/{route_id}', {
      params: { path: { bot_id: selectedBotId.value, route_id: id } },
    })
    if (res.error) throw res.error
    refetch()
    toast.success(t('enterprise.modelRouting.deleteSuccess'))
  } catch {
    toast.error(t('enterprise.modelRouting.deleteFailed'))
  }
}
</script>
