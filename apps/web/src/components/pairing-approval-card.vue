<template>
  <section class="rounded-lg border bg-card p-4 space-y-3">
    <div class="space-y-1">
      <h3 class="text-sm font-medium">
        {{ title || $t('settings.approvePairingCode') }}
      </h3>
      <p class="text-xs text-muted-foreground">
        {{ description || $t('settings.approvePairingCodeHint') }}
      </p>
    </div>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <Input
        :model-value="modelValue"
        class="min-w-0 flex-1"
        :placeholder="placeholder || $t('settings.approvePairingCodePlaceholder')"
        :aria-label="title || $t('settings.approvePairingCode')"
        @update:model-value="onInput"
      />
      <Button
        class="sm:min-w-28"
        :disabled="loading || !modelValue.trim()"
        @click="emit('approve')"
      >
        <Spinner v-if="loading" class="mr-1.5" />
        {{ actionLabel || $t('settings.approvePairingCodeAction') }}
      </Button>
    </div>

    <p v-if="helperText" class="text-xs text-muted-foreground">
      {{ helperText }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { Button, Input, Spinner } from '@memoh/ui'

defineProps<{
  modelValue: string
  loading: boolean
  title?: string
  description?: string
  placeholder?: string
  actionLabel?: string
  helperText?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  approve: []
}>()

function onInput(value: string | number) {
  emit('update:modelValue', String(value).trim().toUpperCase())
}
</script>
