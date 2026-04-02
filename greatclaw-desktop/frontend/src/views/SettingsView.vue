<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '../stores/app'

const router = useRouter()
const store = useAppStore()
const newRoot = ref('')

function addRoot() {
  if (newRoot.value.trim()) {
    store.addShareRoot(newRoot.value.trim())
    newRoot.value = ''
  }
}

function removeRoot(index) {
  store.removeShareRoot(index)
}
</script>

<template>
  <div class="min-h-screen bg-gray-50 p-4">
    <div class="mx-auto max-w-sm space-y-4">
      <div class="flex items-center gap-2">
        <button
          class="rounded-md px-3 py-1 text-sm text-gray-600 hover:bg-gray-200"
          @click="router.push('/status')"
        >
          &larr; Back
        </button>
        <h1 class="text-lg font-semibold text-gray-900">
          Settings
        </h1>
      </div>

      <div class="rounded-lg border bg-white p-4 shadow-sm">
        <h2 class="text-sm font-medium text-gray-900">
          Shared Directories
        </h2>
        <p class="mt-1 text-xs text-gray-500">
          These directories will be accessible to your AI bot.
        </p>

        <div class="mt-3 space-y-2">
          <div
            v-for="(root, i) in store.shareRoots"
            :key="root"
            class="flex items-center justify-between rounded-md bg-gray-50 px-3 py-2 text-sm"
          >
            <span class="font-mono text-gray-700">{{ root }}</span>
            <button
              class="text-red-500 hover:text-red-700"
              @click="removeRoot(i)"
            >
              Remove
            </button>
          </div>
        </div>

        <div class="mt-3 flex gap-2">
          <input
            v-model="newRoot"
            type="text"
            class="flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm"
            placeholder="~/Documents/Projects"
            @keyup.enter="addRoot"
          >
          <button
            class="rounded-md bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800"
            @click="addRoot"
          >
            Add
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
