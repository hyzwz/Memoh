<script setup lang="ts">
import { computed } from 'vue'
import { GenerativeWidget } from '@memoh/ui'
import type { ToolCallBlock } from '@/store/chat-list'

const props = defineProps<{
  block: ToolCallBlock
}>()

const emit = defineEmits<{
  'send-prompt': [prompt: string]
}>()

const widgetHtml = computed(() => {
  try {
    const input = typeof props.block.input === 'string'
      ? JSON.parse(props.block.input)
      : props.block.input
    return input?.html || ''
  } catch {
    return ''
  }
})

const widgetTitle = computed(() => {
  try {
    const input = typeof props.block.input === 'string'
      ? JSON.parse(props.block.input)
      : props.block.input
    return input?.title
  } catch {
    return undefined
  }
})

const widgetHeight = computed(() => {
  try {
    const input = typeof props.block.input === 'string'
      ? JSON.parse(props.block.input)
      : props.block.input
    return input?.height
  } catch {
    return undefined
  }
})
</script>

<template>
  <GenerativeWidget
    v-if="widgetHtml"
    :html="widgetHtml"
    :title="widgetTitle"
    :height="widgetHeight"
    :streaming="!block.done"
    @send-prompt="emit('send-prompt', $event)"
  />
</template>
