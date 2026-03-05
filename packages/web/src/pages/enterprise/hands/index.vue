<template>
  <div class="p-6 space-y-6 mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">
        {{ $t('enterprise.hands.title') }}
      </h1>
      <div class="flex items-center gap-3">
        <Select v-model="selectedBotId">
          <SelectTrigger class="w-56">
            <SelectValue :placeholder="$t('enterprise.hands.selectBot')" />
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
        {{ $t('enterprise.hands.selectBot') }}
      </div>
    </template>

    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <template v-else-if="hands.length > 0">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <Card
          v-for="hand in hands"
          :key="hand.id"
          class="flex flex-col"
        >
          <CardHeader>
            <div class="flex items-center justify-between">
              <CardTitle class="text-base">{{ hand.name }}</CardTitle>
              <Badge :variant="hand.is_enabled ? 'default' : 'secondary'">
                {{ hand.is_enabled ? $t('common.enable') : $t('enterprise.hands.disabled') }}
              </Badge>
            </div>
            <CardDescription>{{ hand.description || $t('enterprise.hands.noDescription') }}</CardDescription>
          </CardHeader>
          <CardContent class="flex-1">
            <div class="text-xs text-muted-foreground space-y-1">
              <div>{{ $t('common.type') }}: {{ hand.type }}</div>
              <div v-if="hand.trigger_type">
                {{ $t('enterprise.hands.trigger') }}: {{ hand.trigger_type }}
              </div>
            </div>
          </CardContent>
          <div class="p-4 pt-0 flex gap-2">
            <Button
              variant="outline"
              size="sm"
              class="flex-1"
              :disabled="executing === hand.id"
              @click="handleExecute(hand.id)"
            >
              <Spinner
                v-if="executing === hand.id"
                class="mr-2 size-3"
              />
              {{ $t('enterprise.hands.execute') }}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="text-destructive"
              @click="handleDelete(hand.id)"
            >
              {{ $t('common.delete') }}
            </Button>
          </div>
        </Card>
      </div>
    </template>

    <template v-else>
      <div class="text-muted-foreground text-center py-12">
        {{ $t('enterprise.hands.noData') }}
      </div>
    </template>

    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ $t('enterprise.hands.createTitle') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.hands.markdown') }}</Label>
            <Textarea
              v-model="markdownInput"
              class="min-h-[200px] font-mono text-sm"
              :placeholder="markdownPlaceholder"
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
            :disabled="!markdownInput.trim() || creating"
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
import { ref, computed, watch } from 'vue'
import { useQuery } from '@pinia/colada'
import { toast } from 'vue-sonner'
import { useI18n } from 'vue-i18n'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Spinner,
  Textarea,
} from '@memoh/ui'
import { getBotsQuery } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'
import { useSyncedQueryParam } from '@/composables/useSyncedQueryParam'

const { t } = useI18n()
const selectedBotId = useSyncedQueryParam('bot', '')
const showCreate = ref(false)
const creating = ref(false)
const executing = ref<string | null>(null)
const markdownInput = ref('')

const markdownPlaceholder = `---
name: my-hand
type: hand
---
Describe what this hand does.`

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) {
    selectedBotId.value = list[0].id
  }
}, { immediate: true })

const { data: hands, status, refetch } = useQuery({
  key: () => ['hands', selectedBotId.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/hands', {
      params: { path: { bot_id: selectedBotId.value } },
    })
    if (res.error) throw new Error('Failed to load hands')
    return res.data ?? []
  },
  enabled: () => !!selectedBotId.value,
})

const isLoading = computed(() => status.value === 'loading')

async function handleCreate() {
  creating.value = true
  try {
    const res = await client.POST('/bots/{bot_id}/hands', {
      params: { path: { bot_id: selectedBotId.value } },
      body: { markdown: markdownInput.value },
    })
    if (res.error) throw res.error
    showCreate.value = false
    markdownInput.value = ''
    refetch()
    toast.success(t('enterprise.hands.createSuccess'))
  } catch {
    toast.error(t('enterprise.hands.createFailed'))
  } finally {
    creating.value = false
  }
}

async function handleExecute(handId: string) {
  executing.value = handId
  try {
    const res = await client.POST('/bots/{bot_id}/hands/{hand_id}/execute', {
      params: { path: { bot_id: selectedBotId.value, hand_id: handId } },
    })
    if (res.error) throw res.error
    toast.success(t('enterprise.hands.executeSuccess'))
  } catch {
    toast.error(t('enterprise.hands.executeFailed'))
  } finally {
    executing.value = null
  }
}

async function handleDelete(handId: string) {
  try {
    const res = await client.DELETE('/bots/{bot_id}/hands/{hand_id}', {
      params: { path: { bot_id: selectedBotId.value, hand_id: handId } },
    })
    if (res.error) throw res.error
    refetch()
    toast.success(t('enterprise.hands.deleteSuccess'))
  } catch {
    toast.error(t('enterprise.hands.deleteFailed'))
  }
}
</script>
