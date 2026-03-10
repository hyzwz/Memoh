<template>
  <Card class="flex flex-col">
    <CardHeader class="pb-3">
      <div class="flex items-start justify-between gap-2">
        <div class="flex items-center gap-2 min-w-0">
          <span
            v-if="template.icon"
            class="text-lg shrink-0"
          >{{ template.icon }}</span>
          <CardTitle
            class="text-base truncate"
            :title="template.name"
          >
            {{ template.name }}
          </CardTitle>
        </div>
        <Badge
          variant="secondary"
          class="shrink-0 text-xs"
        >
          {{ template.category }}
        </Badge>
      </div>
      <CardDescription
        class="line-clamp-2"
        :title="template.description"
      >
        {{ template.description || '-' }}
      </CardDescription>
    </CardHeader>
    <CardContent class="pt-0 mt-auto">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <span v-if="template.author">{{ template.author }}</span>
          <span>v{{ template.version }}</span>
          <span v-if="template.install_count">{{ template.install_count }} {{ $t('skillTemplates.installs') }}</span>
        </div>
        <div class="flex items-center gap-1">
          <template v-if="template.installed">
            <Badge
              v-if="template.customized"
              variant="outline"
              class="text-xs"
            >
              {{ $t('skillTemplates.customized') }}
            </Badge>
            <Button
              v-if="template.update_available && !template.customized"
              size="sm"
              variant="outline"
              :disabled="loading"
              @click="$emit('update', template.id!)"
            >
              <Spinner
                v-if="loading && action === 'update'"
                class="mr-1 size-3"
              />
              {{ $t('skillTemplates.update') }}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              class="text-destructive hover:text-destructive"
              :disabled="loading"
              @click="$emit('uninstall', template.id!)"
            >
              <Spinner
                v-if="loading && action === 'uninstall'"
                class="mr-1 size-3"
              />
              {{ $t('skillTemplates.uninstall') }}
            </Button>
          </template>
          <Button
            v-else
            size="sm"
            :disabled="loading"
            @click="$emit('install', template.id!)"
          >
            <Spinner
              v-if="loading && action === 'install'"
              class="mr-1 size-3"
            />
            {{ $t('skillTemplates.install') }}
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>

<script setup lang="ts">
import {
  Button, Badge, Spinner,
  Card, CardHeader, CardTitle, CardDescription, CardContent,
} from '@memoh/ui'
import type { HandlersStoreTemplateResponse } from '@memoh/sdk'

defineProps<{
  template: HandlersStoreTemplateResponse
  loading?: boolean
  action?: 'install' | 'uninstall' | 'update'
}>()

defineEmits<{
  install: [id: string]
  uninstall: [id: string]
  update: [id: string]
}>()
</script>
