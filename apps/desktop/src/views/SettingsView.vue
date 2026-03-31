<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useSettingsStore } from '@/stores/settings'

const router = useRouter()
const store = useAppStore()
const settingsStore = useSettingsStore()
const i18n = computed(() => settingsStore.t)
const newRoot = ref('')

function addRoot() {
  if (newRoot.value.trim()) {
    store.addShareRoot(newRoot.value.trim())
    newRoot.value = ''
  }
}

function removeRoot(index: number) {
  store.removeShareRoot(index)
}
</script>

<template>
  <div class="flex-1 p-6 overflow-y-auto">
    <div class="max-w-lg mx-auto space-y-6">
      <div class="flex items-center gap-3">
        <button
          class="rounded-lg px-3 py-1.5 text-sm text-muted-foreground hover:bg-accent transition-colors"
          @click="router.push('/chat')"
        >
          &larr; {{ i18n('settings.back') }}
        </button>
        <h1 class="text-lg font-semibold text-foreground">
          {{ i18n('settings.title') }}
        </h1>
      </div>

      <div class="glass-panel p-5">
        <h2 class="text-sm font-medium text-foreground">
          {{ i18n('settings.sharedDirs') }}
        </h2>
        <p class="mt-1 text-xs text-muted-foreground">
          {{ i18n('settings.sharedDirsHint') }}
        </p>

        <div class="mt-3 space-y-2">
          <div
            v-for="(root, idx) in store.shareRoots"
            :key="idx"
            class="flex items-center justify-between rounded-lg bg-muted px-3 py-2 text-sm"
          >
            <span class="font-mono text-foreground">{{ root }}</span>
            <button
              class="text-destructive hover:text-destructive/80 text-xs"
              @click="removeRoot(idx)"
            >
              {{ i18n('settings.remove') }}
            </button>
          </div>
        </div>

        <div class="mt-3 flex gap-2">
          <input
            v-model="newRoot"
            type="text"
            class="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
            placeholder="~/Documents/Projects"
            @keyup.enter="addRoot"
          >
          <button
            class="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
            @click="addRoot"
          >
            {{ i18n('settings.add') }}
          </button>
        </div>
      </div>

      <!-- Connection info -->
      <div class="glass-panel p-5">
        <h2 class="text-sm font-medium text-foreground">
          {{ i18n('settings.connection') }}
        </h2>
        <div class="mt-3 space-y-2 text-sm">
          <div class="flex justify-between">
            <span class="text-muted-foreground">{{ i18n('settings.server') }}</span>
            <span class="font-mono text-foreground">{{ store.serverURL }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">{{ i18n('settings.bot') }}</span>
            <span class="text-foreground">{{ store.botName }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">{{ i18n('settings.hostname') }}</span>
            <span class="font-mono text-foreground">{{ store.hostname }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
