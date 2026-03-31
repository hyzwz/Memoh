<template>
  <div class="p-6 space-y-6 mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">
          预算管理
        </h1>
        <p class="text-sm text-muted-foreground mt-1">
          设置消费限额与告警，控制AI使用成本
        </p>
      </div>
      <Button
        size="sm"
        @click="showCreate = true"
      >
        + 新建预算
      </Button>
    </div>

    <!-- Summary cards -->
    <div
      v-if="budgets && budgets.length > 0"
      class="grid grid-cols-1 sm:grid-cols-3 gap-4"
    >
      <Card>
        <CardContent class="p-5">
          <div class="text-xs text-muted-foreground uppercase tracking-wider mb-2">
            预算总数
          </div>
          <p class="text-2xl font-bold tabular-nums">
            {{ budgets.length }}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="p-5">
          <div class="text-xs text-muted-foreground uppercase tracking-wider mb-2">
            拦截策略
          </div>
          <p class="text-2xl font-bold tabular-nums text-rose-600">
            {{ blockCount }}
          </p>
          <p class="text-xs text-muted-foreground">
            个 Block 规则
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="p-5">
          <div class="text-xs text-muted-foreground uppercase tracking-wider mb-2">
            告警策略
          </div>
          <p class="text-2xl font-bold tabular-nums text-amber-600">
            {{ warnCount }}
          </p>
          <p class="text-xs text-muted-foreground">
            个 Warn 规则
          </p>
        </CardContent>
      </Card>
    </div>

    <!-- Loading -->
    <template v-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <!-- Table -->
    <template v-else-if="budgets && budgets.length > 0">
      <div class="rounded-lg border overflow-hidden">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b bg-muted/50">
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground">
                范围
              </th>
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground">
                范围ID
              </th>
              <th class="p-3 text-left font-medium text-xs uppercase tracking-wider text-muted-foreground">
                周期
              </th>
              <th class="p-3 text-right font-medium text-xs uppercase tracking-wider text-muted-foreground">
                限额
              </th>
              <th class="p-3 text-center font-medium text-xs uppercase tracking-wider text-muted-foreground">
                告警阈值
              </th>
              <th class="p-3 text-center font-medium text-xs uppercase tracking-wider text-muted-foreground">
                超额操作
              </th>
              <th class="p-3 text-right font-medium text-xs uppercase tracking-wider text-muted-foreground">
                操作
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="b in budgets"
              :key="b.id"
              class="border-b hover:bg-muted/30 transition-colors"
            >
              <td class="p-3">
                <Badge
                  variant="outline"
                  class="text-xs"
                  :class="scopeBadgeClass(b.scope_type)"
                >
                  {{ scopeLabel(b.scope_type) }}
                </Badge>
              </td>
              <td class="p-3 text-muted-foreground font-mono text-xs">
                {{ shortId(b.scope_id) }}
              </td>
              <td class="p-3">
                {{ periodLabel(b.period) }}
              </td>
              <td class="p-3 text-right tabular-nums font-semibold">
                ${{ b.limit_amount?.toFixed(2) }}
              </td>
              <td class="p-3 text-center">
                <div class="inline-flex items-center gap-1.5">
                  <div class="w-16 h-1.5 rounded-full bg-muted overflow-hidden">
                    <div
                      class="h-full rounded-full bg-amber-500"
                      :style="{ width: `${(b.alert_threshold ?? 0.8) * 100}%` }"
                    />
                  </div>
                  <span class="text-xs tabular-nums text-muted-foreground">{{ ((b.alert_threshold ?? 0) * 100).toFixed(0) }}%</span>
                </div>
              </td>
              <td class="p-3 text-center">
                <Badge
                  :class="actionBadgeClass(b.action_on_exceed)"
                  class="text-xs"
                >
                  {{ actionLabel(b.action_on_exceed) }}
                </Badge>
              </td>
              <td class="p-3 text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive text-xs"
                  @click="handleDelete(b.id)"
                >
                  删除
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <!-- Empty -->
    <template v-else-if="!isLoading">
      <EmptyState
        icon="wallet"
        message="暂无预算配置"
        sub="创建预算规则后，系统将自动监控AI消费"
      />
    </template>

    <!-- Create dialog -->
    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>创建预算</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label>范围类型</Label>
              <Select v-model="form.scope_type">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="bot">
                    Bot
                  </SelectItem>
                  <SelectItem value="user">
                    用户
                  </SelectItem>
                  <SelectItem value="department">
                    部门
                  </SelectItem>
                  <SelectItem value="system">
                    全局
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-1.5">
              <Label>周期</Label>
              <Select v-model="form.period">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="daily">
                    每日
                  </SelectItem>
                  <SelectItem value="weekly">
                    每周
                  </SelectItem>
                  <SelectItem value="monthly">
                    每月
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div
            v-if="form.scope_type !== 'system'"
            class="space-y-1.5"
          >
            <Label>范围ID</Label>
            <Input
              v-model="form.scope_id"
              placeholder="bot-id, user-id, 部门名等"
            />
          </div>
          <div
            v-else
            class="rounded-md border border-dashed bg-muted/30 px-3 py-2 text-xs text-muted-foreground"
          >
            全局预算会自动使用系统作用域，不需要填写范围 ID。
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label>限额 ($)</Label>
              <Input
                v-model.number="form.limit_amount"
                type="number"
                min="0"
                step="0.01"
              />
            </div>
            <div class="space-y-1.5">
              <Label>告警阈值</Label>
              <Input
                v-model.number="form.alert_threshold"
                type="number"
                min="0"
                max="1"
                step="0.05"
              />
            </div>
          </div>
          <div class="space-y-1.5">
            <Label>超额操作</Label>
            <Select v-model="form.action_on_exceed">
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="warn">
                  告警 (Warn)
                </SelectItem>
                <SelectItem value="block">
                  拦截 (Block)
                </SelectItem>
                <SelectItem value="downgrade">
                  降级 (Downgrade)
                </SelectItem>
              </SelectContent>
            </Select>
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
            :disabled="!form.scope_type || !form.period || form.limit_amount <= 0 || (!isScopeIdOptional(form.scope_type) && !form.scope_id) || creating"
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
import { ref, computed, reactive } from 'vue'
import { useQuery } from '@pinia/colada'
import { toast } from 'vue-sonner'
import {
  Badge, Button, Card, CardContent,
  Dialog, DialogContent, DialogHeader, DialogTitle,
  Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue, Spinner,
} from '@memoh/ui'
import { client } from '@memoh/sdk/client'
import EmptyState from '../_components/EmptyState.vue'

const showCreate = ref(false)
const creating = ref(false)

const form = reactive({
  scope_type: 'bot', scope_id: '', period: 'monthly',
  limit_amount: 100, alert_threshold: 0.8, action_on_exceed: 'warn',
})

const { data: budgets, status, refetch } = useQuery({
  key: () => ['budgets'],
  query: async () => {
    const res = await client.get({ url: '/budgets', query: {} })
    if (res.error) throw new Error('Failed')
    return res.data ?? []
  },
})

const isLoading = computed(() => status.value === 'loading')
const blockCount = computed(() => budgets.value?.filter((b: Record<string, unknown>) => b.action_on_exceed === 'block').length ?? 0)
const warnCount = computed(() => budgets.value?.filter((b: Record<string, unknown>) => b.action_on_exceed === 'warn').length ?? 0)

function scopeLabel(s: string | undefined): string {
  return { bot: 'Bot', user: '用户', department: '部门', system: '全局' }[s ?? ''] ?? s ?? '-'
}
function scopeBadgeClass(s: string | undefined): string {
  return { bot: 'border-blue-200 text-blue-700', user: 'border-violet-200 text-violet-700', department: 'border-emerald-200 text-emerald-700', system: 'border-amber-200 text-amber-700' }[s ?? ''] ?? ''
}
function periodLabel(p: string | undefined): string {
  return { daily: '每日', weekly: '每周', monthly: '每月' }[p ?? ''] ?? p ?? '-'
}
function actionLabel(a: string | undefined): string {
  return { warn: '告警', block: '拦截', downgrade: '降级' }[a ?? ''] ?? a ?? '-'
}
function actionBadgeClass(a: string | undefined): string {
  return { warn: 'bg-amber-500/10 text-amber-700 border-amber-200', block: 'bg-rose-500/10 text-rose-700 border-rose-200', downgrade: 'bg-blue-500/10 text-blue-700 border-blue-200' }[a ?? ''] ?? ''
}
function isScopeIdOptional(scopeType: string | undefined): boolean {
  return scopeType === 'system'
}
function shortId(id: string | undefined): string {
  if (!id) return '-'
  return id.length > 16 ? id.slice(0, 8) + '...' + id.slice(-4) : id
}

async function handleCreate() {
  creating.value = true
  try {
    const scopeId = isScopeIdOptional(form.scope_type) ? 'system' : form.scope_id
    const res = await client.post({
      url: '/budgets',
      body: { ...form, scope_id: scopeId },
      headers: { 'Content-Type': 'application/json' },
    })
    if (res.error) throw res.error
    showCreate.value = false
    form.scope_id = ''
    form.scope_type = 'bot'
    form.period = 'monthly'
    form.alert_threshold = 0.8
    form.action_on_exceed = 'warn'
    form.limit_amount = 100
    refetch()
    toast.success('预算创建成功')
  } catch { toast.error('创建预算失败') } finally { creating.value = false }
}

async function handleDelete(id: string) {
  try {
    const res = await client.delete({
      url: '/budgets/{budget_id}',
      path: { budget_id: id },
    })
    if (res.error) throw res.error
    refetch()
    toast.success('预算已删除')
  } catch { toast.error('删除预算失败') }
}
</script>
