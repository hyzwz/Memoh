<template>
  <div class="flex h-full min-h-0 flex-col">
    <div class="border-b px-4 py-3">
      <p class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
        {{ $t('chat.sessions') }}
      </p>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto">
      <div
        v-if="!currentBotId"
        class="px-4 py-6 text-sm text-muted-foreground"
      >
        {{ $t('chat.selectBotFirst') }}
      </div>

      <div
        v-else-if="loadingChats"
        class="flex justify-center py-4"
      >
        <FontAwesomeIcon
          :icon="['fas', 'spinner']"
          class="size-4 animate-spin text-muted-foreground"
        />
      </div>

      <div
        v-else-if="!chats.length"
        class="px-4 py-6 text-sm text-muted-foreground"
      >
        {{ $t('chat.emptySessions') }}
      </div>

      <div
        v-else
        class="space-y-4 px-2 py-3"
      >
        <div
          v-if="participantChats.length"
          class="space-y-2"
        >
          <p class="px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {{ $t('chat.historyParticipant') }}
          </p>
          <SessionItem
            v-for="session in participantChats"
            :key="session.id"
            :session="session"
            :active="chatId === session.id"
            @select="chatStore.selectChat(session.id)"
          />
        </div>

        <div
          v-if="observedChats.length"
          class="space-y-2"
        >
          <p class="px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {{ $t('chat.historyObserved') }}
          </p>
          <SessionItem
            v-for="session in observedChats"
            :key="session.id"
            :session="session"
            :active="chatId === session.id"
            @select="chatStore.selectChat(session.id)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useChatStore } from '@/store/chat-list'
import SessionItem from './session-item.vue'

const chatStore = useChatStore()
const { chats, chatId, currentBotId, loadingChats, participantChats, observedChats } = storeToRefs(chatStore)
</script>
