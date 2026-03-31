<template>
  <div class="p-6 space-y-6 mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">
          模型路由
        </h1>
        <p class="text-sm text-muted-foreground mt-1">
          根据任务复杂度智能选择最优AI模型
        </p>
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
        <Button
          size="sm"
          :disabled="!selectedBotId"
          @click="showCreate = true"
        >
          + 新建路由
        </Button>
      </div>
    </div>

    <!-- No bot -->
    <template v-if="!selectedBotId">
      <EmptyState
        icon="route"
        message="请先选择一个Bot管理模型路由"
      />
    </template>

    <!-- Loading -->
    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <!-- Route cards -->
    <template v-else-if="routes && routes.length > 0">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <Card
          v-for="route in routes"
          :key="route.id"
          class="relative overflow-hidden"
        >
          <div
            class="absolute top-0 left-0 right-0 h-1"
            :class="tierColor(route.complexity_tier)"
          />
          <CardHeader class="pb-3">
            <div class="flex items-center justify-between">
              <CardTitle class="text-base">
                {{ route.name }}
              </CardTitle>
              <Badge
                :variant="route.is_enabled ? 'default' : 'secondary'"
                class="text-xs"
              >
                {{ route.is_enabled ? '启用' : '禁用' }}
              </Badge>
            </div>
          </CardHeader>
          <CardContent class="space-y-3">
            <div class="grid grid-cols-2 gap-3 text-sm">
              <div>
                <span class="text-xs text-muted-foreground block">模型ID</span>
                <span class="font-mono text-xs">{{ shortId(route.model_id) }}</span>
              </div>
              <div>
                <span class="text-xs text-muted-foreground block">优先级</span>
                <span class="font-bold tabular-nums">{{ route.priority }}</span>
              </div>
            </div>
            <div>
              <span class="text-xs text-muted-foreground">复杂度级别</span>
              <Badge
                variant="outline"
                class="ml-2 text-xs"
                :class="tierBadgeClass(route.complexity_tier)"
              >
                {{ tierLabel(route.complexity_tier) }}
              </Badge>
            </div>
          </CardContent>
          <div class="px-4 pb-4 flex justify-end">
            <Button
              variant="ghost"
              size="sm"
              class="text-destructive text-xs"
              @click="handleDelete(route.id)"
            >
              删除
            </Button>
          </div>
        </Card>
      </div>
    </template>

    <!-- Empty -->
    <template v-else>
      <EmptyState
        icon="route"
        message="暂无模型路由配置"
        sub="添加路由后，系统将根据任务复杂度自动选择模型"
      />
    </template>

    <!-- Create dialog -->
    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>创建模型路由</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="space-y-1.5">
            <Label>名称</Label>
            <Input
              v-model="form.name"
              placeholder="例: GPT-4o 高复杂度路由"
            />
          </div>
          <div class="space-y-1.5">
            <Label>模型ID</Label>
            <Select v-model="form.model_id">
              <SelectTrigger>
                <SelectValue placeholder="选择模型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="m in modelList"
                  :key="m.id"
                  :value="m.id!"
                >
                  {{ m.name || m.id }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-1.5">
            <Label>复杂度级别</Label>
            <Select v-model="form.complexity_tier">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="simple">
                  简单
                </SelectItem>
                <SelectItem value="medium">
                  中等
                </SelectItem>
                <SelectItem value="complex">
                  复杂
                </SelectItem>
                <SelectItem value="fallback">
                  兜底
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-1.5">
            <Label>优先级 <span class="text-muted-foreground text-xs">(数值越大越优先)</span></Label>
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
            取消
          </Button>
          <Button
            :disabled="!form.name || !form.model_id || creating"
            @click="handleCreate"
          >
            <Spinner
              v-if="creating"
              class="mr-2 size-4"
            />
            确认
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
import {
  Badge, Button, Card, CardContent, CardHeader, CardTitle,
  Dialog, DialogContent, DialogHeader, DialogTitle,
  Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner,
} from '@memoh/ui'
import { getBotsQuery } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'
import { useSyncedQueryParam } from '@/composables/useSyncedQueryParam'
import EmptyState from '../_components/EmptyState.vue'

const selectedBotId = useSyncedQueryParam('bot', '')
const showCreate = ref(false)
const creating = ref(false)

const form = reactive({ name: '', model_id: '', complexity_tier: 'medium', priority: 10 })

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) selectedBotId.value = list[0].id
}, { immediate: true })

const { data: modelData } = useQuery({
  key: () => ['models'],
  query: async () => {
    const res = await client.get({ url: '/models' })
    if (res.error) return []
    return res.data ?? []
  },
})
const modelList = computed(() => (modelData.value as Record<string, unknown>[]) ?? [])

const { data: routes, status, refetch } = useQuery({
  key: () => ['model-routes', selectedBotId.value],
  query: async () => {
    const res = await client.get({
      url: '/bots/{bot_id}/model-routes',
      path: { bot_id: selectedBotId.value },
    })
    if (res.error) throw new Error('Failed to load model routes')
    return res.data ?? []
  },
  enabled: () => !!selectedBotId.value,
})

const isLoading = computed(() => status.value === 'loading')

function tierLabel(tier: string | undefined): string {
  return { simple: '简单', medium: '中等', complex: '复杂', fallback: '兜底' }[tier ?? ''] ?? tier ?? '-'
}
function tierColor(tier: string | undefined): string {
  return { simple: 'bg-emerald-500', medium: 'bg-blue-500', complex: 'bg-violet-500', fallback: 'bg-gray-400' }[tier ?? ''] ?? 'bg-gray-300'
}
function tierBadgeClass(tier: string | undefined): string {
  return { simple: 'border-emerald-200 text-emerald-700', medium: 'border-blue-200 text-blue-700', complex: 'border-violet-200 text-violet-700', fallback: 'border-gray-200 text-gray-700' }[tier ?? ''] ?? ''
}
function shortId(id: string | undefined): string {
  if (!id) return '-'
  return id.length > 12 ? id.slice(0, 8) + '...' : id
}

async function handleCreate() {
  creating.value = true
  try {
    const res = await client.post({
      url: '/bots/{bot_id}/model-routes',
      path: { bot_id: selectedBotId.value },
      body: { ...form },
      headers: { 'Content-Type': 'application/json' },
    })
    if (res.error) throw res.error
    showCreate.value = false
    Object.assign(form, { name: '', model_id: '', complexity_tier: 'medium', priority: 10 })
    refetch()
    toast.success('模型路由创建成功')
  } catch {
    toast.error('创建模型路由失败')
  } finally {
    creating.value = false
  }
}

async function handleDelete(id: string) {
  try {
    const res = await client.delete({
      url: '/bots/{bot_id}/model-routes/{route_id}',
      path: { bot_id: selectedBotId.value, route_id: id },
    })
    if (res.error) throw res.error
    refetch()
    toast.success('模型路由已删除')
  } catch {
    toast.error('删除模型路由失败')
  }
}
</script>
