<template>
  <div class="space-y-4">
    <!-- Category filter + search -->
    <div class="flex items-center gap-3 flex-wrap">
      <div class="flex items-center gap-1">
        <Button
          v-for="cat in allCategories"
          :key="cat"
          size="sm"
          :variant="selectedCategory === cat ? 'default' : 'outline'"
          @click="selectedCategory = cat"
        >
          {{ cat === '' ? $t('skillTemplates.allCategories') : cat }}
        </Button>
      </div>
      <Input
        v-model="searchQuery"
        :placeholder="$t('skillTemplates.searchPlaceholder')"
        class="max-w-[200px] h-8"
      />
    </div>

    <!-- Loading -->
    <div
      v-if="isLoading"
      class="flex items-center justify-center py-8 text-sm text-muted-foreground"
    >
      <Spinner class="mr-2" />
      {{ $t('common.loading') }}
    </div>

    <!-- Empty -->
    <div
      v-else-if="!filteredTemplates.length"
      class="flex flex-col items-center justify-center py-12 text-center"
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
    </div>

    <!-- Template grid -->
    <div
      v-else
      class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
    >
      <SkillTemplateCard
        v-for="tpl in filteredTemplates"
        :key="tpl.id"
        :template="tpl"
        :loading="actionLoading && actionId === tpl.id"
        :action="currentAction"
        @install="handleInstall"
        @uninstall="handleUninstall"
        @update="handleUpdate"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Button, Input, Spinner } from '@memoh/ui'
import SkillTemplateCard from './skill-template-card.vue'
import {
  getBotsByBotIdSkillStore,
  postBotsByBotIdSkillStoreInstall,
  postBotsByBotIdSkillStoreUninstall,
  postBotsByBotIdSkillStoreUpdate,
  type HandlersStoreTemplateResponse,
} from '@memoh/sdk'
import { resolveApiErrorMessage } from '@/utils/api-error'

const props = defineProps<{
  botId: string
}>()

const emit = defineEmits<{
  skillsChanged: []
}>()

const { t } = useI18n()

const isLoading = ref(false)
const actionLoading = ref(false)
const actionId = ref('')
const currentAction = ref<'install' | 'uninstall' | 'update'>('install')
const templates = ref<HandlersStoreTemplateResponse[]>([])
const selectedCategory = ref('')
const searchQuery = ref('')

const allCategories = ['', 'general', 'coding', 'writing', 'data', 'web', 'automation', 'communication', 'research']

const filteredTemplates = computed(() => {
  let result = templates.value
  if (selectedCategory.value) {
    result = result.filter(t => t.category === selectedCategory.value)
  }
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(t =>
      (t.name || '').toLowerCase().includes(q)
      || (t.description || '').toLowerCase().includes(q),
    )
  }
  return result
})

async function fetchStore() {
  isLoading.value = true
  try {
    const { data } = await getBotsByBotIdSkillStore({
      path: { bot_id: props.botId },
      throwOnError: true,
    })
    templates.value = data || []
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('skillTemplates.loadFailed')))
  } finally {
    isLoading.value = false
  }
}

async function handleInstall(templateId: string) {
  actionLoading.value = true
  actionId.value = templateId
  currentAction.value = 'install'
  try {
    await postBotsByBotIdSkillStoreInstall({
      path: { bot_id: props.botId },
      body: { template_id: templateId },
      throwOnError: true,
    })
    toast.success(t('skillTemplates.installSuccess'))
    await fetchStore()
    emit('skillsChanged')
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('skillTemplates.installFailed')))
  } finally {
    actionLoading.value = false
    actionId.value = ''
  }
}

async function handleUninstall(templateId: string) {
  actionLoading.value = true
  actionId.value = templateId
  currentAction.value = 'uninstall'
  try {
    await postBotsByBotIdSkillStoreUninstall({
      path: { bot_id: props.botId },
      body: { template_id: templateId },
      throwOnError: true,
    })
    toast.success(t('skillTemplates.uninstallSuccess'))
    await fetchStore()
    emit('skillsChanged')
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('skillTemplates.uninstallFailed')))
  } finally {
    actionLoading.value = false
    actionId.value = ''
  }
}

async function handleUpdate(templateId: string) {
  actionLoading.value = true
  actionId.value = templateId
  currentAction.value = 'update'
  try {
    await postBotsByBotIdSkillStoreUpdate({
      path: { bot_id: props.botId },
      body: { template_id: templateId },
      throwOnError: true,
    })
    toast.success(t('skillTemplates.updateSuccess'))
    await fetchStore()
    emit('skillsChanged')
  } catch (error) {
    toast.error(resolveApiErrorMessage(error, t('skillTemplates.updateFailed')))
  } finally {
    actionLoading.value = false
    actionId.value = ''
  }
}

onMounted(() => {
  fetchStore()
})

defineExpose({ fetchStore })
</script>
