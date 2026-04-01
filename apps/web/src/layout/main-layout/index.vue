<template>
  <section class="flex">
    <sidebar-provider
      :default-open="sidebarDefaultOpen"
      class="**:data-[sidebar=sidebar]:bg-transparent"
    >
      <slot name="sidebar" />
      <slot name="main" />
    </sidebar-provider>
  </section>
</template>
<script setup lang="ts">
import { SidebarProvider } from '@memoh/ui'
import { getSidebarDefaultOpen } from '../sidebar-default-open'

const sidebarDefaultOpen = getSidebarDefaultOpen(document.cookie)
</script>
