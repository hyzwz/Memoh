<template>
  <div class="p-6 space-y-6 mx-auto">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold tracking-tight">
        {{ $t('enterprise.budgets.title') }}
      </h1>
      <Button
        size="sm"
        @click="showCreate = true"
      >
        {{ $t('common.add') }}
      </Button>
    </div>

    <template v-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <template v-else-if="budgets.length > 0">
      <div class="rounded-md border">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b bg-muted/50">
              <th class="p-3 text-left font-medium">{{ $t('enterprise.budgets.scopeType') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.budgets.scopeId') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.budgets.period') }}</th>
              <th class="p-3 text-right font-medium">{{ $t('enterprise.budgets.limit') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.budgets.alertThreshold') }}</th>
              <th class="p-3 text-left font-medium">{{ $t('enterprise.budgets.action') }}</th>
              <th class="p-3 text-right font-medium">{{ $t('common.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="b in budgets"
              :key="b.id"
              class="border-b"
            >
              <td class="p-3">
                <Badge variant="outline">{{ b.scope_type }}</Badge>
              </td>
              <td class="p-3 text-muted-foreground">{{ b.scope_id }}</td>
              <td class="p-3">{{ b.period }}</td>
              <td class="p-3 text-right tabular-nums">${{ b.limit_amount?.toFixed(2) }}</td>
              <td class="p-3 tabular-nums">{{ ((b.alert_threshold ?? 0) * 100).toFixed(0) }}%</td>
              <td class="p-3">
                <Badge variant="secondary">{{ b.action_on_exceed || '-' }}</Badge>
              </td>
              <td class="p-3 text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  class="text-destructive"
                  @click="handleDelete(b.id)"
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
        {{ $t('enterprise.budgets.noData') }}
      </div>
    </template>

    <Dialog v-model:open="showCreate">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ $t('enterprise.budgets.createTitle') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.budgets.scopeType') }}</Label>
            <Select v-model="form.scope_type">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="bot">Bot</SelectItem>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="department">Department</SelectItem>
                <SelectItem value="global">Global</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.budgets.scopeId') }}</Label>
            <Input
              v-model="form.scope_id"
              placeholder="bot-1, user-123, etc."
            />
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.budgets.period') }}</Label>
            <Select v-model="form.period">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="daily">Daily</SelectItem>
                <SelectItem value="weekly">Weekly</SelectItem>
                <SelectItem value="monthly">Monthly</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.budgets.limit') }} ($)</Label>
            <Input
              v-model.number="form.limit_amount"
              type="number"
              min="0"
              step="0.01"
            />
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.budgets.alertThreshold') }} (0-1)</Label>
            <Input
              v-model.number="form.alert_threshold"
              type="number"
              min="0"
              max="1"
              step="0.05"
            />
          </div>
          <div class="space-y-1.5">
            <Label>{{ $t('enterprise.budgets.action') }}</Label>
            <Select v-model="form.action_on_exceed">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="warn">Warn</SelectItem>
                <SelectItem value="block">Block</SelectItem>
                <SelectItem value="downgrade">Downgrade</SelectItem>
              </SelectContent>
            </Select>
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
            :disabled="!form.scope_type || !form.scope_id || !form.period || form.limit_amount <= 0 || creating"
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
import { ref, computed, reactive } from 'vue'
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
import { client } from '@memoh/sdk/client'

const { t } = useI18n()
const showCreate = ref(false)
const creating = ref(false)

const form = reactive({
  scope_type: 'bot',
  scope_id: '',
  period: 'monthly',
  limit_amount: 100,
  alert_threshold: 0.8,
  action_on_exceed: 'warn',
})

const { data: budgets, status, refetch } = useQuery({
  key: () => ['budgets'],
  query: async () => {
    const res = await client.GET('/budgets', {
      params: { query: {} },
    })
    if (res.error) throw new Error('Failed to load budgets')
    return res.data ?? []
  },
})

const isLoading = computed(() => status.value === 'loading')

async function handleCreate() {
  creating.value = true
  try {
    const res = await client.POST('/budgets', {
      body: { ...form },
    })
    if (res.error) throw res.error
    showCreate.value = false
    form.scope_id = ''
    form.limit_amount = 100
    refetch()
    toast.success(t('enterprise.budgets.createSuccess'))
  } catch {
    toast.error(t('enterprise.budgets.createFailed'))
  } finally {
    creating.value = false
  }
}

async function handleDelete(id: string) {
  try {
    const res = await client.DELETE('/budgets/{budget_id}', {
      params: { path: { budget_id: id } },
    })
    if (res.error) throw res.error
    refetch()
    toast.success(t('enterprise.budgets.deleteSuccess'))
  } catch {
    toast.error(t('enterprise.budgets.deleteFailed'))
  }
}
</script>
