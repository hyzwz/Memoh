<template>
  <Dialog
    v-model:open="isOpen"
  >
    <DialogContent class="sm:max-w-3xl max-h-[90vh] flex flex-col">
      <DialogHeader>
        <DialogTitle>{{ isEditing ? $t('skillTemplates.editTemplate') : $t('skillTemplates.createTemplate') }}</DialogTitle>
      </DialogHeader>

      <div class="flex-1 overflow-y-auto space-y-4 py-4">
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label>{{ $t('skillTemplates.name') }}</Label>
            <Input
              v-model="form.name"
              :placeholder="$t('skillTemplates.namePlaceholder')"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('skillTemplates.slug') }}</Label>
            <Input
              v-model="form.slug"
              :placeholder="$t('skillTemplates.slugPlaceholder')"
            />
          </div>
        </div>

        <div class="space-y-2">
          <Label>{{ $t('skillTemplates.description') }}</Label>
          <Input
            v-model="form.description"
            :placeholder="$t('skillTemplates.descriptionPlaceholder')"
          />
        </div>

        <div class="grid grid-cols-3 gap-4">
          <div class="space-y-2">
            <Label>{{ $t('skillTemplates.category') }}</Label>
            <Select v-model="form.category">
              <SelectTrigger>
                <SelectValue :placeholder="$t('skillTemplates.selectCategory')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem
                  v-for="cat in categories"
                  :key="cat"
                  :value="cat"
                >
                  {{ cat }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label>{{ $t('skillTemplates.author') }}</Label>
            <Input
              v-model="form.author"
              :placeholder="$t('skillTemplates.authorPlaceholder')"
            />
          </div>
          <div class="space-y-2">
            <Label>{{ $t('skillTemplates.icon') }}</Label>
            <Input
              v-model="form.icon"
              :placeholder="$t('skillTemplates.iconPlaceholder')"
            />
          </div>
        </div>

        <div class="flex items-center gap-2">
          <Switch
            :checked="form.is_published"
            @update:checked="form.is_published = $event"
          />
          <Label>{{ $t('skillTemplates.published') }}</Label>
        </div>

        <div class="space-y-2">
          <Label>{{ $t('skillTemplates.content') }}</Label>
          <div class="h-[300px] border rounded-md overflow-hidden">
            <MonacoEditor
              v-model="form.content"
              language="markdown"
              :readonly="isSaving"
            />
          </div>
        </div>
      </div>

      <DialogFooter>
        <DialogClose as-child>
          <Button
            variant="outline"
            :disabled="isSaving"
          >
            {{ $t('common.cancel') }}
          </Button>
        </DialogClose>
        <Button
          :disabled="!canSave || isSaving"
          @click="handleSave"
        >
          <Spinner
            v-if="isSaving"
            class="mr-2 size-4"
          />
          {{ $t('common.save') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import {
  Button, Input, Label,
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogClose,
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
  Switch, Spinner,
} from '@memoh/ui'
import MonacoEditor from '@/components/monaco-editor/index.vue'
import {
  postSkillTemplates,
  putSkillTemplatesById,
  type HandlersSkillTemplateResponse,
} from '@memoh/sdk'
import { resolveApiErrorMessage } from '@/utils/api-error'

const props = defineProps<{
  template?: HandlersSkillTemplateResponse | null
}>()

const emit = defineEmits<{
  saved: []
}>()

const isOpen = defineModel<boolean>('open', { default: false })

const { t } = useI18n()
const isSaving = ref(false)

const categories = ['general', 'coding', 'writing', 'data', 'web', 'automation', 'communication', 'research']

const defaultForm = () => ({
  name: '',
  slug: '',
  description: '',
  category: 'general',
  content: '',
  author: '',
  icon: '',
  tags: [] as string[],
  is_published: false,
})

const form = ref(defaultForm())

const isEditing = computed(() => !!props.template?.id)

const canSave = computed(() => {
  return form.value.name.trim().length > 0 && form.value.content.trim().length > 0
})

watch(isOpen, (open) => {
  if (open) {
    if (props.template?.id) {
      form.value = {
        name: props.template.name || '',
        slug: props.template.slug || '',
        description: props.template.description || '',
        category: props.template.category || 'general',
        content: props.template.content || '',
        author: props.template.author || '',
        icon: props.template.icon || '',
        tags: props.template.tags || [],
        is_published: props.template.is_published || false,
      }
    } else {
      form.value = defaultForm()
    }
  }
})

async function handleSave() {
  if (!canSave.value) return
  isSaving.value = true
  try {
    if (isEditing.value) {
      await putSkillTemplatesById({
        path: { id: props.template!.id! },
        body: form.value,
        throwOnError: true,
      })
    } else {
      await postSkillTemplates({
        body: form.value,
        throwOnError: true,
      })
    }
    toast.success(t('skillTemplates.saveSuccess'))
    isOpen.value = false
    emit('saved')
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('skillTemplates.saveFailed')))
  } finally {
    isSaving.value = false
  }
}
</script>
