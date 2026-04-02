import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath } from 'url'
import type { IncomingMessage, ServerResponse } from 'node:http'

function isServerResponse(value: unknown): value is ServerResponse<IncomingMessage> {
  return typeof value === 'object' && value !== null && 'writeHead' in value && 'end' in value
}

// Dynamic proxy target — updated by the frontend at runtime via /__set_proxy_target
let proxyTarget = 'http://localhost:8080'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    {
      name: 'dynamic-proxy-target',
      configureServer(server) {
        server.middlewares.use('/__set_proxy_target', (req, res) => {
          let body = ''
          req.on('data', (chunk: Buffer) => { body += chunk.toString() })
          req.on('end', () => {
            try {
              const { target } = JSON.parse(body)
              if (target) {
                proxyTarget = target.replace(/\/+$/, '')
                console.log(`[proxy] target set to: ${proxyTarget}`)
              }
              res.writeHead(200, { 'Content-Type': 'application/json' })
              res.end(JSON.stringify({ ok: true, target: proxyTarget }))
            } catch {
              res.writeHead(400)
              res.end('invalid json')
            }
          })
        })
      },
    },
  ],
  server: {
    port: 34115,
    proxy: {
      '/api': {
        target: proxyTarget,
        changeOrigin: true,
        ws: true,
        rewrite: (path: string) => path.replace(/^\/api/, ''),
        configure: (proxy) => {
          // Dynamically route to the target set at runtime
          proxy.on('proxyReq', (proxyReq) => {
            try {
              const url = new URL(proxyTarget)
              proxyReq.setHeader('host', url.host)
            } catch { /* ignore */ }
          })
          proxy.on('error', (err, _req, res) => {
            console.error('[proxy] error:', err.message)
            if (isServerResponse(res)) {
              try {
                res.writeHead(502)
                res.end(`Proxy error: ${err.message}`)
              } catch { /* already sent */ }
            }
          })
        },
      },
    },
  },
  resolve: {
    alias: {
      '#': fileURLToPath(new URL('../../packages/ui/src', import.meta.url)),
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
})
