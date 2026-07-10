<template>
  <section class="admin-section card flex flex-col gap-6">
    <h2 class="mb-0 border-b border-border-subtle pb-3 text-xl font-medium">AI Configuration</h2>
    <form class="flex flex-col gap-6" @submit.prevent="saveSettings">
      <div class="form-group">
        <label class="form-label">AI Provider</label>
        <div class="mt-2 grid gap-4 md:grid-cols-2">
          <label
            class="flex cursor-pointer flex-col gap-1 rounded-sm border-2 border-border-subtle bg-surface px-4 py-4 transition-colors hover:border-gold focus-within:outline-2 focus-within:outline-gold focus-within:outline-offset-2"
            :class="settings.AIProvider === 'anthropic' ? 'border-gold bg-[rgba(212,168,67,0.08)]' : ''"
          >
            <input v-model="settings.AIProvider" class="sr-only" type="radio" value="anthropic" />
            <span class="text-base font-semibold text-text-primary">Anthropic (Recommended)</span>
            <span class="text-chip text-text-secondary">Claude models with built-in web search</span>
          </label>
          <label
            class="flex cursor-pointer flex-col gap-1 rounded-sm border-2 border-border-subtle bg-surface px-4 py-4 transition-colors hover:border-gold focus-within:outline-2 focus-within:outline-gold focus-within:outline-offset-2"
            :class="settings.AIProvider === 'ollama' ? 'border-gold bg-[rgba(212,168,67,0.08)]' : ''"
          >
            <input v-model="settings.AIProvider" class="sr-only" type="radio" value="ollama" />
            <span class="text-base font-semibold text-text-primary">Ollama</span>
            <span class="text-chip text-text-secondary">Self-hosted models. Requires SearXNG for web search.</span>
          </label>
        </div>
        <p
          v-if="!settings.AIProvider"
          class="mt-2 rounded-sm border border-[rgba(231,176,60,0.3)] bg-[rgba(231,176,60,0.1)] px-3 py-2 text-body text-warning"
        >
          Please select an AI provider to enable agent features.
        </p>
      </div>

      <template v-if="settings.AIProvider === 'anthropic'">
        <div class="form-group">
          <label class="form-label">Anthropic API Key</label>
          <input v-model="settings.AnthropicAPIKey" class="form-input" type="password" placeholder="Enter your Anthropic API key" />
          <span class="mt-1 block text-sm text-text-muted">Get a key at <a class="text-gold hover:underline" href="https://console.anthropic.com/" target="_blank" rel="noopener">console.anthropic.com</a></span>
        </div>
        <div class="form-group">
          <label class="form-label">Anthropic Model</label>
          <select v-model="settings.AnthropicModel" class="form-input">
            <option v-for="m in anthropicModels" :key="m.id" :value="m.id">{{ m.name }}</option>
          </select>
        </div>
        <div class="mb-2 mt-3 flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="anthropicTesting" @click="testAnthropicConn">
            {{ anthropicTesting ? 'Testing...' : 'Test Anthropic API' }}
          </button>
          <div
            v-if="anthropicTestResult"
            class="flex items-center gap-2 rounded-sm border px-3 py-[0.6rem] text-body"
            :class="anthropicTestOk ? 'border-[rgba(46,204,113,0.3)] bg-[rgba(46,204,113,0.1)] text-[var(--color-positive)]' : 'border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.1)] text-[var(--color-negative)]'"
          >
            <span class="text-label">{{ anthropicTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
            {{ anthropicTestResult }}
          </div>
        </div>
      </template>

      <template v-if="settings.AIProvider === 'ollama'">
        <div class="form-group">
          <label class="form-label">Ollama URL</label>
          <input v-model="settings.OllamaURL" class="form-input" placeholder="http://localhost:11434" />
        </div>
        <div class="form-group">
          <label class="form-label">Vision Model</label>
          <input v-model="settings.OllamaModel" class="form-input" placeholder="llava" />
          <span class="mt-1 block text-sm text-text-muted">e.g. llava, llama3.2-vision, bakllava</span>
        </div>
        <div class="form-group">
          <label class="form-label">Request Timeout (seconds)</label>
          <input v-model="settings.OllamaTimeout" class="form-input" type="number" min="10" max="1800" step="10" />
          <span class="mt-1 block text-sm text-text-muted">Time limit for AI analysis calls. Default: 300 (5 minutes)</span>
        </div>
        <div class="form-group">
          <label class="form-label">SearXNG URL</label>
          <input v-model="settings.SearXNGURL" class="form-input" placeholder="http://localhost:8888" />
          <span class="mt-1 block text-sm text-text-muted">Required for web search features (coin search, coin shows, valuations).</span>
          <p
            v-if="settings.AIProvider === 'ollama' && !settings.SearXNGURL"
            class="mt-2 rounded-sm border border-[rgba(231,176,60,0.3)] bg-[rgba(231,176,60,0.1)] px-3 py-2 text-body text-warning"
          >
            Web search features require a SearXNG instance. Configure the URL or switch to Anthropic.
          </p>
        </div>
        <div class="mb-2 mt-3 flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="ollamaTesting" @click="testOllamaConnection">
            {{ ollamaTesting ? 'Testing...' : 'Test Ollama' }}
          </button>
          <button v-if="settings.SearXNGURL" type="button" class="btn btn-secondary btn-sm" :disabled="searxngTesting" @click="testSearxngConn">
            {{ searxngTesting ? 'Testing...' : 'Test SearXNG' }}
          </button>
        </div>
        <div
          v-if="ollamaTestResult"
          class="flex items-center gap-2 rounded-sm border px-3 py-[0.6rem] text-body"
          :class="ollamaTestOk ? 'border-[rgba(46,204,113,0.3)] bg-[rgba(46,204,113,0.1)] text-[var(--color-positive)]' : 'border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.1)] text-[var(--color-negative)]'"
        >
          <span class="text-label">{{ ollamaTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
          {{ ollamaTestResult }}
        </div>
        <div
          v-if="searxngTestResult"
          class="mt-3 flex items-center gap-2 rounded-sm border px-3 py-[0.6rem] text-body"
          :class="searxngTestOk ? 'border-[rgba(46,204,113,0.3)] bg-[rgba(46,204,113,0.1)] text-[var(--color-positive)]' : 'border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.1)] text-[var(--color-negative)]'"
        >
          <span class="text-label">{{ searxngTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
          {{ searxngTestResult }}
        </div>
      </template>

      <template v-if="settings.AIProvider">
        <p class="mt-1 block text-sm text-text-muted">
          Provider tests validate the selected AI provider only. Agent chat and image analysis also require the internal agent service to be configured and running.
        </p>
        <hr class="my-6 border-0 border-t border-border-subtle" />
        <h3 class="mb-4 text-base font-semibold text-text-primary">Agent Prompts</h3>
        <div class="form-group">
          <div class="mb-1 flex items-center justify-between gap-2">
            <label class="form-label mb-0">Coin Search Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.CoinSearchPrompt === coinSearchPromptDefault"
              @click="settings.CoinSearchPrompt = coinSearchPromptDefault"
            >
              Revert to Default
            </button>
          </div>
          <textarea v-model="settings.CoinSearchPrompt" class="form-textarea w-full" rows="8" />
          <span class="mt-1 block text-sm text-text-muted">Search instructions for the coin search agent (Team 1). Controls which dealer sites to search, availability rules, and search strategy.</span>
        </div>
        <div class="form-group">
          <div class="mb-1 flex items-center justify-between gap-2">
            <label class="form-label mb-0">Coin Shows Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.CoinShowsPrompt === coinShowsPromptDefault"
              @click="settings.CoinShowsPrompt = coinShowsPromptDefault"
            >
              Revert to Default
            </button>
          </div>
          <textarea v-model="settings.CoinShowsPrompt" class="form-textarea w-full" rows="8" />
          <span class="mt-1 block text-sm text-text-muted">Search instructions for the coin shows agent (Team 2). Controls which show directories and organizations to search.</span>
        </div>
        <div class="form-group">
          <div class="mb-1 flex items-center justify-between gap-2">
            <label class="form-label mb-0">Value Estimator Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.ValuationPrompt === valuationPromptDefault"
              @click="settings.ValuationPrompt = valuationPromptDefault"
            >
              Revert to Default
            </button>
          </div>
          <textarea v-model="settings.ValuationPrompt" class="form-textarea w-full" rows="8" />
          <span class="mt-1 block text-sm text-text-muted">System prompt for the AI value estimator. Controls how it researches and estimates coin values.</span>
        </div>
        <h3 class="mb-4 text-base font-semibold text-text-primary">Analysis Prompts</h3>
        <div class="form-group">
          <div class="mb-1 flex items-center justify-between gap-2">
            <label class="form-label mb-0">Obverse Analysis Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.ObversePrompt === settingDefaults.ObversePrompt"
              @click="settings.ObversePrompt = settingDefaults.ObversePrompt"
            >
              Revert to Default
            </button>
          </div>
          <textarea v-model="settings.ObversePrompt" class="form-textarea w-full" rows="6" />
          <span class="mt-1 block text-sm text-text-muted">Prompt for obverse image analysis. Coin context is appended automatically.</span>
        </div>
        <div class="form-group">
          <div class="mb-1 flex items-center justify-between gap-2">
            <label class="form-label mb-0">Reverse Analysis Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.ReversePrompt === settingDefaults.ReversePrompt"
              @click="settings.ReversePrompt = settingDefaults.ReversePrompt"
            >
              Revert to Default
            </button>
          </div>
          <textarea v-model="settings.ReversePrompt" class="form-textarea w-full" rows="6" />
          <span class="mt-1 block text-sm text-text-muted">Prompt for reverse image analysis. Coin context is appended automatically.</span>
        </div>
        <div class="form-group">
          <div class="mb-1 flex items-center justify-between gap-2">
            <label class="form-label mb-0">Text Extraction Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.TextExtractionPrompt === settingDefaults.TextExtractionPrompt"
              @click="settings.TextExtractionPrompt = settingDefaults.TextExtractionPrompt"
            >
              Revert to Default
            </button>
          </div>
          <textarea v-model="settings.TextExtractionPrompt" class="form-textarea w-full" rows="6" />
          <span class="mt-1 block text-sm text-text-muted">Prompt for extracting text from store card images.</span>
        </div>
      </template>

      <p v-if="settingsMsg" class="my-2 text-body text-gold" :class="settingsError ? 'text-[var(--color-negative)]' : ''">{{ settingsMsg }}</p>
      <div class="flex items-center gap-2">
        <button type="submit" class="btn btn-primary btn-sm" :disabled="settingsSaving">
          {{ settingsSaving ? 'Saving...' : 'Save AI Settings' }}
        </button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import type { AppSettings } from '@/types'
import type { AnthropicModel } from '@/api/client'

defineProps<{
  settings: AppSettings
  settingDefaults: AppSettings
  settingsMsg: string
  settingsError: boolean
  settingsSaving: boolean
  anthropicModels: AnthropicModel[]
  anthropicTesting: boolean
  anthropicTestResult: string
  anthropicTestOk: boolean
  ollamaTesting: boolean
  ollamaTestResult: string
  ollamaTestOk: boolean
  searxngTesting: boolean
  searxngTestResult: string
  searxngTestOk: boolean
  coinSearchPromptDefault: string
  coinShowsPromptDefault: string
  valuationPromptDefault: string
}>()

const emit = defineEmits<{
  save: []
  testAnthropicConn: []
  testOllamaConnection: []
  testSearxngConn: []
}>()

function saveSettings() {
  emit('save')
}
function testAnthropicConn() {
  emit('testAnthropicConn')
}
function testOllamaConnection() {
  emit('testOllamaConnection')
}
function testSearxngConn() {
  emit('testSearxngConn')
}
</script>
