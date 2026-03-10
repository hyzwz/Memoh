<template>
  <aside aria-label="Primary">
    <Sidebar
      collapsible="icon"
      role="navigation"
      aria-label="Primary"
    >
      <SidebarHeader>
        <div class="flex items-center gap-2 px-1 py-1 group-data-[collapsible=icon]:justify-center">
          <img
            src="/logo.png"
            class="size-6 shrink-0"
            alt="Memoh logo"
          >
          <span class="text-lg font-bold text-muted-foreground truncate group-data-[collapsible=icon]:hidden">
            Memoh
          </span>
        </div>
      </SidebarHeader>

      <SidebarContent class="scrollbar-thin">
        <!-- Main -->
        <SidebarGroup>
          <SidebarGroupLabel class="text-[10px] uppercase tracking-wider text-text-secondary">
            {{ $t('sidebar.groupMain') }}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem
                v-for="item in mainItems"
                :key="item.name"
              >
                <SidebarMenuButton
                  :tooltip="item.title"
                  :is-active="isItemActive(item.name)"
                  :aria-current="isItemActive(item.name) ? 'page' : undefined"
                  @click="router.push({ name: item.name })"
                >
                  <FontAwesomeIcon :icon="item.icon" />
                  <span>{{ item.title }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <!-- Bots & AI -->
        <SidebarGroup>
          <SidebarGroupLabel class="text-[10px] uppercase tracking-wider text-text-secondary">
            {{ $t('sidebar.groupBotsAi') }}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem
                v-for="item in botsAiItems"
                :key="item.name"
              >
                <SidebarMenuButton
                  :tooltip="item.title"
                  :is-active="isItemActive(item.name)"
                  :aria-current="isItemActive(item.name) ? 'page' : undefined"
                  @click="router.push({ name: item.name })"
                >
                  <FontAwesomeIcon :icon="item.icon" />
                  <span>{{ item.title }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <!-- Providers -->
        <SidebarGroup>
          <SidebarGroupLabel class="text-[10px] uppercase tracking-wider text-text-secondary">
            {{ $t('sidebar.groupProviders') }}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem
                v-for="item in providerItems"
                :key="item.name"
              >
                <SidebarMenuButton
                  :tooltip="item.title"
                  :is-active="isItemActive(item.name)"
                  :aria-current="isItemActive(item.name) ? 'page' : undefined"
                  @click="router.push({ name: item.name })"
                >
                  <FontAwesomeIcon :icon="item.icon" />
                  <span>{{ item.title }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <!-- Analytics -->
        <SidebarGroup>
          <SidebarGroupLabel class="text-[10px] uppercase tracking-wider text-text-secondary">
            {{ $t('sidebar.groupAnalytics') }}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem
                v-for="item in analyticsItems"
                :key="item.name"
              >
                <SidebarMenuButton
                  :tooltip="item.title"
                  :is-active="isItemActive(item.name)"
                  :aria-current="isItemActive(item.name) ? 'page' : undefined"
                  @click="router.push({ name: item.name })"
                >
                  <FontAwesomeIcon :icon="item.icon" />
                  <span>{{ item.title }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <!-- Enterprise -->
        <SidebarGroup>
          <SidebarGroupLabel class="text-[10px] uppercase tracking-wider text-text-secondary">
            {{ $t('sidebar.groupEnterprise') }}
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem
                v-for="item in enterpriseItems"
                :key="item.name"
              >
                <SidebarMenuButton
                  :tooltip="item.title"
                  :is-active="isItemActive(item.name)"
                  :aria-current="isItemActive(item.name) ? 'page' : undefined"
                  @click="router.push({ name: item.name })"
                >
                  <FontAwesomeIcon :icon="item.icon" />
                  <span>{{ item.title }}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter class="border-t border-glass-border p-2">
        <SidebarMenu>
          <!-- Theme toggle -->
          <SidebarMenuItem>
            <SidebarMenuButton
              :tooltip="$t('settings.theme')"
              @click="cycleTheme"
            >
              <FontAwesomeIcon :icon="themeIcon" />
              <span class="truncate text-sm">{{ themeLabel }}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <!-- Settings -->
          <SidebarMenuItem>
            <SidebarMenuButton
              :tooltip="$t('sidebar.settings')"
              :is-active="isItemActive('settings')"
              @click="router.push({ name: 'settings' })"
            >
              <FontAwesomeIcon :icon="['fas', 'gear']" />
              <span class="truncate text-sm">{{ $t('sidebar.settings') }}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <!-- User -->
          <SidebarMenuItem>
            <SidebarMenuButton
              :tooltip="displayTitle"
              @click="router.push({ name: 'settings' })"
            >
              <Avatar class="size-4 shrink-0">
                <AvatarImage
                  v-if="userInfo.avatarUrl"
                  :src="userInfo.avatarUrl"
                  :alt="displayTitle"
                />
                <AvatarFallback class="text-[7px] text-muted-foreground">
                  {{ avatarFallback }}
                </AvatarFallback>
              </Avatar>
              <span class="truncate text-sm">{{ displayNameLabel }}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  </aside>
</template>

<script setup lang="ts">
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@memoh/ui'
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/store/user'
import { useSettingsStore } from '@/store/settings'
import type { Theme } from '@/store/settings'
import { useAvatarInitials } from '@/composables/useAvatarInitials'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const { userInfo } = useUserStore()
const settingsStore = useSettingsStore()
const { theme } = storeToRefs(settingsStore)

const displayNameLabel = computed(() =>
  userInfo.displayName || userInfo.username || userInfo.id || '-',
)
const displayTitle = computed(() =>
  userInfo.displayName || userInfo.username || userInfo.id || t('settings.user'),
)
const avatarFallback = useAvatarInitials(() => displayTitle.value, 'U')

function isItemActive(name: string): boolean {
  return new RegExp(`^/${name}(\\b|/)`).test(route.path)
}

// Theme cycling: light → dark → deep-space → light
const themeOrder: Theme[] = ['light', 'dark', 'deep-space']

const themeIcon = computed(() => {
  switch (theme.value) {
    case 'light': return ['fas', 'sun']
    case 'dark': return ['fas', 'moon']
    case 'deep-space': return ['fas', 'star']
    default: return ['fas', 'sun']
  }
})

const themeLabel = computed(() => {
  switch (theme.value) {
    case 'light': return t('settings.themeLight')
    case 'dark': return t('settings.themeDark')
    case 'deep-space': return t('settings.themeDeepSpace')
    default: return t('settings.themeLight')
  }
})

function cycleTheme() {
  const currentIndex = themeOrder.indexOf(theme.value)
  const nextIndex = (currentIndex + 1) % themeOrder.length
  settingsStore.setTheme(themeOrder[nextIndex])
}

// --- Navigation Groups ---

const mainItems = computed(() => [
  {
    title: t('sidebar.chat'),
    name: 'chat',
    icon: ['fas', 'comment-dots'],
  },
  {
    title: t('sidebar.virtualOffice'),
    name: 'virtual-office',
    icon: ['fas', 'building-user'],
  },
])

const botsAiItems = computed(() => [
  {
    title: t('sidebar.bots'),
    name: 'bots',
    icon: ['fas', 'robot'],
  },
  {
    title: t('sidebar.models'),
    name: 'models',
    icon: ['fas', 'cubes'],
  },
  {
    title: t('sidebar.skillTemplates'),
    name: 'skill-templates',
    icon: ['fas', 'wand-magic-sparkles'],
  },
])

const providerItems = computed(() => [
  {
    title: t('sidebar.searchProvider'),
    name: 'search-providers',
    icon: ['fas', 'globe'],
  },
  {
    title: t('sidebar.memoryProvider'),
    name: 'memory-providers',
    icon: ['fas', 'brain'],
  },
  {
    title: t('sidebar.emailProvider'),
    name: 'email-providers',
    icon: ['fas', 'envelope'],
  },
  {
    title: t('sidebar.browserContexts'),
    name: 'browser-contexts',
    icon: ['fas', 'window-maximize'],
  },
])

const analyticsItems = computed(() => [
  {
    title: t('sidebar.usage'),
    name: 'usage',
    icon: ['fas', 'chart-line'],
  },
  {
    title: t('sidebar.cockpit'),
    name: 'cockpit',
    icon: ['fas', 'gauge-high'],
  },
])

const enterpriseItems = computed(() => [
  {
    title: t('sidebar.audit'),
    name: 'audit',
    icon: ['fas', 'clipboard-list'],
  },
  {
    title: t('sidebar.modelRouting'),
    name: 'model-routing',
    icon: ['fas', 'route'],
  },
  {
    title: t('sidebar.budgets'),
    name: 'budgets',
    icon: ['fas', 'wallet'],
  },
  {
    title: t('sidebar.hands'),
    name: 'hands',
    icon: ['fas', 'hand-sparkles'],
  },
  {
    title: t('sidebar.departments'),
    name: 'departments',
    icon: ['fas', 'building'],
  },
])
</script>
