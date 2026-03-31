<template>
  <div class="flex h-screen bg-background text-foreground">
    <!-- Sidebar -->
    <aside class="w-64 flex flex-col border-r border-border bg-sidebar text-sidebar-foreground">
      <!-- Header -->
      <div class="p-4 border-b border-border">
        <h1 class="text-lg font-semibold tracking-tight">
          GreatClaw
        </h1>
        <p class="text-xs text-muted-foreground truncate">
          {{ appStore.botName || 'Desktop' }}
        </p>
      </div>

      <!-- New Chat -->
      <div class="p-3">
        <button
          class="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
          @click="handleNewChat"
        >
          <FontAwesomeIcon
            :icon="['fas', 'plus']"
            class="size-3.5"
          />
          {{ i18n('app.newChat') }}
        </button>
      </div>

      <!-- Chat list -->
      <div class="flex-1 overflow-y-auto scrollbar-thin px-2">
        <div
          v-for="chat in chatStore.participantChats"
          :key="chat.id"
          class="flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer text-sm transition-colors mb-0.5"
          :class="chat.id === chatStore.chatId
            ? 'bg-sidebar-accent text-sidebar-accent-foreground'
            : 'hover:bg-sidebar-accent/50 text-sidebar-foreground'"
          @click="handleSelectChat(chat.id)"
        >
          <FontAwesomeIcon
            :icon="['fas', 'message']"
            class="size-3 shrink-0 text-muted-foreground"
          />
          <span class="truncate">{{ chat.title || 'Chat' }}</span>
        </div>
        <p
          v-if="chatStore.participantChats.length === 0 && !chatStore.loadingChats"
          class="px-3 py-4 text-xs text-muted-foreground text-center"
        >
          {{ i18n('app.noChats') }}
        </p>
      </div>

      <!-- Footer -->
      <div class="p-3 border-t border-border space-y-2">
        <!-- Theme toggle -->
        <div class="flex items-center gap-1">
          <button
            v-for="th in themes"
            :key="th.value"
            class="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 text-xs rounded-md transition-colors"
            :class="settingsStore.theme === th.value
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-sidebar-accent text-muted-foreground'"
            @click="settingsStore.setTheme(th.value)"
          >
            <FontAwesomeIcon
              :icon="['fas', th.icon]"
              class="size-3"
            />
          </button>
        </div>

        <!-- Language toggle -->
        <div class="flex items-center gap-1">
          <button
            class="flex-1 px-2 py-1.5 text-xs rounded-md transition-colors"
            :class="settingsStore.locale === 'zh'
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-sidebar-accent text-muted-foreground'"
            @click="settingsStore.setLocale('zh')"
          >
            中文
          </button>
          <button
            class="flex-1 px-2 py-1.5 text-xs rounded-md transition-colors"
            :class="settingsStore.locale === 'en'
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-sidebar-accent text-muted-foreground'"
            @click="settingsStore.setLocale('en')"
          >
            EN
          </button>
        </div>

        <!-- Actions -->
        <div class="flex items-center gap-2">
          <button
            class="flex-1 flex items-center justify-center gap-1.5 px-2 py-1.5 text-xs rounded-md hover:bg-sidebar-accent text-muted-foreground transition-colors"
            @click="$router.push('/settings')"
          >
            <FontAwesomeIcon
              :icon="['fas', 'gear']"
              class="size-3"
            />
            {{ i18n('app.settings') }}
          </button>
          <button
            class="flex-1 flex items-center justify-center gap-1.5 px-2 py-1.5 text-xs rounded-md hover:bg-sidebar-accent text-muted-foreground transition-colors"
            @click="handleLogout"
          >
            <FontAwesomeIcon
              :icon="['fas', 'right-from-bracket']"
              class="size-3"
            />
            {{ i18n('app.logout') }}
          </button>
        </div>

        <!-- Connection status -->
        <div class="flex items-center gap-2 px-2 text-xs text-muted-foreground">
          <span
            class="size-2 rounded-full"
            :class="appStore.connected ? 'bg-accent-success' : 'bg-accent-danger'"
          />
          {{ appStore.connected ? i18n('app.connected') : i18n('app.disconnected') }}
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 flex flex-col min-w-0">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useSettingsStore } from '@/stores/settings'
import { useRouter } from 'vue-router'
import type { Theme } from '@/stores/settings'

const appStore = useAppStore()
const chatStore = useChatStore()
const settingsStore = useSettingsStore()
const router = useRouter()

const i18n = computed(() => settingsStore.t)

const themes: { value: Theme; icon: string }[] = [
  { value: 'light', icon: 'sun' },
  { value: 'dark', icon: 'moon' },
  { value: 'deep-space', icon: 'rocket' },
]

function handleNewChat() {
  chatStore.createNewChat()
  router.push('/chat')
}

function handleSelectChat(chatId: string) {
  chatStore.selectChat(chatId)
  router.push('/chat')
}

function handleLogout() {
  appStore.logout()
  router.push('/login')
}
</script>
