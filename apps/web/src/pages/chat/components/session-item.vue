<template>
  <button
    type="button"
    class="w-full rounded-lg border px-3 py-2 text-left transition-colors"
    :class="active
      ? 'border-border bg-accent/70'
      : 'border-transparent hover:bg-accent/40'"
    @click="$emit('select')"
  >
    <div class="flex items-start gap-3">
      <div class="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
        <FontAwesomeIcon
          :icon="icon"
          class="size-4"
        />
      </div>
      <div class="min-w-0 flex-1">
        <div class="truncate text-sm font-medium text-foreground">
          {{ title }}
        </div>
        <div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
          <span>{{ typeLabel }}</span>
          <span v-if="session.updated_at">· {{ relativeUpdatedAt }}</span>
        </div>
      </div>
    </div>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatRelativeTime } from '@/utils/date-time'
import type { ChatSummary } from '@/composables/api/useChat'
import { getSessionTypeMeta } from './session-meta'

const props = defineProps<{
  session: ChatSummary
  active?: boolean
}>()

defineEmits<{
  select: []
}>()

const { t } = useI18n()

const meta = computed(() => getSessionTypeMeta(props.session.type))
const title = computed(() => props.session.title || props.session.id)
const typeLabel = computed(() => t(meta.value.labelKey))
const relativeUpdatedAt = computed(() =>
  props.session.updated_at ? formatRelativeTime(props.session.updated_at) : '',
)

const icon = computed(() => {
  switch (meta.value.icon) {
    case 'heartbeat':
      return ['fas', 'heartbeat'] as const
    case 'calendar':
      return ['fas', 'calendar-alt'] as const
    case 'bot':
      return ['fas', 'robot'] as const
    default:
      return ['far', 'comment-dots'] as const
  }
})
</script>
