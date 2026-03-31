import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useColorMode, useStorage } from '@vueuse/core'

export type Theme = 'light' | 'dark' | 'deep-space'
export type Locale = 'en' | 'zh'

const messages: Record<Locale, Record<string, string>> = {
  en: {
    'app.title': 'GreatClaw',
    'app.newChat': 'New Chat',
    'app.noChats': 'No conversations yet',
    'app.settings': 'Settings',
    'app.logout': 'Logout',
    'app.connected': 'Connected',
    'app.disconnected': 'Disconnected',
    'chat.greeting': 'Start a conversation...',
    'chat.selectBot': 'Select a bot',
    'chat.selectBotHint': 'Choose a bot to start chatting',
    'chat.placeholder': 'Type a message...',
    'chat.readonly': 'Read-only chat',
    'chat.thinking': 'Thinking...',
    'chat.thinkingInProgress': 'Thinking in progress',
    'chat.thinkingDone': 'Thinking done',
    'chat.done': 'Done',
    'chat.running': 'Running...',
    'chat.input': 'Input',
    'chat.result': 'Result',
    'chat.changes': 'Changes',
    'chat.content': 'Content',
    'chat.output': 'Output',
    'chat.searchResults': 'Search Results',
    'login.title': 'Sign in to your Memoh server',
    'login.server': 'Server URL',
    'login.username': 'Username',
    'login.password': 'Password',
    'login.submit': 'Sign In',
    'login.loading': 'Connecting...',
    'settings.title': 'Settings',
    'settings.back': 'Back',
    'settings.sharedDirs': 'Shared Directories',
    'settings.sharedDirsHint': 'These directories will be accessible to your AI bot.',
    'settings.add': 'Add',
    'settings.remove': 'Remove',
    'settings.connection': 'Connection',
    'settings.server': 'Server',
    'settings.bot': 'Bot',
    'settings.hostname': 'Hostname',
  },
  zh: {
    'app.title': 'GreatClaw',
    'app.newChat': '新对话',
    'app.noChats': '暂无对话',
    'app.settings': '设置',
    'app.logout': '退出',
    'app.connected': '已连接',
    'app.disconnected': '未连接',
    'chat.greeting': '开始对话...',
    'chat.selectBot': '选择一个 Bot',
    'chat.selectBotHint': '选择要聊天的 Bot',
    'chat.placeholder': '输入消息...',
    'chat.readonly': '只读对话',
    'chat.thinking': '思考中...',
    'chat.thinkingInProgress': '正在思考',
    'chat.thinkingDone': '思考完成',
    'chat.done': '完成',
    'chat.running': '运行中...',
    'chat.input': '输入',
    'chat.result': '结果',
    'chat.changes': '变更',
    'chat.content': '内容',
    'chat.output': '输出',
    'chat.searchResults': '搜索结果',
    'login.title': '登录 Memoh 服务器',
    'login.server': '服务器地址',
    'login.username': '用户名',
    'login.password': '密码',
    'login.submit': '登录',
    'login.loading': '连接中...',
    'settings.title': '设置',
    'settings.back': '返回',
    'settings.sharedDirs': '共享目录',
    'settings.sharedDirsHint': '这些目录将对你的 AI Bot 开放访问。',
    'settings.add': '添加',
    'settings.remove': '移除',
    'settings.connection': '连接信息',
    'settings.server': '服务器',
    'settings.bot': 'Bot',
    'settings.hostname': '主机名',
  },
}

export const useSettingsStore = defineStore('settings', () => {
  const colorMode = useColorMode()
  const theme = useStorage<Theme>('theme', 'light')
  const locale = useStorage<Locale>('locale', 'zh')

  applyTheme(theme.value)

  const setTheme = (value: Theme) => {
    theme.value = value
    applyTheme(value)
  }

  const setLocale = (value: Locale) => {
    locale.value = value
  }

  const t = computed(() => {
    const dict = messages[locale.value] ?? messages.en
    return (key: string, fallback?: string) => dict[key] ?? fallback ?? key
  })

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
    theme,
    locale,
    t,
    setTheme,
    setLocale,
  }
})
