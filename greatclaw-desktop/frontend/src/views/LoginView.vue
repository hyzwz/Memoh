<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '../stores/app'

const router = useRouter()
const store = useAppStore()

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
    router.push('/status')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-50 p-4">
    <div class="w-full max-w-sm rounded-lg border bg-white p-6 shadow-sm">
      <h1 class="mb-6 text-center text-xl font-semibold text-gray-900">
        greatClaw
      </h1>

      <form
        class="space-y-4"
        @submit.prevent="handleLogin"
      >
        <div>
          <label class="block text-sm font-medium text-gray-700">
            Server URL
          </label>
          <input
            v-model="serverURL"
            type="url"
            class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="http://localhost:8080"
          >
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700">
            Username
          </label>
          <input
            v-model="username"
            type="text"
            class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="Enter username"
            required
          >
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700">
            Password
          </label>
          <input
            v-model="password"
            type="password"
            class="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            placeholder="Enter password"
            required
          >
        </div>

        <div
          v-if="error"
          class="rounded-md bg-red-50 p-3 text-sm text-red-600"
        >
          {{ error }}
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 disabled:opacity-50"
        >
          {{ loading ? 'Connecting...' : 'Login' }}
        </button>
      </form>
    </div>
  </div>
</template>
