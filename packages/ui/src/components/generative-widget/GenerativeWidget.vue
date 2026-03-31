<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import DOMPurify from 'dompurify'

const props = defineProps<{
  html: string
  title?: string
  height?: number
  streaming: boolean
}>()

const emit = defineEmits<{
  'send-prompt': [prompt: string]
}>()

const shadowHost = ref<HTMLDivElement>()
let shadowRoot: ShadowRoot | null = null

// Rate limiting for sendPrompt
const RATE_LIMIT_MAX = 5
const RATE_LIMIT_WINDOW_MS = 60_000
const MAX_PROMPT_LENGTH = 500
let promptTimestamps: number[] = []

function sendPrompt(text: string) {
  if (typeof text !== 'string' || !text.trim()) return

  const trimmed = text.trim().slice(0, MAX_PROMPT_LENGTH)

  // Rate limiting
  const now = Date.now()
  promptTimestamps = promptTimestamps.filter(t => now - t < RATE_LIMIT_WINDOW_MS)
  if (promptTimestamps.length >= RATE_LIMIT_MAX) {
    console.warn('[GenerativeWidget] sendPrompt rate limit exceeded')
    return
  }
  promptTimestamps.push(now)

  emit('send-prompt', trimmed)
}

// CDN allowlist pattern
const ALLOWED_SCRIPT_SRC_PATTERN = /^https:\/\/cdn\.jsdelivr\.net\/npm\//

type ExtractedScript = {
  src?: string
  text?: string
}

type LoadableScriptElement = HTMLScriptElement & {
  __loaded?: boolean
}

// Configure DOMPurify to allow sendPrompt onclick pattern
function createPurifyInstance() {
  const purify = DOMPurify(window)

  // Allow only onclick="sendPrompt('...')" — strict match, no trailing code
  purify.addHook('uponSanitizeAttribute', (_node: Element, data) => {
    if (data.attrName === 'onclick') {
      const val = data.attrValue.trim()
      // Only allow: sendPrompt('...') or sendPrompt("...") — nothing after closing paren
      if (/^sendPrompt\s*\(\s*(['"])[\s\S]*?\1\s*\)\s*;?\s*$/.test(val)) {
        data.forceKeepAttr = true
      }
    }
  })

  return purify
}

function sanitizeHTML(html: string): string {
  const purify = createPurifyInstance()
  return purify.sanitize(html, {
    ADD_TAGS: ['style'],
    ADD_ATTR: ['onclick'],
    ALLOW_UNKNOWN_PROTOCOLS: false,
    WHOLE_DOCUMENT: false,
  })
}

function extractScriptsFromHTML(html: string): { htmlWithoutScripts: string, scripts: ExtractedScript[] } {
  const template = document.createElement('template')
  template.innerHTML = html

  const scriptContents: ExtractedScript[] = []
  const scripts = template.content.querySelectorAll('script')
  scripts.forEach((s) => {
    if (s.src && ALLOWED_SCRIPT_SRC_PATTERN.test(s.src)) {
      scriptContents.push({ src: s.src })
    } else if (!s.src && s.textContent?.trim()) {
      scriptContents.push({ text: s.textContent })
    }
    s.remove()
  })

  return {
    htmlWithoutScripts: template.innerHTML,
    scripts: scriptContents,
  }
}

function renderToShadow(html: string) {
  if (!shadowRoot) return

  const { htmlWithoutScripts, scripts: scriptContents } = extractScriptsFromHTML(html)
  const sanitized = sanitizeHTML(htmlWithoutScripts)

  // Parse sanitized HTML after script extraction.
  const template = document.createElement('template')
  template.innerHTML = sanitized
  const fragment = template.content

  // Apply CSS variables from host theme
  const themeStyle = document.createElement('style')
  themeStyle.textContent = `
    :host {
      display: block;
      font-family: system-ui, -apple-system, sans-serif;
      color: var(--foreground, #0a0a0a);
      line-height: 1.5;
    }
  `

  // Clear shadow root and rebuild
  while (shadowRoot.firstChild) {
    shadowRoot.removeChild(shadowRoot.firstChild)
  }
  shadowRoot.appendChild(themeStyle)
  shadowRoot.appendChild(fragment)

  // Inject sendPrompt as a global function so onclick="sendPrompt('...')" works
  const bridgeScript = document.createElement('script')
  bridgeScript.textContent = `window.sendPrompt = window.__widgetSendPrompt_${shadowHostId.value};`
  shadowRoot.appendChild(bridgeScript)

  // Execute scripts: load CDN scripts first, then run inline scripts
  const cdnScripts = scriptContents.filter(s => s.src)
  const inlineScripts = scriptContents.filter(s => s.text)

  // Store shadow root reference on window so inline scripts can access it
  const srKey = `__widgetShadowRoot_${shadowHostId.value}`
  Reflect.set(window, srKey, shadowRoot)

  // Shadow-root-aware DOM query shim — uses stored shadowRoot reference, not document.currentScript
  const shimCode = [
    `var __sr = window['${srKey}'];`,
    'var document_getElementById = function(id) { return __sr.getElementById(id); };',
    'var document_querySelector = function(sel) { return __sr.querySelector(sel); };',
    'var document_querySelectorAll = function(sel) { return __sr.querySelectorAll(sel); };',
  ].join('\n')

  // Auto-fix chart containers with zero height before scripts run.
  // ECharts/Chart.js need non-zero dimensions to render; LLMs often forget inline styles.
  function ensureChartContainerSizing() {
    if (!shadowRoot) return
    shadowRoot.querySelectorAll('div[id], canvas[id]').forEach((el) => {
      const div = el as HTMLElement
      if (div.offsetHeight === 0) {
        div.style.width = div.style.width || '100%'
        div.style.height = div.style.height || '400px'
      }
    })
  }

  // Track scripts we add to document.head so we can clean them up
  const headScripts: HTMLScriptElement[] = []

  function runInlineScripts() {
    ensureChartContainerSizing()
    if (!shadowRoot) return
    for (const script of inlineScripts) {
      const el = document.createElement('script')
      // Rewrite document.getElementById/querySelector calls and provide sendPrompt
      const rewritten = script.text!
        .replace(/\bdocument\.getElementById\b/g, 'document_getElementById')
        .replace(/\bdocument\.querySelector\b(?!All)/g, 'document_querySelector')
        .replace(/\bdocument\.querySelectorAll\b/g, 'document_querySelectorAll')
      el.textContent = `(function(sendPrompt) { ${shimCode}\n${rewritten} })(window.__widgetSendPrompt_${shadowHostId.value});`
      // Execute inline scripts in document.head (scripts in Shadow DOM may not execute reliably)
      document.head.appendChild(el)
      headScripts.push(el)
    }
  }

  // Load CDN scripts in document.head with deduplication (not in Shadow DOM where they may fail to execute)
  function loadCDNScripts(scripts: Array<{ src?: string }>, onDone: () => void) {
    if (scripts.length === 0) { onDone(); return }
    let loaded = 0
    const total = scripts.length
    const onOne = () => { loaded++; if (loaded >= total) onDone() }
    for (const script of scripts) {
      const src = script.src!
      // Deduplicate: check if this CDN script is already loaded in head
      const existing = document.head.querySelector(`script[src="${CSS.escape(src)}"]`) as LoadableScriptElement | null
      if (existing) {
        // Script tag exists — wait for it if still loading.
        // We can't rely on addEventListener('load') because the event may have already fired.
        // Instead, poll the __loaded flag which is set by the original loader's onload.
        if (existing.__loaded === true) {
          onOne()
        } else {
          const poll = setInterval(() => {
            if (existing.__loaded === true) {
              clearInterval(poll)
              onOne()
            }
          }, 20)
          // Safety timeout: give up after 30s
          setTimeout(() => { clearInterval(poll); onOne() }, 30000)
        }
        continue
      }
      const el = document.createElement('script') as LoadableScriptElement
      el.src = src
      el.onload = () => { el.__loaded = true; onOne() }
      el.onerror = () => { console.warn('[GenerativeWidget] Failed to load CDN script:', src); el.__loaded = true; onOne() }
      document.head.appendChild(el)
      headScripts.push(el)
    }
  }

  loadCDNScripts(cdnScripts, runInlineScripts)
}

// Unique ID for this widget instance's sendPrompt bridge
const shadowHostId = ref(Math.random().toString(36).slice(2, 10))

onMounted(() => {
  if (!shadowHost.value) return

  shadowRoot = shadowHost.value.attachShadow({ mode: 'open' })

  // Expose sendPrompt as a global bridge for script execution inside shadow
  Reflect.set(window, `__widgetSendPrompt_${shadowHostId.value}`, sendPrompt)

  if (props.html) {
    renderToShadow(props.html)
  }
})

onBeforeUnmount(() => {
  // Cleanup global bridges
  Reflect.deleteProperty(window, `__widgetSendPrompt_${shadowHostId.value}`)
  Reflect.deleteProperty(window, `__widgetShadowRoot_${shadowHostId.value}`)
  Reflect.deleteProperty(window, 'sendPrompt')
})

watch(() => props.html, async (newHtml) => {
  if (newHtml) {
    await nextTick()
    renderToShadow(newHtml)
  }
})
</script>

<template>
  <div class="generative-widget">
    <div
      v-if="title"
      class="widget-header"
    >
      <span class="widget-title">{{ title }}</span>
      <span
        v-if="streaming"
        class="widget-streaming"
      />
    </div>
    <div
      ref="shadowHost"
      class="widget-shadow-host"
      :style="height ? { height: `${height}px` } : undefined"
    />
  </div>
</template>

<style scoped>
.generative-widget {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  overflow: hidden;
  background: hsl(var(--background));
}

.widget-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid hsl(var(--border));
  background: hsl(var(--muted));
}

.widget-title {
  font-size: 0.8125rem;
  font-weight: 500;
  color: hsl(var(--foreground));
}

.widget-streaming {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: hsl(var(--primary));
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}

.widget-shadow-host {
  min-height: 40px;
  overflow: auto;
}
</style>
