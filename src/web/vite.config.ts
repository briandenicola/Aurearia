import { existsSync, readFileSync } from 'node:fs'
import { execSync } from 'node:child_process'
import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'

const repoRoot = fileURLToPath(new URL('../..', import.meta.url))
const versionFile = new URL('../../VERSION', import.meta.url)

function readBaseVersion() {
  if (!existsSync(versionFile)) return 'dev'
  return readFileSync(versionFile, 'utf8').trim() || 'dev'
}

function readCommitSha() {
  try {
    return execSync('git rev-parse --short HEAD', { cwd: repoRoot, encoding: 'utf8' }).trim()
  } catch {
    return ''
  }
}

const baseVersion = readBaseVersion()
const commitSha = readCommitSha()

function buildAppVersion(rawVersion: string | undefined) {
  const version = rawVersion?.trim()
  if (!version) {
    return commitSha ? `${baseVersion}.${commitSha}` : baseVersion
  }
  if (/^[a-f0-9]{7,40}$/i.test(version)) {
    return `${baseVersion}.${version.substring(0, 7)}`
  }
  return version
}

const appVersion = buildAppVersion(process.env.VITE_APP_VERSION)

export default defineConfig({
  define: {
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(appVersion),
  },
  plugins: [
    tailwindcss(),
    vue(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['coin-logo.jpg'],
      manifest: {
        name: 'Aurearia - Coin Collection',
        short_name: 'Aurearia',
        description: 'Track and manage your coin collection',
        theme_color: '#1a1a2e',
        background_color: '#0f0f1a',
        display: 'standalone',
        scope: '/',
        start_url: '/',
        icons: [
          {
            src: 'pwa-192x192.png',
            sizes: '192x192',
            type: 'image/png',
          },
          {
            src: 'pwa-512x512.png',
            sizes: '512x512',
            type: 'image/png',
          },
          {
            src: 'pwa-512x512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'any maskable',
          },
        ],
      },
      workbox: {
        inlineWorkboxRuntime: true,
        globPatterns: ['**/*.{js,css,html,ico,png,svg,jpg,woff2}'],
        navigateFallbackDenylist: [/^\/api/, /^\/uploads/, /^\/sw\.js/],
        cleanupOutdatedCaches: true,
        skipWaiting: true,
        clientsClaim: true,
        runtimeCaching: [
          {
            urlPattern: /^https?:\/\/.*\/api\//,
            handler: 'NetworkOnly',
            method: 'PUT',
          },
          {
            urlPattern: /^https?:\/\/.*\/api\//,
            handler: 'NetworkOnly',
            method: 'POST',
          },
          {
            urlPattern: /^https?:\/\/.*\/api\//,
            handler: 'NetworkOnly',
            method: 'DELETE',
          },
          {
            // Cache only public showcase reads; all authenticated API reads stay network-only.
            urlPattern: /^https?:\/\/.*\/api\/showcase(?:\/.*)?$/,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'showcase-api-cache',
              expiration: {
                maxEntries: 50,
                maxAgeSeconds: 300,
              },
              cacheableResponse: {
                statuses: [0, 200],
              },
            },
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
