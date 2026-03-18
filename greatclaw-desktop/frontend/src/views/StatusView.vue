<script setup lang="ts">
import { useAppStore } from '../stores/app'
import { useRouter } from 'vue-router'

const store = useAppStore()
const router = useRouter()

function goToSettings() {
  router.push('/settings')
}

function handleLogout() {
  store.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 p-4">
    <div class="mx-auto max-w-sm space-y-4">
      <div class="rounded-lg border bg-white p-4 shadow-sm">
        <h2 class="text-lg font-semibold text-gray-900">
          Connection Status
        </h2>
        <div class="mt-3 space-y-2 text-sm">
          <div class="flex justify-between">
            <span class="text-gray-500">Status</span>
            <span :class="store.connected ? 'text-green-600' : 'text-red-600'">
              {{ store.connected ? 'Connected' : 'Disconnected' }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Tailscale IP</span>
            <span class="font-mono text-gray-900">{{ store.tsIP || '—' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Hostname</span>
            <span class="text-gray-900">{{ store.hostname || '—' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Bot</span>
            <span class="text-gray-900">{{ store.botName || '—' }}</span>
          </div>
        </div>
      </div>

      <div class="flex gap-2">
        <button
          class="flex-1 rounded-md border bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          @click="goToSettings"
        >
          Settings
        </button>
        <button
          class="flex-1 rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
          @click="handleLogout"
        >
          Logout
        </button>
      </div>
    </div>
  </div>
</template>
