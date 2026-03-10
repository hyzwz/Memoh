import type { Locale } from '@/i18n'
import { defineStore } from 'pinia'
import { useColorMode, useStorage } from '@vueuse/core'
import { useI18n } from 'vue-i18n'

export type Theme = 'light' | 'dark' | 'deep-space'

export interface Settings {
  language: Locale;
  theme: Theme;
}

export const useSettingsStore = defineStore('settings', () => {
  const colorMode = useColorMode()
  const i18n = useI18n()
  const language = useStorage<Locale>('language', 'en')
  const theme = useStorage<Theme>('theme', 'light')

  applyTheme(theme.value)
  i18n.locale.value = language.value

  const setLanguage = (value: Locale) => {
    language.value = value
    i18n.locale.value = value
  }

  const setTheme = (value: Theme) => {
    theme.value = value
    applyTheme(value)
  }

  function applyTheme(value: Theme) {
    if (value === 'deep-space') {
      colorMode.value = 'dark'
      document.documentElement.setAttribute('data-theme', 'deep-space')
    } else {
      colorMode.value = value
      document.documentElement.removeAttribute('data-theme')
    }
  }

  return {
    language,
    theme,
    setLanguage,
    setTheme,
  }
})
