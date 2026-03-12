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
          class="relative overflow-hidden group cursor-pointer hover:ring-2 hover:ring-blue-500/30 transition-all"
          @click="openDetail(dept)"
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

    <!-- Department detail sheet -->
    <Sheet v-model:open="showDetail">
      <SheetContent
        side="right"
        class="sm:max-w-lg w-full overflow-y-auto"
      >
        <SheetHeader class="pr-8">
          <SheetTitle>{{ selectedDept?.name }}</SheetTitle>
          <p
            v-if="selectedDept?.description"
            class="text-sm text-muted-foreground"
          >
            {{ selectedDept.description }}
          </p>
        </SheetHeader>

        <div class="mt-6 space-y-8">
          <!-- Sync buttons -->
          <div class="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              :disabled="syncingSkills"
              @click="handleSyncSkills"
            >
              <Spinner
                v-if="syncingSkills"
                class="mr-2 size-3"
              />
              同步技能
            </Button>
            <Button
              size="sm"
              variant="outline"
              :disabled="syncingDirs"
              @click="handleSyncDirectories"
            >
              <Spinner
                v-if="syncingDirs"
                class="mr-2 size-3"
              />
              同步目录
            </Button>
          </div>

          <Separator />

          <!-- Skill templates section -->
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold">
                技能模板
              </h3>
              <Button
                size="sm"
                variant="outline"
                @click="showSkillPicker = true"
              >
                + 添加
              </Button>
            </div>

            <div
              v-if="loadingSkillTemplates"
              class="flex justify-center py-4"
            >
              <Spinner class="size-5" />
            </div>
            <div
              v-else-if="deptSkillTemplates.length === 0"
              class="text-sm text-muted-foreground py-4 text-center"
            >
              暂无关联的技能模板
            </div>
            <div
              v-else
              class="space-y-2"
            >
              <div
                v-for="tpl in deptSkillTemplates"
                :key="tpl.id"
                class="flex items-center justify-between p-3 rounded-lg border bg-muted/30"
              >
                <div class="min-w-0">
                  <div class="text-sm font-medium truncate">
                    {{ tpl.name }}
                  </div>
                  <div
                    v-if="tpl.description"
                    class="text-xs text-muted-foreground truncate"
                  >
                    {{ tpl.description }}
                  </div>
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  class="text-destructive hover:text-destructive shrink-0 ml-2"
                  :disabled="removingSkillId === tpl.id"
                  @click.stop="handleRemoveSkillTemplate(tpl.id!)"
                >
                  <Spinner
                    v-if="removingSkillId === tpl.id"
                    class="size-4"
                  />
                  <svg
                    v-else
                    xmlns="http://www.w3.org/2000/svg"
                    class="size-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  ><path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"
                  /></svg>
                </Button>
              </div>
            </div>
          </div>

          <Separator />

          <!-- Directory templates section -->
          <div class="space-y-3">
            <h3 class="text-sm font-semibold">
              目录模板
            </h3>

            <div
              v-if="loadingDirTemplates"
              class="flex justify-center py-4"
            >
              <Spinner class="size-5" />
            </div>
            <template v-else>
              <div
                v-if="dirPaths.length === 0"
                class="text-sm text-muted-foreground py-2 text-center"
              >
                暂无目录路径
              </div>
              <div
                v-else
                class="space-y-2"
              >
                <div
                  v-for="(p, idx) in dirPaths"
                  :key="idx"
                  class="flex items-center justify-between p-2 rounded-lg border bg-muted/30"
                >
                  <code class="text-sm font-mono truncate">{{ p }}</code>
                  <Button
                    size="sm"
                    variant="ghost"
                    class="text-destructive hover:text-destructive shrink-0 ml-2"
                    @click="removeDirPath(idx)"
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      class="size-4"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="1.5"
                    ><path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"
                    /></svg>
                  </Button>
                </div>
              </div>

              <!-- Add path input -->
              <div class="flex items-center gap-2">
                <Input
                  v-model="newDirPath"
                  placeholder="/data/example"
                  class="flex-1 font-mono text-sm"
                  @keydown.enter="addDirPath"
                />
                <Button
                  size="sm"
                  variant="outline"
                  :disabled="!isValidDirPath"
                  @click="addDirPath"
                >
                  添加
                </Button>
              </div>
              <p
                v-if="newDirPath && !isValidDirPath"
                class="text-xs text-destructive"
              >
                路径必须以 /data/ 开头，且不能包含 ..
              </p>

              <!-- Save button -->
              <div class="flex justify-end">
                <Button
                  size="sm"
                  :disabled="!dirPathsDirty || savingDirs"
                  @click="handleSaveDirTemplates"
                >
                  <Spinner
                    v-if="savingDirs"
                    class="mr-2 size-3"
                  />
                  保存
                </Button>
              </div>
            </template>
          </div>
        </div>
      </SheetContent>
    </Sheet>

    <!-- Skill template picker dialog -->
    <Dialog v-model:open="showSkillPicker">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>选择技能模板</DialogTitle>
        </DialogHeader>
        <div class="mt-4 space-y-3 max-h-80 overflow-y-auto">
          <div
            v-if="loadingAllTemplates"
            class="flex justify-center py-8"
          >
            <Spinner class="size-6" />
          </div>
          <div
            v-else-if="availableTemplates.length === 0"
            class="text-sm text-muted-foreground text-center py-8"
          >
            没有可添加的模板
          </div>
          <div
            v-for="tpl in availableTemplates"
            v-else
            :key="tpl.id"
            class="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/50 cursor-pointer transition-colors"
            @click="handleAddSkillTemplate(tpl.id!)"
          >
            <div class="min-w-0">
              <div class="text-sm font-medium truncate">
                {{ tpl.name }}
              </div>
              <div
                v-if="tpl.description"
                class="text-xs text-muted-foreground truncate"
              >
                {{ tpl.description }}
              </div>
            </div>
            <Button
              size="sm"
              variant="ghost"
              :disabled="addingSkillId === tpl.id"
            >
              <Spinner
                v-if="addingSkillId === tpl.id"
                class="size-4"
              />
              <span v-else>+ 添加</span>
            </Button>
          </div>
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
  Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
  Separator, Sheet, SheetContent, SheetHeader, SheetTitle, Spinner,
} from '@memoh/ui'
import { getBotsQuery } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'
import { useSyncedQueryParam } from '@/composables/useSyncedQueryParam'
import EmptyState from '../_components/EmptyState.vue'

import type { HandlersSkillTemplateBriefDto } from '@memoh/sdk'

interface Department {
  id?: string
  name?: string
  description?: string
  parent_id?: string
  created_at?: string
}

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
    const res = await client.get({
      url: '/bots/{bot_id}/departments',
      path: { bot_id: selectedBotId.value },
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
    const res = await client.post({
      url: '/bots/{bot_id}/departments',
      path: { bot_id: selectedBotId.value },
      body: { name: form.name, description: form.description, parent_id: (form.parent_id && form.parent_id !== '__none__') ? form.parent_id : undefined },
      headers: { 'Content-Type': 'application/json' },
    })
    if (res.error) throw res.error
    showCreate.value = false
    Object.assign(form, { name: '', description: '', parent_id: '' })
    refetch()
    toast.success('部门创建成功')
  } catch { toast.error('创建部门失败') } finally { creating.value = false }
}

// ─── Department Detail Sheet ───────────────────────────────────

const showDetail = ref(false)
const selectedDept = ref<Department | null>(null)

// Skill template state
const deptSkillTemplates = ref<HandlersSkillTemplateBriefDto[]>([])
const loadingSkillTemplates = ref(false)
const removingSkillId = ref<string | null>(null)
const addingSkillId = ref<string | null>(null)
const showSkillPicker = ref(false)
const loadingAllTemplates = ref(false)
const allSkillTemplates = ref<Array<{ id?: string, name?: string, description?: string }>>([])

// Directory template state
const dirPaths = ref<string[]>([])
const dirPathsOriginal = ref<string[]>([])
const loadingDirTemplates = ref(false)
const savingDirs = ref(false)
const newDirPath = ref('')

// Sync state
const syncingSkills = ref(false)
const syncingDirs = ref(false)

const dirPathsDirty = computed(() =>
  JSON.stringify(dirPaths.value) !== JSON.stringify(dirPathsOriginal.value),
)

const isValidDirPath = computed(() => {
  const p = newDirPath.value.trim()
  return p.startsWith('/data/') && !p.includes('..')
})

const availableTemplates = computed(() => {
  const existingIds = new Set(deptSkillTemplates.value.map(t => t.id))
  return allSkillTemplates.value.filter(t => !existingIds.has(t.id))
})

function openDetail(dept: Department) {
  selectedDept.value = dept
  showDetail.value = true
  fetchDeptSkillTemplates()
  fetchDeptDirTemplates()
}

async function fetchDeptSkillTemplates() {
  if (!selectedDept.value?.id || !selectedBotId.value) return
  loadingSkillTemplates.value = true
  try {
    const res = await client.get({
      url: '/bots/{bot_id}/departments/{department_id}/skill-templates',
      path: { bot_id: selectedBotId.value, department_id: selectedDept.value.id },
    })
    if (res.error) throw res.error
    deptSkillTemplates.value = res.data ?? []
  } catch {
    toast.error('加载技能模板失败')
    deptSkillTemplates.value = []
  } finally {
    loadingSkillTemplates.value = false
  }
}

async function fetchDeptDirTemplates() {
  if (!selectedDept.value?.id || !selectedBotId.value) return
  loadingDirTemplates.value = true
  try {
    const res = await client.get({
      url: '/bots/{bot_id}/departments/{department_id}/directory-templates',
      path: { bot_id: selectedBotId.value, department_id: selectedDept.value.id },
    })
    if (res.error) throw res.error
    const paths = res.data?.paths ?? []
    dirPaths.value = [...paths]
    dirPathsOriginal.value = [...paths]
  } catch {
    toast.error('加载目录模板失败')
    dirPaths.value = []
    dirPathsOriginal.value = []
  } finally {
    loadingDirTemplates.value = false
  }
}

async function fetchAllSkillTemplates() {
  loadingAllTemplates.value = true
  try {
    const res = await client.get({ url: '/skill-templates' })
    if (res.error) throw res.error
    allSkillTemplates.value = res.data ?? []
  } catch {
    toast.error('加载模板列表失败')
    allSkillTemplates.value = []
  } finally {
    loadingAllTemplates.value = false
  }
}

watch(showSkillPicker, (open) => {
  if (open) fetchAllSkillTemplates()
})

async function handleAddSkillTemplate(templateId: string) {
  if (!selectedDept.value?.id) return
  addingSkillId.value = templateId
  try {
    const res = await client.post({
      url: '/bots/{bot_id}/departments/{department_id}/skill-templates',
      path: { bot_id: selectedBotId.value, department_id: selectedDept.value.id },
      body: { template_id: templateId },
      headers: { 'Content-Type': 'application/json' },
    })
    if (res.error) throw res.error
    toast.success('技能模板已添加')
    showSkillPicker.value = false
    fetchDeptSkillTemplates()
  } catch {
    toast.error('添加技能模板失败')
  } finally {
    addingSkillId.value = null
  }
}

async function handleRemoveSkillTemplate(templateId: string) {
  if (!selectedDept.value?.id) return
  removingSkillId.value = templateId
  try {
    const res = await client.delete({
      url: '/bots/{bot_id}/departments/{department_id}/skill-templates/{template_id}',
      path: { bot_id: selectedBotId.value, department_id: selectedDept.value.id, template_id: templateId },
    })
    if (res.error) throw res.error
    toast.success('技能模板已移除')
    fetchDeptSkillTemplates()
  } catch {
    toast.error('移除技能模板失败')
  } finally {
    removingSkillId.value = null
  }
}

// ─── Directory template management ─────────────────────────────

function addDirPath() {
  const p = newDirPath.value.trim()
  if (!isValidDirPath.value) return
  if (dirPaths.value.includes(p)) {
    toast.error('路径已存在')
    return
  }
  dirPaths.value.push(p)
  newDirPath.value = ''
}

function removeDirPath(idx: number) {
  dirPaths.value.splice(idx, 1)
}

async function handleSaveDirTemplates() {
  if (!selectedDept.value?.id) return
  savingDirs.value = true
  try {
    const res = await client.put({
      url: '/bots/{bot_id}/departments/{department_id}/directory-templates',
      path: { bot_id: selectedBotId.value, department_id: selectedDept.value.id },
      body: { paths: dirPaths.value },
      headers: { 'Content-Type': 'application/json' },
    })
    if (res.error) throw res.error
    dirPathsOriginal.value = [...dirPaths.value]
    toast.success('目录模板已保存')
  } catch {
    toast.error('保存目录模板失败')
  } finally {
    savingDirs.value = false
  }
}

// ─── Sync operations ───────────────────────────────────────────

async function handleSyncSkills() {
  if (!selectedDept.value?.id) return
  syncingSkills.value = true
  try {
    const res = await client.post({
      url: '/bots/{bot_id}/departments/{department_id}/sync-skills',
      path: { bot_id: selectedBotId.value, department_id: selectedDept.value.id },
    })
    if (res.error) throw res.error
    const d = res.data
    toast.success(`同步完成: ${d?.installed ?? 0} 已安装, ${d?.skipped ?? 0} 已跳过, 共 ${d?.total_bots ?? 0} 个Bot`)
    if (d?.errors && d.errors.length > 0) {
      toast.warning(`${d.errors.length} 个Bot同步出错`)
    }
  } catch {
    toast.error('同步技能失败')
  } finally {
    syncingSkills.value = false
  }
}

async function handleSyncDirectories() {
  if (!selectedDept.value?.id) return
  syncingDirs.value = true
  try {
    const res = await client.post({
      url: '/bots/{bot_id}/departments/{department_id}/sync-directories',
      path: { bot_id: selectedBotId.value, department_id: selectedDept.value.id },
    })
    if (res.error) throw res.error
    const d = res.data
    toast.success(`同步完成: ${d?.installed ?? 0} 已安装, ${d?.skipped ?? 0} 已跳过, 共 ${d?.total_bots ?? 0} 个Bot`)
    if (d?.errors && d.errors.length > 0) {
      toast.warning(`${d.errors.length} 个Bot同步出错`)
    }
  } catch {
    toast.error('同步目录失败')
  } finally {
    syncingDirs.value = false
  }
}
</script>
