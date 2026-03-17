import { defineStore } from 'pinia'
import { ref } from 'vue'

declare global {
  interface Window {
    go: {
      main: {
        App: {
          LoginWithPassword(serverURL: string, username: string, password: string, hostname: string, platform: string): Promise<Record<string, any>>
          GetStatus(): Promise<Record<string, any>>
        }
      }
    }
  }
}

export const useAppStore = defineStore('app', () => {
  const connected = ref(false)
  const tsIP = ref('')
  const hostname = ref('')
  const botName = ref('')
  const serverURL = ref('')
  const token = ref('')
  const shareRoots = ref<string[]>(['~/Documents', '~/Desktop'])

  async function login(server: string, username: string, password: string) {
    const hn = await getHostname()
    const platform = getPlatform()

    const result = await window.go.main.App.LoginWithPassword(
      server, username, password, hn, platform,
    )

    if (result.error) {
      throw new Error(result.error)
    }

    serverURL.value = server
    token.value = result.token || ''
    botName.value = result.bot_name || ''
    connected.value = true
    hostname.value = hn
  }

  function logout() {
    connected.value = false
    tsIP.value = ''
    token.value = ''
    botName.value = ''
  }

  function addShareRoot(root: string) {
    if (!shareRoots.value.includes(root)) {
      shareRoots.value.push(root)
    }
  }

  function removeShareRoot(index: number) {
    shareRoots.value.splice(index, 1)
  }

  async function getHostname(): Promise<string> {
    // In Wails, we'd call a Go binding. Fallback for dev.
    return 'desktop'
  }

  function getPlatform(): string {
    const ua = navigator.userAgent.toLowerCase()
    if (ua.includes('mac')) return 'darwin'
    if (ua.includes('win')) return 'windows'
    return 'linux'
  }

  return {
    connected,
    tsIP,
    hostname,
    botName,
    serverURL,
    token,
    shareRoots,
    login,
    logout,
    addShareRoot,
    removeShareRoot,
  }
})
