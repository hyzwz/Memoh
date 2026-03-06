<template>
  <div class="flex flex-col items-center justify-center py-20 text-muted-foreground">
    <svg v-if="icon === 'route'" xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21 3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5" /></svg>
    <svg v-else-if="icon === 'wallet'" xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M21 12a2.25 2.25 0 0 0-2.25-2.25H15a3 3 0 1 1-6 0H5.25A2.25 2.25 0 0 0 3 12m18 0v6a2.25 2.25 0 0 1-2.25 2.25H5.25A2.25 2.25 0 0 1 3 18v-6m18 0V9M3 12V9m18 0a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 9m18 0V6a2.25 2.25 0 0 0-2.25-2.25H5.25A2.25 2.25 0 0 0 3 6v3" /></svg>
    <svg v-else-if="icon === 'hand'" xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M10.05 4.575a1.575 1.575 0 1 0-3.15 0v3.15M10.05 4.575a1.575 1.575 0 0 1 3.15 0M10.05 4.575v3.15M3.75 4.575a1.575 1.575 0 0 0-3.15 0v3.15m3.15-3.15v3.15m0 0v3.375c0 .621.504 1.125 1.125 1.125h2.25M10.05 7.725V12m0 0v3.375c0 .621.504 1.125 1.125 1.125h5.25c.621 0 1.125-.504 1.125-1.125V7.725" /><path stroke-linecap="round" stroke-linejoin="round" d="M7.5 10.5h4.5m-4.5 3h4.5m-4.5 3h4.5" /></svg>
    <svg v-else-if="icon === 'building'" xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 21h16.5M4.5 3h15M5.25 3v18m13.5-18v18M9 6.75h1.5m-1.5 3h1.5m-1.5 3h1.5m3-6H15m-1.5 3H15m-1.5 3H15M9 21v-3.375c0-.621.504-1.125 1.125-1.125h3.75c.621 0 1.125.504 1.125 1.125V21" /></svg>
    <svg v-else xmlns="http://www.w3.org/2000/svg" class="size-12 mb-4 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="m20.25 7.5-.625 10.632a2.25 2.25 0 0 1-2.247 2.118H6.622a2.25 2.25 0 0 1-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0-3-3m3 3 3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125Z" /></svg>
    <p class="text-sm">{{ message }}</p>
    <p v-if="sub" class="text-xs mt-1 text-muted-foreground/60">{{ sub }}</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  icon?: 'route' | 'wallet' | 'hand' | 'building' | 'default'
  message: string
  sub?: string
}>()
</script>
