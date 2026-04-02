<template>
  <SidebarInset class="grid grid-rows-[auto_1fr]">
    <header
      class="flex h-12 shrink-0 items-center gap-2 border-b border-glass-border bg-surface-card/50 backdrop-blur-sm"
    >
      <div class="flex flex-1 items-center gap-2 px-4">
        <SidebarTrigger class="-ml-1" />
        <Separator
          orientation="vertical"
          class="mr-2 data-[orientation=vertical]:h-4"
        />
        <Breadcrumb>
          <BreadcrumbList>
            <template
              v-for="(breadcrumbItem, index) in curBreadcrumb"
              :key="breadcrumbItem"
            >
              <template v-if="(index + 1) !== curBreadcrumb.length">
                <BreadcrumbItem class="hidden md:block">
                  <BreadcrumbLink :href="breadcrumbItem.path">
                    {{ breadcrumbItem.breadcrumb }}
                  </BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator />
              </template>

              <BreadcrumbItem v-else>
                <BreadcrumbPage class="font-medium text-text-primary">
                  {{ breadcrumbItem.breadcrumb }}
                </BreadcrumbPage>
              </BreadcrumbItem>
            </template>
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
    <section class="w-full relative min-h-0">
      <h1 class="sr-only">
        {{ currentPageTitle }}
      </h1>
      <router-view v-slot="{ Component }">
        <KeepAlive>
          <component
            :is="Component"
            v-if="route.path === '/chat'"
            class="absolute inset-0"
          />
          <ScrollArea
            v-else
            class="absolute! inset-0"
          >
            <component :is="Component" />
          </ScrollArea>
        </KeepAlive>
      </router-view>
    </section>
  </SidebarInset>
</template>

<script setup lang="ts">
import {
  SidebarTrigger, SidebarInset, Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
  Separator,
} from '@memoh/ui'
import { useRoute } from 'vue-router'
import { computed, unref } from 'vue'
import { ScrollArea } from '@memoh/ui'

const route = useRoute()

const curBreadcrumb = computed(() => {
  return route.matched
    .filter(routeItem => routeItem.meta['breadcrumb'])
    .map(routeItem => {
      const raw = routeItem.meta['breadcrumb']
      return {
        path: routeItem.path,
        breadcrumb: typeof raw === 'function' ? raw(route) : raw,
      }
    })
})

const currentPageTitle = computed(() => {
  const last = curBreadcrumb.value[curBreadcrumb.value.length - 1]
  const title = String(unref(last?.breadcrumb) ?? '').trim()
  return title || 'GreatClaw'
})
</script>
