<template>
  <div class="rounded-xl border border-border bg-muted/40 p-4 text-sm">
    <div class="mb-2 flex items-center gap-2 font-medium">
      <FontAwesomeIcon :icon="['fas', 'calendar-alt']" class="size-4 text-primary" />
      <span>{{ $t('chat.sessionType.schedule') }}</span>
    </div>
    <div class="space-y-1 text-muted-foreground">
      <p v-if="meta['schedule-name']"><span class="font-medium text-foreground">Name:</span> {{ meta['schedule-name'] }}</p>
      <p v-if="meta['schedule-description']"><span class="font-medium text-foreground">Description:</span> {{ meta['schedule-description'] }}</p>
      <p v-if="meta['cron-pattern']"><span class="font-medium text-foreground">Cron:</span> {{ meta['cron-pattern'] }}</p>
      <p v-if="meta['max-calls']"><span class="font-medium text-foreground">Max calls:</span> {{ meta['max-calls'] }}</p>
      <p v-if="body" class="pt-2 text-foreground whitespace-pre-wrap">{{ body }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { extractTriggerBody, extractTriggerFrontmatter } from './trigger-frontmatter'

const props = defineProps<{
  text: string
}>()

const meta = computed(() => extractTriggerFrontmatter(props.text))
const body = computed(() => extractTriggerBody(props.text))
</script>
