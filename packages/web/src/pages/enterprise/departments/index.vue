<template>
  <div class="p-6 space-y-6 mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">
        {{ $t('enterprise.departments.title') }}
      </h1>
      <div class="flex items-center gap-3">
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-56">
            <SelectValue :placeholder="$t('enterprise.departments.selectBot')" />
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
        {{ $t('enterprise.departments.selectBot') }}
      </div>
    </template>

    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <template v-else-if="departments.length > 0">
      <div class="rounded-md border">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b bg-muted/50">
              <th class="p-3 text-left font-medium">{{ $t('common.name') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.departments.parentId') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('common.createdAt') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="dept in departments"
              :key="dept.id"
              class="border-b"
            >
              <td class="p-3 font-medium">{{ dept.name }}</td>
              <td class="p-3 text-muted-foreground">{{ dept.parent_id || '-' }}</td>
              <td class="p-3 tabular-nums text-muted-foreground">
                {{ dept.created_at ? new Date(dept.created_at).toLocaleDateString() : '-' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <template v-else>
      <div class="text-muted-foreground text-center py-12">
        {{ $t('enterprise.departments.noData') }}
      </div>
    </template>

    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('enterprise.departments.createTitle') }}</DialogTitle>
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
            <Label>{{ $t('enterprise.departments.parentId') }} ({{ $t('common.optional') }})</Label>
            <Input
              v-model="form.parent_id"
              :placeholder="$t('enterprise.departments.parentIdPlaceholder')"
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
            :disabled="!form.name.trim() || creating"
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
  parent_id: '',
})

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) {
    selectedBotId.value = list[0].id
  }
}, { immediate: true })

const { data: departments, status, refetch } = useQuery({
  key: () => ['departments', selectedBotId.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/departments', {
      params: { path: { bot_id: selectedBotId.value } },
    })
    if (res.error) throw new Error('Failed to load departments')
    return res.data ?? []
  },
  enabled: () => !!selectedBotId.value,
})

const isLoading = computed(() => status.value === 'loading')

async function handleCreate() {
  creating.value = true
  try {
    const res = await client.POST('/bots/{bot_id}/departments', {
      params: { path: { bot_id: selectedBotId.value } },
      body: {
        name: form.name,
        parent_id: form.parent_id || undefined,
      },
    })
    if (res.error) throw res.error
    showCreate.value = false
    form.name = ''
    form.parent_id = ''
    refetch()
    toast.success(t('enterprise.departments.createSuccess'))
  } catch {
    toast.error(t('enterprise.departments.createFailed'))
  } finally {
    creating.value = false
  }
}
</script>
