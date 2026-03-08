<template>
  <div class="p-6 space-y-6 mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">
          部门管理
        </h1>
        <p class="text-sm text-muted-foreground mt-1">
          管理组织架构，关联Bot与部门的访问权限
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
          + 新建部门
        </Button>
      </div>
    </div>

    <!-- No bot -->
    <template v-if="!selectedBotId">
      <EmptyState
        icon="building"
        message="请先选择一个Bot管理部门"
      />
    </template>

    <!-- Loading -->
    <template v-else-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <!-- Department cards -->
    <template v-else-if="departments && departments.length > 0">
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <Card
          v-for="dept in departments"
          :key="dept.id"
          class="relative overflow-hidden group"
        >
          <div class="absolute top-0 left-0 right-0 h-1 bg-blue-500" />
          <CardContent class="p-5">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center gap-3">
                <div class="size-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    class="size-5 text-blue-600"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  ><path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M3.75 21h16.5M4.5 3h15M5.25 3v18m13.5-18v18M9 6.75h1.5m-1.5 3h1.5m-1.5 3h1.5m3-6H15m-1.5 3H15m-1.5 3H15M9 21v-3.375c0-.621.504-1.125 1.125-1.125h3.75c.621 0 1.125.504 1.125 1.125V21"
                  /></svg>
                </div>
                <div>
                  <h3 class="font-semibold">
                    {{ dept.name }}
                  </h3>
                  <p
                    v-if="dept.description"
                    class="text-xs text-muted-foreground"
                  >
                    {{ dept.description }}
                  </p>
                </div>
              </div>
            </div>
            <div class="flex items-center justify-between text-xs text-muted-foreground">
              <div class="flex items-center gap-4">
                <span v-if="dept.parent_id">
                  上级: <span class="font-mono">{{ shortId(dept.parent_id) }}</span>
                </span>
                <span
                  v-else
                  class="text-muted-foreground/50"
                >顶级部门</span>
              </div>
              <span class="tabular-nums">{{ dept.created_at ? new Date(dept.created_at).toLocaleDateString('zh-CN') : '' }}</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </template>

    <!-- Empty -->
    <template v-else>
      <EmptyState
        icon="building"
        message="暂无部门"
        sub="创建部门后可关联Bot访问权限和预算管理"
      />
    </template>

    <!-- Create dialog -->
    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>创建部门</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="space-y-1.5">
            <Label>部门名称</Label>
            <Input
              v-model="form.name"
              placeholder="例: 放射科"
            />
          </div>
          <div class="space-y-1.5">
            <Label>描述 <span class="text-muted-foreground text-xs">(可选)</span></Label>
            <Input
              v-model="form.description"
              placeholder="部门职责描述"
            />
          </div>
          <div class="space-y-1.5">
            <Label>上级部门 <span class="text-muted-foreground text-xs">(可选)</span></Label>
            <Select
              v-if="departments && departments.length > 0"
              v-model="form.parent_id"
            >
              <SelectTrigger>
                <SelectValue placeholder="无 (顶级部门)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">
                  无 (顶级部门)
                </SelectItem>
                <SelectItem
                  v-for="d in departments"
                  :key="d.id"
                  :value="d.id!"
                >
                  {{ d.name }}
                </SelectItem>
              </SelectContent>
            </Select>
            <Input
              v-else
              v-model="form.parent_id"
              placeholder="上级部门ID (可选)"
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
            :disabled="!form.name.trim() || creating"
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
  Button, Card, CardContent,
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

const form = reactive({ name: '', description: '', parent_id: '' })

const { data: botData } = useQuery(getBotsQuery())
const botList = computed(() => botData.value?.items ?? [])

watch(botList, (list) => {
  if (!selectedBotId.value && list.length > 0 && list[0].id) selectedBotId.value = list[0].id
}, { immediate: true })

const { data: departments, status, refetch } = useQuery({
  key: () => ['departments', selectedBotId.value],
  query: async () => {
    const res = await client.GET('/bots/{bot_id}/departments', {
      params: { path: { bot_id: selectedBotId.value } },
    })
    if (res.error) throw new Error('Failed')
    return res.data ?? []
  },
  enabled: () => !!selectedBotId.value,
})

const isLoading = computed(() => status.value === 'loading')

function shortId(id: string | undefined): string {
  if (!id) return '-'
  return id.length > 12 ? id.slice(0, 8) + '...' : id
}

async function handleCreate() {
  creating.value = true
  try {
    const res = await client.POST('/bots/{bot_id}/departments', {
      params: { path: { bot_id: selectedBotId.value } },
      body: { name: form.name, description: form.description, parent_id: (form.parent_id && form.parent_id !== '__none__') ? form.parent_id : undefined },
    })
    if (res.error) throw res.error
    showCreate.value = false
    Object.assign(form, { name: '', description: '', parent_id: '' })
    refetch()
    toast.success('部门创建成功')
  } catch { toast.error('创建部门失败') } finally { creating.value = false }
}
</script>
