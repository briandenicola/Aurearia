import { fileURLToPath, URL } from 'node:url'

import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'

// Builds the coin viewer MCP App into a single self-contained HTML file that
// the Go API embeds and serves as ui://aurearia/coin-viewer.html. MCP Apps
// hosts render it in a sandbox with no network access, so the script and
// stylesheet must be inlined. Run `npm run build:mcp-app` after changing
// src/mcp-apps/coin-viewer and commit the regenerated HTML; `task check:web`
// fails when it is stale.

const root = fileURLToPath(new URL('./src/mcp-apps/coin-viewer', import.meta.url))
const outDir = fileURLToPath(new URL('../api/handlers/mcpapps', import.meta.url))

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// Inline JavaScript must not contain sequences that end or confuse the
// surrounding <script> element; in minified output they only occur inside
// string literals, where the escaped forms are equivalent.
function escapeInlineScript(code: string) {
  return code.replace(/<\/script/gi, '<\\/script').replace(/<!--/g, '<\\!--')
}

function inlineIntoHtml(): Plugin {
  return {
    name: 'aurearia-inline-mcp-app',
    enforce: 'post',
    generateBundle(_options, bundle) {
      const page = Object.values(bundle).find((file) => file.type === 'asset' && file.fileName.endsWith('.html'))
      if (!page || page.type !== 'asset') return
      let html = String(page.source)
      for (const [key, file] of Object.entries(bundle)) {
        if (file.type === 'chunk') {
          const tag = new RegExp(`<script[^>]*src="[^"]*${escapeRegExp(file.fileName)}"[^>]*></script>`)
          html = html.replace(tag, () => `<script type="module">${escapeInlineScript(file.code)}</script>`)
          delete bundle[key]
        } else if (file.fileName.endsWith('.css')) {
          const tag = new RegExp(`<link[^>]*href="[^"]*${escapeRegExp(file.fileName)}"[^>]*>`)
          html = html.replace(tag, () => `<style>${String(file.source)}</style>`)
          delete bundle[key]
        }
      }
      if (/<script[^>]*\ssrc=|<link[^>]*stylesheet/i.test(html)) {
        this.error('coin viewer still references an external script or stylesheet')
      }
      page.source = html
    },
  }
}

export default defineConfig({
  root,
  base: './',
  publicDir: false,
  plugins: [vue(), inlineIntoHtml()],
  build: {
    outDir,
    // outDir holds Go sources; never clear it.
    emptyOutDir: false,
    modulePreload: false,
    cssCodeSplit: false,
    assetsInlineLimit: Number.MAX_SAFE_INTEGER,
    reportCompressedSize: false,
    rolldownOptions: {
      input: fileURLToPath(new URL('./src/mcp-apps/coin-viewer/coin-viewer.html', import.meta.url)),
      output: { codeSplitting: false },
    },
  },
})
