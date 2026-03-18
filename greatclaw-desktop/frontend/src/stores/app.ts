import { defineStore } from 'pinia'
import { ref } from 'vue'

declare global {
  interface Window {
    go: {
      main: {
        App: {
          LoginWithPassword(serverURL: string, username: string, password: string): Promise<Record<string, any>>
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
    const result = await window.go.main.App.LoginWithPassword(
      server, username, password,
    )

    serverURL.value = server
    token.value = result.access_token || ''
    botName.value = result.bot_name || ''
    connected.value = true
    hostname.value = result.hostname || ''
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
