<template>
  <div class="glass-panel p-5">
    <h3 class="text-sm font-medium text-text-secondary mb-4">
      <FontAwesomeIcon
        :icon="['fas', 'wand-magic-sparkles']"
        class="mr-1.5 text-violet-400"
      />
      {{ $t('cockpit.innovationHighlights') }}
    </h3>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <div
        v-for="item in data"
        :key="item.id"
        class="glass-panel-hover p-3 glow-border-purple"
      >
        <div class="flex items-start gap-2">
          <span class="inline-flex items-center rounded-md bg-violet-500/15 px-2 py-0.5 text-[10px] font-medium text-violet-400 shrink-0">
            <FontAwesomeIcon
              :icon="['fas', 'sparkles']"
              class="mr-1 size-2.5"
            />
            {{ $t('cockpit.innovation') }}
          </span>
          <span class="text-xs text-text-muted">{{ item.userName }}</span>
        </div>
        <p class="text-sm text-text-primary mt-2 line-clamp-2">
          {{ item.description }}
        </p>
        <div class="flex items-center gap-3 mt-2 text-xs text-text-muted">
          <span>{{ item.category }}</span>
          <span>{{ item.savedHours.toFixed(1) }}h {{ $t('cockpit.saved') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  data: Array<{
    id: string
    userName: string
    description: string
    category: string
    savedHours: number
  }>
}>()
</script>
