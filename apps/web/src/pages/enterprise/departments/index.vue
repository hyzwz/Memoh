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
      <Button
        size="sm"
        @click="showCreate = true"
      >
        + 新建部门
      </Button>
    </div>

    <!-- Loading -->
    <template v-if="isLoading">
      <div class="flex justify-center py-20">
        <Spinner class="size-8" />
      </div>
    </template>

    <!-- Department org chart -->
    <template v-else-if="rootDepartments.length > 0">
      <div class="overflow-x-auto pb-8 -mx-6">
        <div class="inline-flex gap-12 min-w-full justify-center px-6 pt-2">
          <OrgChartNode
            v-for="rootDept in rootDepartments"
            :key="rootDept.id"
            :dept="rootDept"
            :children-map="childrenMap"
            @select="openDetail"
          />
        </div>
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

    <!-- Edit dialog -->
    <Dialog v-model:open="showEdit">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>编辑部门</DialogTitle>
        </DialogHeader>
        <div class="space-y-4 mt-4">
          <div class="space-y-1.5">
            <Label>部门名称</Label>
            <Input
              v-model="editForm.name"
              placeholder="例: 放射科"
            />
          </div>
          <div class="space-y-1.5">
            <Label>描述 <span class="text-muted-foreground text-xs">(可选)</span></Label>
            <Input
              v-model="editForm.description"
              placeholder="部门职责描述"
            />
          </div>
          <div class="space-y-1.5">
            <Label>上级部门 <span class="text-muted-foreground text-xs">(可选)</span></Label>
            <Select
              v-if="departments && departments.length > 0"
              v-model="editForm.parent_id"
            >
              <SelectTrigger>
                <SelectValue placeholder="无 (顶级部门)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">
                  无 (顶级部门)
                </SelectItem>
                <SelectItem
                  v-for="d in departments!.filter(x => x.id !== editForm.id)"
                  :key="d.id"
                  :value="d.id!"
                >
                  {{ d.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <Button
            variant="outline"
            @click="showEdit = false"
          >
            取消
          </Button>
          <Button
            :disabled="!editForm.name.trim() || editing"
            @click="handleEdit"
          >
            <Spinner
              v-if="editing"
              class="mr-2 size-4"
            />
            保存
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
          <div class="flex items-center justify-between">
            <SheetTitle>{{ selectedDept?.name }}</SheetTitle>
            <div class="flex items-center gap-1">
              <Button
                size="sm"
                variant="ghost"
                @click="openEditFromDetail"
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
                  d="m16.862 4.487 1.687-1.688a1.875 1.875 0 1 1 2.652 2.652L10.582 16.07a4.5 4.5 0 0 1-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 0 1 1.13-1.897l8.932-8.931Zm0 0L19.5 7.125M18 14v4.75A2.25 2.25 0 0 1 15.75 21H5.25A2.25 2.25 0 0 1 3 18.75V8.25A2.25 2.25 0 0 1 5.25 6H10"
                /></svg>
              </Button>
              <Button
                size="sm"
                variant="ghost"
                class="text-destructive hover:text-destructive"
                :disabled="deleting"
                @click="handleDelete"
              >
                <Spinner
                  v-if="deleting"
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
                  v-for="(dirPath, idx) in dirPaths"
                  :key="idx"
                  class="flex items-center justify-between p-2 rounded-lg border bg-muted/30"
                >
                  <code class="text-sm font-mono truncate">{{ dirPath }}</code>
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

          <Separator />

          <!-- Associated bots section -->
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold">
                关联 Bot
              </h3>
              <Button
                size="sm"
                variant="outline"
                @click="showBotPicker = true"
              >
                + 添加
              </Button>
            </div>

            <div
              v-if="loadingDeptBots"
              class="flex justify-center py-4"
            >
              <Spinner class="size-5" />
            </div>
            <div
              v-else-if="deptBots.length === 0"
              class="text-sm text-muted-foreground py-4 text-center"
            >
              暂无关联的 Bot
            </div>
            <div
              v-else
              class="space-y-2"
            >
              <div
                v-for="bot in deptBots"
                :key="bot.id"
                class="flex items-center justify-between p-3 rounded-lg border bg-muted/30"
              >
                <div class="text-sm font-medium truncate">
                  {{ bot.display_name || bot.id }}
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  class="text-destructive hover:text-destructive shrink-0 ml-2"
                  :disabled="removingBotId === bot.id"
                  @click.stop="handleRemoveBot(bot.id!)"
                >
                  <Spinner
                    v-if="removingBotId === bot.id"
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

    <!-- Bot picker dialog -->
    <Dialog v-model:open="showBotPicker">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>选择 Bot</DialogTitle>
        </DialogHeader>
        <div class="mt-4 space-y-3 max-h-80 overflow-y-auto">
          <div
            v-if="availableBots.length === 0"
            class="text-sm text-muted-foreground text-center py-8"
          >
            没有可添加的 Bot
          </div>
          <div
            v-for="bot in availableBots"
            v-else
            :key="bot.id"
            class="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/50 cursor-pointer transition-colors"
            @click="handleAddBot(bot.id!)"
          >
            <div class="text-sm font-medium truncate">
              {{ bot.display_name || bot.id }}
            </div>
            <Button
              size="sm"
              variant="ghost"
              :disabled="addingBotId === bot.id"
            >
              <Spinner
                v-if="addingBotId === bot.id"
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
import { ref, computed, reactive, watch, defineComponent, h } from 'vue'
import { useQuery } from '@pinia/colada'
import { toast } from 'vue-sonner'
import {
  Button,
  Dialog, DialogContent, DialogHeader, DialogTitle,
  Input, Label, Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
  Separator, Sheet, SheetContent, SheetHeader, SheetTitle, Spinner,
} from '@memoh/ui'
import { getBotsQuery } from '@memoh/sdk/colada'
import { client } from '@memoh/sdk/client'
import EmptyState from '../_components/EmptyState.vue'

import type { HandlersSkillTemplateBriefDto } from '@memoh/sdk'

interface Department {
  id?: string
  name?: string
  description?: string
  parent_id?: string
  created_at?: string
}

interface BotBrief {
  id?: string
  display_name?: string
}

// ─── Org chart node component ───────────────────────────────────

const OrgChartNode = defineComponent({
  name: 'OrgChartNode',
  props: {
    dept: { type: Object as () => Department, required: true },
    childrenMap: { type: Object as () => Map<string, Department[]>, required: true },
  },
  emits: ['select'],
  setup(props, { emit }) {
    const expanded = ref(true)
    const children = computed(() => props.childrenMap.get(props.dept.id!) ?? [])
    const hasChildren = computed(() => children.value.length > 0)

    const buildingPath = 'M3.75 21h16.5M4.5 3h15M5.25 3v18m13.5-18v18M9 6.75h1.5m-1.5 3h1.5m-1.5 3h1.5m3-6H15m-1.5 3H15m-1.5 3H15M9 21v-3.375c0-.621.504-1.125 1.125-1.125h3.75c.621 0 1.125.504 1.125 1.125V21'

    return () => h('div', { class: 'flex flex-col items-center' }, [
      // ── Department Card ──
      h('div', {
        class: 'org-card relative flex flex-col items-center px-5 py-4 rounded-xl border-2 border-indigo-200 dark:border-indigo-700/60 bg-indigo-50/30 dark:bg-indigo-950/20 hover:border-indigo-400 dark:hover:border-indigo-500 hover:shadow-lg cursor-pointer transition-all duration-200',
        style: { minWidth: '150px', maxWidth: '200px' },
        onClick: () => emit('select', props.dept),
      }, [
        // Avatar circle with building icon
        h('div', { class: 'size-14 rounded-full bg-indigo-100 dark:bg-indigo-900/50 flex items-center justify-center mb-2 ring-2 ring-indigo-200/50 dark:ring-indigo-700/30' }, [
          h('svg', {
            xmlns: 'http://www.w3.org/2000/svg',
            class: 'size-7 text-indigo-500 dark:text-indigo-400',
            fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5',
          }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: buildingPath })]),
        ]),
        // Department name
        h('div', { class: 'text-sm font-semibold text-center w-full leading-snug' }, props.dept.name),
        // Description
        props.dept.description
          ? h('div', { class: 'text-xs text-muted-foreground text-center w-full mt-0.5 truncate' }, props.dept.description)
          : null,
      ]),

      // ── Connector line + Children ──
      hasChildren.value
        ? h('div', { class: 'flex flex-col items-center' }, [
            // Vertical line going down from card
            h('div', { class: 'w-0.5 bg-border', style: { height: '20px' } }),

            // Expand/collapse toggle badge
            h('button', {
              class: 'relative z-10 flex items-center justify-center rounded-full border-2 border-border bg-background hover:bg-indigo-50 dark:hover:bg-indigo-950 hover:border-indigo-400 transition-all duration-200',
              style: { width: '26px', height: '26px' },
              onClick: (e: Event) => { e.stopPropagation(); expanded.value = !expanded.value },
            }, [
              expanded.value
                ? h('svg', {
                    xmlns: 'http://www.w3.org/2000/svg', class: 'size-3.5 text-muted-foreground',
                    fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '2.5',
                  }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M4.5 15.75l7.5-7.5 7.5 7.5' })])
                : h('span', { class: 'text-xs font-semibold text-indigo-500' }, `${children.value.length}`),
            ]),

            // Expanded children
            expanded.value
              ? h('div', { class: 'flex flex-col items-center' }, [
                  // Vertical line from toggle down to horizontal bar
                  h('div', { class: 'w-0.5 bg-border', style: { height: '20px' } }),

                  // Children row
                  h('div', { class: 'flex items-start' },
                    children.value.map((child, idx) => {
                      const isFirst = idx === 0
                      const isLast = idx === children.value.length - 1
                      const isOnly = children.value.length === 1

                      return h('div', {
                        key: child.id,
                        class: 'flex flex-col items-center',
                        style: { padding: '0 8px' },
                      }, [
                        // Connector area: horizontal bar segment + vertical drop
                        h('div', {
                          class: 'relative w-full',
                          style: { height: '20px' },
                        }, [
                          // Horizontal line segment
                          !isOnly
                            ? h('div', {
                                class: 'absolute top-0 bg-border',
                                style: {
                                  height: '2px',
                                  ...(isFirst
                                    ? { left: '50%', right: '0' }
                                    : isLast
                                      ? { left: '0', right: '50%' }
                                      : { left: '0', right: '0' }),
                                },
                              })
                            : null,
                          // Vertical line down to child card
                          h('div', {
                            class: 'absolute bg-border',
                            style: { left: 'calc(50% - 1px)', top: '0', width: '2px', height: '100%' },
                          }),
                        ]),

                        // Recursive child node
                        h(OrgChartNode, {
                          dept: child,
                          childrenMap: props.childrenMap,
                          onSelect: (d: Department) => emit('select', d),
                        }),
                      ])
                    }),
                  ),
                ])
              : null,
          ])
        : null,
    ])
  },
})

// ─── State ──────────────────────────────────────────────────────

const showCreate = ref(false)
const creating = ref(false)
const form = reactive({ name: '', description: '', parent_id: '' })

const { data: departments, status, refetch } = useQuery({
  key: () => ['departments'],
  query: async () => {
    const res = await client.get({ url: '/departments' })
    if (res.error) throw new Error('Failed')
    return (res.data ?? []) as Department[]
  },
})

const isLoading = computed(() => status.value === 'loading')

// Build tree structure
const childrenMap = computed(() => {
  const map = new Map<string, Department[]>()
  for (const dept of (departments.value ?? [])) {
    const parentId = dept.parent_id || '__root__'
    if (!map.has(parentId)) map.set(parentId, [])
    map.get(parentId)!.push(dept)
  }
  return map
})

const rootDepartments = computed(() => {
  const all = departments.value ?? []
  const idSet = new Set(all.map(d => d.id))
  // A department is a root if it has no parent_id, or its parent doesn't exist in the list.
  // Also detect cycles: walk up from each node; if we revisit a node, treat it as root.
  const parentMap = new Map<string, string>()
  for (const d of all) {
    if (d.id && d.parent_id && idSet.has(d.parent_id)) {
      parentMap.set(d.id, d.parent_id)
    }
  }
  const rootIds = new Set<string>()
  for (const d of all) {
    if (!d.id) continue
    if (!parentMap.has(d.id)) {
      rootIds.add(d.id)
      continue
    }
    // Walk ancestors to check for cycle
    const visited = new Set<string>()
    let cur: string | undefined = d.id
    while (cur && parentMap.has(cur)) {
      if (visited.has(cur)) {
        // Cycle detected — mark this node as root to prevent tree collapse
        rootIds.add(d.id)
        break
      }
      visited.add(cur)
      cur = parentMap.get(cur)
    }
  }
  return all.filter(d => d.id && rootIds.has(d.id))
})

async function handleCreate() {
  creating.value = true
  try {
    const res = await client.post({
      url: '/departments',
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

// ─── Edit ───────────────────────────────────────────────────────

const showEdit = ref(false)
const editing = ref(false)
const editForm = reactive({ id: '', name: '', description: '', parent_id: '' })

function openEditFromDetail() {
  if (!selectedDept.value) return
  editForm.id = selectedDept.value.id ?? ''
  editForm.name = selectedDept.value.name ?? ''
  editForm.description = selectedDept.value.description ?? ''
  editForm.parent_id = selectedDept.value.parent_id ?? ''
  showEdit.value = true
}

async function handleEdit() {
  if (!editForm.id) return
  editing.value = true
  try {
    const res = await client.put({
      url: '/departments/{department_id}',
      path: { department_id: editForm.id },
      body: {
        name: editForm.name,
        description: editForm.description,
        parent_id: (editForm.parent_id && editForm.parent_id !== '__none__') ? editForm.parent_id : undefined,
      },
      headers: { 'Content-Type': 'application/json' },
    })
    if (res.error) throw res.error
    showEdit.value = false
    if (selectedDept.value) {
      selectedDept.value = { ...selectedDept.value, name: editForm.name, description: editForm.description, parent_id: editForm.parent_id }
    }
    refetch()
    toast.success('部门已更新')
  } catch { toast.error('更新部门失败') } finally { editing.value = false }
}

// ─── Delete ─────────────────────────────────────────────────────

const deleting = ref(false)

async function handleDelete() {
  if (!selectedDept.value?.id) return
  deleting.value = true
  try {
    const res = await client.delete({
      url: '/departments/{department_id}',
      path: { department_id: selectedDept.value.id },
    })
    if (res.error) throw res.error
    showDetail.value = false
    refetch()
    toast.success('部门已删除')
  } catch { toast.error('删除部门失败') } finally { deleting.value = false }
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

// Associated bots state
const deptBots = ref<BotBrief[]>([])
const loadingDeptBots = ref(false)
const removingBotId = ref<string | null>(null)
const addingBotId = ref<string | null>(null)
const showBotPicker = ref(false)
// Bot list loaded via getBotsQuery() — no separate loading state needed

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

const { data: botData } = useQuery(getBotsQuery())
const allBotList = computed(() => botData.value?.items ?? [])

const availableBots = computed(() => {
  const existingIds = new Set(deptBots.value.map(b => b.id))
  return allBotList.value.filter(b => !existingIds.has(b.id))
})

function openDetail(dept: Department) {
  selectedDept.value = dept
  showDetail.value = true
  fetchDeptSkillTemplates()
  fetchDeptDirTemplates()
  fetchDeptBots()
}

async function fetchDeptSkillTemplates() {
  if (!selectedDept.value?.id) return
  loadingSkillTemplates.value = true
  try {
    const res = await client.get({
      url: '/departments/{department_id}/skill-templates',
      path: { department_id: selectedDept.value.id },
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
  if (!selectedDept.value?.id) return
  loadingDirTemplates.value = true
  try {
    const res = await client.get({
      url: '/departments/{department_id}/directory-templates',
      path: { department_id: selectedDept.value.id },
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

async function fetchDeptBots() {
  if (!selectedDept.value?.id) return
  loadingDeptBots.value = true
  try {
    const res = await client.get({
      url: '/departments/{department_id}/bots',
      path: { department_id: selectedDept.value.id },
    })
    if (res.error) throw res.error
    deptBots.value = (res.data ?? []) as BotBrief[]
  } catch {
    toast.error('加载关联Bot失败')
    deptBots.value = []
  } finally {
    loadingDeptBots.value = false
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

// Bot list is fetched via getBotsQuery() reactively, no extra loading needed

async function handleAddSkillTemplate(templateId: string) {
  if (!selectedDept.value?.id) return
  addingSkillId.value = templateId
  try {
    const res = await client.post({
      url: '/departments/{department_id}/skill-templates',
      path: { department_id: selectedDept.value.id },
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
      url: '/departments/{department_id}/skill-templates/{template_id}',
      path: { department_id: selectedDept.value.id, template_id: templateId },
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

// ─── Bot association management ─────────────────────────────────

async function handleAddBot(botId: string) {
  if (!selectedDept.value?.id) return
  addingBotId.value = botId
  try {
    const res = await client.post({
      url: '/bots/{bot_id}/departments',
      path: { bot_id: botId },
      body: { department_id: selectedDept.value.id },
      headers: { 'Content-Type': 'application/json' },
    })
    if (res.error) throw res.error
    toast.success('Bot 已关联')
    showBotPicker.value = false
    fetchDeptBots()
  } catch {
    toast.error('关联 Bot 失败')
  } finally {
    addingBotId.value = null
  }
}

async function handleRemoveBot(botId: string) {
  if (!selectedDept.value?.id) return
  removingBotId.value = botId
  try {
    const res = await client.delete({
      url: '/bots/{bot_id}/departments/{department_id}',
      path: { bot_id: botId, department_id: selectedDept.value.id },
    })
    if (res.error) throw res.error
    toast.success('Bot 已移除')
    fetchDeptBots()
  } catch {
    toast.error('移除 Bot 失败')
  } finally {
    removingBotId.value = null
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
      url: '/departments/{department_id}/directory-templates',
      path: { department_id: selectedDept.value.id },
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
      url: '/departments/{department_id}/sync-skills',
      path: { department_id: selectedDept.value.id },
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
      url: '/departments/{department_id}/sync-directories',
      path: { department_id: selectedDept.value.id },
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
