<template>
  <section>
    <!-- Deep Space ambient orbs -->
    <div
      v-if="isDeepSpace"
      class="ambient-orb ambient-orb-cyan"
    />
    <div
      v-if="isDeepSpace"
      class="ambient-orb ambient-orb-purple"
    />

    <MainLayout>
      <template #sidebar>
        <SideBar />
      </template>
      <template #main>
        <MainContainer />
      </template>
    </MainLayout>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import MainLayout from '@/layout/main-layout/index.vue'
import SideBar from '@/components/sidebar/index.vue'
import MainContainer from '@/components/main-container/index.vue'
import { useSettingsStore } from '@/store/settings'

const { theme } = storeToRefs(useSettingsStore())
const isDeepSpace = computed(() => theme.value === 'deep-space')
</script>
