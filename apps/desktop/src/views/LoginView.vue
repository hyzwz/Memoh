<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useSettingsStore } from '@/stores/settings'
import { setProxyTarget } from '@/composables/api/useChat.message-api'

const router = useRouter()
const store = useAppStore()
const chatStore = useChatStore()
const settingsStore = useSettingsStore()
const i18n = computed(() => settingsStore.t)

const serverURL = ref('http://localhost:8080')
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await store.login(serverURL.value, username.value, password.value)
    // Tell Vite dev proxy to forward /api to this server
    await setProxyTarget(serverURL.value)
    // Set the bot from login response
    if (store.botID) {
      chatStore.currentBotId = store.botID
    }
    router.push('/chat')
  } catch (e: any) {
    error.value = e.message || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <div class="w-full max-w-sm glass-panel p-8">
      <div class="text-center mb-8">
        <h1 class="text-2xl font-bold tracking-tight text-foreground">
          GreatClaw
        </h1>
        <p class="text-sm text-muted-foreground mt-1">
          {{ i18n('login.title') }}
        </p>
      </div>

      <!-- Language toggle -->
      <div class="flex justify-center gap-2 mb-6">
        <button
          class="px-3 py-1 text-xs rounded-md transition-colors"
          :class="settingsStore.locale === 'zh' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-accent'"
          @click="settingsStore.setLocale('zh')"
        >
          中文
        </button>
        <button
          class="px-3 py-1 text-xs rounded-md transition-colors"
          :class="settingsStore.locale === 'en' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-accent'"
          @click="settingsStore.setLocale('en')"
        >
          EN
        </button>
      </div>

      <form
        class="space-y-4"
        @submit.prevent="handleLogin"
      >
        <div>
          <label class="block text-sm font-medium text-foreground mb-1">{{ i18n('login.server') }}</label>
          <input
            v-model="serverURL"
            type="url"
            class="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
            placeholder="http://localhost:8080"
          >
        </div>

        <div>
          <label class="block text-sm font-medium text-foreground mb-1">{{ i18n('login.username') }}</label>
          <input
            v-model="username"
            type="text"
            class="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
            required
          >
        </div>

        <div>
          <label class="block text-sm font-medium text-foreground mb-1">{{ i18n('login.password') }}</label>
          <input
            v-model="password"
            type="password"
            class="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-1 focus:ring-ring"
            required
          >
        </div>

        <div
          v-if="error"
          class="rounded-lg bg-destructive/10 p-3 text-sm text-destructive"
        >
          {{ error }}
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
        >
          {{ loading ? i18n('login.loading') : i18n('login.submit') }}
        </button>
      </form>
    </div>
  </div>
</template>
