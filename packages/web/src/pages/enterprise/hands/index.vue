<template>
  <div class="p-6 space-y-6 mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">Hands 管理</h1>
        <p class="text-sm text-muted-foreground mt-1">管理Bot的自主执行单元，实现自动化任务</p>
      </div>
      <div class="flex items-center gap-3">
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
        <Button size="sm" :disabled="!selectedBotId" @click="showCreate = true">
          + 新建Hand
        </Button>
      </div>
    </div>

    <!-- No bot -->
    <template v-if="!selectedBotId">
      <EmptyState icon="hand" message="请先选择一个Bot管理Hands" />
    </template>

    <!-- Loading -->
    <template v-else-if="isLoading">
      <div class="flex justify-center py-20"><Spinner class="size-8" /></div>
    </template>

    <!-- Cards -->
    <template v-else-if="hands && hands.length > 0">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <Card v-for="hand in hands" :key="hand.id" class="flex flex-col relative overflow-hidden group">
          <div class="absolute top-0 left-0 right-0 h-1" :class="hand.is_enabled ? 'bg-emerald-500' : 'bg-gray-300'" />
          <CardHeader class="pb-2">
            <div class="flex items-center justify-between">
              <CardTitle class="text-base">{{ hand.name }}</CardTitle>
              <Badge :class="hand.is_enabled ? 'bg-emerald-500/10 text-emerald-700 border-emerald-200' : 'bg-muted text-muted-foreground'" class="text-xs">
                {{ hand.is_enabled ? '启用' : '禁用' }}
              </Badge>
            </div>
            <p class="text-sm text-muted-foreground line-clamp-2">
              {{ hand.description || '暂无描述' }}
            </p>
          </CardHeader>
          <CardContent class="flex-1">
            <div class="flex flex-wrap gap-2">
              <Badge v-if="hand.type" variant="outline" class="text-xs">{{ hand.type }}</Badge>
              <Badge v-if="hand.trigger_type" variant="outline" class="text-xs border-blue-200 text-blue-700">
                触发: {{ hand.trigger_type }}
              </Badge>
            </div>
          </CardContent>
          <div class="px-4 pb-4 flex gap-2">
            <Button
              variant="outline" size="sm" class="flex-1 text-xs"
              :disabled="executing === hand.id"
              @click="handleExecute(hand.id)"
            >
              <Spinner v-if="executing === hand.id" class="mr-1.5 size-3" />
              <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-3 mr-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347a1.125 1.125 0 0 1-1.667-.986V5.653Z" /></svg>
              执行
            </Button>
            <Button variant="ghost" size="sm" class="text-destructive text-xs" @click="handleDelete(hand.id)">
              删除
            </Button>
          </div>
        </Card>
      </div>
    </template>

    <!-- Empty -->
    <template v-else>
      <EmptyState icon="hand" message="暂无Hand配置" sub="创建Hand后，Bot可以自主执行自动化任务" />
    </template>

    <!-- Create dialog -->
    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>创建Hand</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="space-y-1.5">
            <Label>Hand Markdown</Label>
            <p class="text-xs text-muted-foreground">使用YAML frontmatter定义Hand属性，正文描述功能</p>
            <Textarea
              v-model="markdownInput"
              class="min-h-[200px] font-mono text-sm"
              :placeholder="markdownPlaceholder"
            />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <Button variant="outline" @click="showCreate = false">取消</Button>
          <Button :disabled="!markdownInput.trim() || creating" @click="handleCreate">
            <Spinner v-if="creating" class="mr-2 size-4" />
            确认
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
import {
  Badge, Button, Card, CardContent, CardHeader, CardTitle,
  Dialog, DialogContent, DialogHeader, DialogTitle,
  Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
  Spinner, Textarea,
} from '@memoh/ui'
import { getBotsQuery } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'
import { useSyncedQueryParam } from '@/composables/useSyncedQueryParam'
import EmptyState from '../_components/EmptyState.vue'

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
  if (!selectedBotId.value && list.length > 0 && list[0].id) selectedBotId.value = list[0].id
}, { immediate: true })

const { data: hands, status, refetch } = useQuery({
  key: () => ['hands', selectedBotId.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/hands', {
      params: { path: { bot_id: selectedBotId.value } },
    })
    if (res.error) throw new Error('Failed')
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
    toast.success('Hand创建成功')
  } catch { toast.error('创建Hand失败') } finally { creating.value = false }
}

async function handleExecute(handId: string) {
  executing.value = handId
  try {
    const res = await client.POST('/bots/{bot_id}/hands/{hand_id}/execute', {
      params: { path: { bot_id: selectedBotId.value, hand_id: handId } },
    })
    if (res.error) throw res.error
    toast.success('Hand执行成功')
  } catch { toast.error('执行Hand失败') } finally { executing.value = null }
}

async function handleDelete(handId: string) {
  try {
    await client.DELETE('/bots/{bot_id}/hands/{hand_id}', {
      params: { path: { bot_id: selectedBotId.value, hand_id: handId } },
    })
    refetch()
    toast.success('Hand已删除')
  } catch { toast.error('删除Hand失败') }
}
</script>
