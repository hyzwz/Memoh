<template>
  <div class="p-6 space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold">
          {{ $t('skillTemplates.title') }}
        </h2>
        <p class="text-sm text-muted-foreground mt-1">
          {{ $t('skillTemplates.subtitle') }}
        </p>
      </div>
      <Button @click="handleCreate">
        <FontAwesomeIcon
          :icon="['fas', 'plus']"
          class="mr-2"
        />
        {{ $t('skillTemplates.createTemplate') }}
      </Button>
    </div>

    <!-- Loading -->
    <div
      v-if="isLoading"
      class="flex items-center justify-center py-12 text-sm text-muted-foreground"
    >
      <Spinner class="mr-2" />
      {{ $t('common.loading') }}
    </div>

    <!-- Empty -->
    <div
      v-else-if="!templates.length"
      class="flex flex-col items-center justify-center py-16 text-center"
    >
      <div class="rounded-full bg-muted p-3 mb-4">
        <FontAwesomeIcon
          :icon="['fas', 'wand-magic-sparkles']"
          class="size-6 text-muted-foreground"
        />
      </div>
      <h3 class="text-lg font-medium">
        {{ $t('skillTemplates.emptyTitle') }}
      </h3>
      <p class="text-sm text-muted-foreground mt-1">
        {{ $t('skillTemplates.emptyDescription') }}
      </p>
    </div>

    <!-- Templates Table -->
    <div
      v-else
      class="border rounded-lg"
    >
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b bg-muted/50">
            <th class="text-left p-3 font-medium">
              {{ $t('skillTemplates.name') }}
            </th>
            <th class="text-left p-3 font-medium">
              {{ $t('skillTemplates.category') }}
            </th>
            <th class="text-left p-3 font-medium">
              {{ $t('skillTemplates.version') }}
            </th>
            <th class="text-left p-3 font-medium">
              {{ $t('skillTemplates.status') }}
            </th>
            <th class="text-left p-3 font-medium">
              {{ $t('skillTemplates.installs') }}
            </th>
            <th class="text-right p-3 font-medium">
              {{ $t('skillTemplates.actions') }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="tpl in templates"
            :key="tpl.id"
            class="border-b last:border-0 hover:bg-muted/30"
          >
            <td class="p-3">
              <div class="flex items-center gap-2">
                <span v-if="tpl.icon">{{ tpl.icon }}</span>
                <div>
                  <div class="font-medium">
                    {{ tpl.name }}
                  </div>
                  <div class="text-xs text-muted-foreground truncate max-w-[300px]">
                    {{ tpl.description }}
                  </div>
                </div>
              </div>
            </td>
            <td class="p-3">
              <Badge variant="secondary">
                {{ tpl.category }}
              </Badge>
            </td>
            <td class="p-3">
              v{{ tpl.version }}
            </td>
            <td class="p-3">
              <Badge :variant="tpl.is_published ? 'default' : 'outline'">
                {{ tpl.is_published ? $t('skillTemplates.published') : $t('skillTemplates.draft') }}
              </Badge>
            </td>
            <td class="p-3">
              {{ tpl.install_count || 0 }}
            </td>
            <td class="p-3 text-right">
              <div class="flex items-center justify-end gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  class="size-8 p-0"
                  :title="$t('common.edit')"
                  @click="handleEdit(tpl)"
                >
                  <FontAwesomeIcon
                    :icon="['fas', 'pen-to-square']"
                    class="size-3.5"
                  />
                </Button>
                <ConfirmPopover
                  :message="$t('skillTemplates.deleteConfirm')"
                  :loading="isDeleting && deletingId === tpl.id"
                  @confirm="handleDelete(tpl.id!)"
                >
                  <template #trigger>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="size-8 p-0 text-destructive hover:text-destructive"
                      :disabled="isDeleting"
                      :title="$t('common.delete')"
                    >
                      <FontAwesomeIcon
                        :icon="['far', 'trash-can']"
                        class="size-3.5"
                      />
                    </Button>
                  </template>
                </ConfirmPopover>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Editor Dialog -->
    <TemplateEditorDialog
      v-model:open="isDialogOpen"
      :template="editingTemplate"
      @saved="fetchTemplates"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Button, Badge, Spinner } from '@memoh/ui'
import ConfirmPopover from '@/components/confirm-popover/index.vue'
import TemplateEditorDialog from './components/template-editor-dialog.vue'
import {
  getSkillTemplates,
  deleteSkillTemplatesById,
  type HandlersSkillTemplateResponse,
} from '@memoh/sdk'
import { resolveApiErrorMessage } from '@/utils/api-error'

const { t } = useI18n()

const isLoading = ref(false)
const isDeleting = ref(false)
const deletingId = ref('')
const templates = ref<HandlersSkillTemplateResponse[]>([])
const isDialogOpen = ref(false)
const editingTemplate = ref<HandlersSkillTemplateResponse | null>(null)

async function fetchTemplates() {
  isLoading.value = true
  try {
    const { data } = await getSkillTemplates({
      throwOnError: true,
    })
    templates.value = data || []
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('skillTemplates.loadFailed')))
  } finally {
    isLoading.value = false
  }
}

function handleCreate() {
  editingTemplate.value = null
  isDialogOpen.value = true
}

function handleEdit(tpl: HandlersSkillTemplateResponse) {
  editingTemplate.value = tpl
  isDialogOpen.value = true
}

async function handleDelete(id: string) {
  isDeleting.value = true
  deletingId.value = id
  try {
    await deleteSkillTemplatesById({
      path: { id },
      throwOnError: true,
    })
    toast.success(t('skillTemplates.deleteSuccess'))
    await fetchTemplates()
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('skillTemplates.deleteFailed')))
  } finally {
    isDeleting.value = false
    deletingId.value = ''
  }
}

onMounted(() => {
  fetchTemplates()
})
</script>
