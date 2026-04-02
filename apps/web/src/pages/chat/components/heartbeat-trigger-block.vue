<template>
  <div class="rounded-xl border border-border bg-muted/40 p-4 text-sm">
    <div class="mb-2 flex items-center gap-2 font-medium">
      <FontAwesomeIcon :icon="['fas', 'heartbeat']" class="size-4 text-primary" />
      <span>{{ $t('chat.sessionType.heartbeat') }}</span>
    </div>
    <div class="space-y-1 text-muted-foreground">
      <p v-if="meta.interval"><span class="font-medium text-foreground">Interval:</span> {{ meta.interval }}</p>
      <p v-if="meta.time"><span class="font-medium text-foreground">Time:</span> {{ meta.time }}</p>
      <p v-if="meta.last_heartbeat"><span class="font-medium text-foreground">Last:</span> {{ meta.last_heartbeat }}</p>
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
