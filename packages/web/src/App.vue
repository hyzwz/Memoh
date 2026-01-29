<script setup lang="ts">
import { RouterView } from 'vue-router'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@memoh/ui'
import SvgIcon from '@jamescoyle/vue-icon'
import { mdiTranslate } from '@mdi/js'

</script>

<template>
  <section>
    <div
      class="fixed top-8 right-2 z-9999 [&:is(:has([data-state=open]))_.translate-icon]:opacity-100"
    >
      <DropdownMenu>
        <DropdownMenuTrigger class="ml-auto mr-4 cursor-pointer">
          <svg-icon
            type="mdi"
            :path="mdiTranslate"
            class="translate-icon opacity-30 hover:opacity-100"
          />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem @click="$i18n.locale= 'zh'">
            中文
          </DropdownMenuItem>
          <DropdownMenuItem @click="$i18n.locale = 'en'">
            English
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
    <RouterView />
  </section>
</template>