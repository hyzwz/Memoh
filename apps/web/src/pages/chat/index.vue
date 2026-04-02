<template>
  <div class="flex h-full min-h-0 overflow-hidden">
    <aside class="hidden w-72 shrink-0 border-r border-border md:flex md:flex-col min-h-0">
      <div class="p-4 border-b">
        <p class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {{ $t('sidebar.bots') }}
        </p>
      </div>
      <div class="min-h-0 flex-1 overflow-auto px-2">
        <BotSidebar />
      </div>
    </aside>

    <aside class="hidden w-80 shrink-0 border-r border-border md:flex md:flex-col min-h-0">
      <SessionSidebar />
    </aside>

    <div class="min-w-0 min-h-0 flex-1 overflow-hidden">
      <ChatArea />
    </div>
  </div>
</template>

<script setup lang="ts">
import BotSidebar from './components/bot-sidebar.vue'
import ChatArea from './components/chat-area.vue'
import SessionSidebar from './components/session-sidebar.vue'

</script>
