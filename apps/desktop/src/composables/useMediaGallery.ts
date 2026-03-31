import { computed, ref, type Ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useAppStore } from '@/stores/app'
import type { ChatMessage } from '@/stores/chat'
import type { MediaGalleryItem } from '@/components/chat/media-gallery-lightbox.vue'

function isMediaType(att: Record<string, unknown>): boolean {
  const type = String(att.type ?? '').toLowerCase()
  if (type === 'image' || type === 'gif' || type === 'video') return true
  const mime = String(att.mime ?? '').toLowerCase()
  return mime.startsWith('image/') || mime.startsWith('video/')
}

function isBrowserAccessibleUrl(url: string): boolean {
  if (!url) return false
  const lower = url.toLowerCase()
  return lower.startsWith('http://') || lower.startsWith('https://') || lower.startsWith('data:')
}

function resolveBotId(att: Record<string, unknown>): string {
  let botId = String(att.bot_id ?? '').trim()
  if (botId) return botId
  const meta = att.metadata as Record<string, unknown> | undefined
  botId = String(meta?.bot_id ?? '').trim()
  if (botId) return botId
  try {
    const store = useChatStore()
    return (store.currentBotId ?? '').trim()
  } catch {
    return ''
  }
}

function resolveAssetApiUrl(att: Record<string, unknown>): string {
  const contentHash = String(att.content_hash ?? '').trim()
  if (!contentHash) return ''
  const botId = resolveBotId(att)
  if (!botId) return ''
  const appStore = useAppStore()
  const token = appStore.token
  const isDev = import.meta.env.DEV
  if (isDev) {
    return `/api/bots/${botId}/media/${contentHash}?token=${encodeURIComponent(token)}`
  }
  const baseUrl = appStore.serverURL.replace(/\/+$/, '')
  return `${baseUrl}/bots/${botId}/media/${contentHash}?token=${encodeURIComponent(token)}`
}

export function resolveUrl(att: Record<string, unknown>): string {
  const assetUrl = resolveAssetApiUrl(att)
  if (assetUrl) return assetUrl
  const url = String(att.url ?? '').trim()
  if (isBrowserAccessibleUrl(url)) return url
  const base64 = String(att.base64 ?? '').trim()
  if (isBrowserAccessibleUrl(base64)) return base64
  return ''
}

export { isMediaType }

function normalizeSrc(src: string): string {
  if (!src || src.startsWith('data:')) return src
  try {
    const u = new URL(src, window.location.origin)
    return u.pathname + u.search
  } catch {
    return src
  }
}

export function useMediaGallery(messages: Ref<ChatMessage[]>) {
  const openIndex = ref<number | null>(null)

  const items = computed((): MediaGalleryItem[] => {
    const result: MediaGalleryItem[] = []
    for (const msg of messages.value) {
      for (const block of msg.blocks) {
        if (block.type !== 'attachment') continue
        for (const att of block.attachments) {
          if (!isMediaType(att)) continue
          const src = resolveUrl(att)
          if (!src) continue
          const type = String(att.type ?? '').toLowerCase()
          result.push({
            src,
            type: type === 'video' ? 'video' : 'image',
            name: String(att.name ?? '').trim() || undefined,
          })
        }
      }
    }
    return result
  })

  function findIndexBySrc(src: string): number {
    const norm = normalizeSrc(src)
    if (!norm) return -1
    return items.value.findIndex((item) => normalizeSrc(item.src) === norm)
  }

  function openBySrc(src: string) {
    const idx = findIndexBySrc(src)
    if (idx >= 0) {
      openIndex.value = idx
    }
  }

  function setOpenIndex(v: number | null) {
    openIndex.value = v
  }

  return {
    items,
    openIndex,
    setOpenIndex,
    openBySrc,
    findIndexBySrc,
  }
}
