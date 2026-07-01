<template>
  <section class="admin-section card">
    <h2>AI Configuration</h2>
    <form @submit.prevent="saveSettings">
      <!-- Provider Selection -->
      <div class="form-group">
        <label class="form-label">AI Provider</label>
        <div class="provider-toggle">
          <label class="provider-option" :class="{ active: settings.AIProvider === 'anthropic' }">
            <input type="radio" v-model="settings.AIProvider" value="anthropic" />
            <span class="provider-label">Anthropic (Recommended)</span>
            <span class="provider-desc">Claude models with built-in web search</span>
          </label>
          <label class="provider-option" :class="{ active: settings.AIProvider === 'ollama' }">
            <input type="radio" v-model="settings.AIProvider" value="ollama" />
            <span class="provider-label">Ollama</span>
            <span class="provider-desc">Self-hosted models. Requires SearXNG for web search.</span>
          </label>
        </div>
        <p v-if="!settings.AIProvider" class="provider-warning">
          Please select an AI provider to enable agent features.
        </p>
      </div>

      <!-- Anthropic Settings -->
      <template v-if="settings.AIProvider === 'anthropic'">
        <div class="form-group">
          <label class="form-label">Anthropic API Key</label>
          <input v-model="settings.AnthropicAPIKey" class="form-input" type="password" placeholder="Enter your Anthropic API key" />
          <span class="form-hint">Get a key at <a href="https://console.anthropic.com/" target="_blank" rel="noopener">console.anthropic.com</a></span>
        </div>
        <div class="form-group">
          <label class="form-label">Anthropic Model</label>
          <select v-model="settings.AnthropicModel" class="form-input">
            <option v-for="m in anthropicModels" :key="m.id" :value="m.id">{{ m.name }}</option>
          </select>
        </div>
        <div class="connectivity-actions">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="anthropicTesting" @click="testAnthropicConn">
            {{ anthropicTesting ? 'Testing...' : 'Test Anthropic API' }}
          </button>
          <div v-if="anthropicTestResult" class="connectivity-result" :class="{ success: anthropicTestOk, error: !anthropicTestOk }">
            <span class="connectivity-icon">{{ anthropicTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
            {{ anthropicTestResult }}
          </div>
        </div>
      </template>

      <!-- Ollama Settings -->
      <template v-if="settings.AIProvider === 'ollama'">
        <div class="form-group">
          <label class="form-label">Ollama URL</label>
          <input v-model="settings.OllamaURL" class="form-input" placeholder="http://localhost:11434" />
        </div>
        <div class="form-group">
          <label class="form-label">Vision Model</label>
          <input v-model="settings.OllamaModel" class="form-input" placeholder="llava" />
          <span class="form-hint">e.g. llava, llama3.2-vision, bakllava</span>
        </div>
        <div class="form-group">
          <label class="form-label">Request Timeout (seconds)</label>
          <input v-model="settings.OllamaTimeout" class="form-input" type="number" min="10" max="1800" step="10" />
          <span class="form-hint">Time limit for AI analysis calls. Default: 300 (5 minutes)</span>
        </div>
        <div class="form-group">
          <label class="form-label">SearXNG URL</label>
          <input v-model="settings.SearXNGURL" class="form-input" placeholder="http://localhost:8888" />
          <span class="form-hint">Required for web search features (coin search, coin shows, valuations).</span>
          <p v-if="settings.AIProvider === 'ollama' && !settings.SearXNGURL" class="provider-warning">
            Web search features require a SearXNG instance. Configure the URL or switch to Anthropic.
          </p>
        </div>
        <div class="connectivity-actions">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="ollamaTesting" @click="testOllamaConnection">
            {{ ollamaTesting ? 'Testing...' : 'Test Ollama' }}
          </button>
          <button v-if="settings.SearXNGURL" type="button" class="btn btn-secondary btn-sm" :disabled="searxngTesting" @click="testSearxngConn">
            {{ searxngTesting ? 'Testing...' : 'Test SearXNG' }}
          </button>
        </div>
        <div v-if="ollamaTestResult" class="connectivity-result" :class="{ success: ollamaTestOk, error: !ollamaTestOk }">
          <span class="connectivity-icon">{{ ollamaTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
          {{ ollamaTestResult }}
        </div>
        <div v-if="searxngTestResult" class="connectivity-result" :class="{ success: searxngTestOk, error: !searxngTestOk }">
          <span class="connectivity-icon">{{ searxngTestOk ? '&#x25CF;' : '&#x25CF;' }}</span>
          {{ searxngTestResult }}
        </div>
      </template>

      <!-- Shared Prompt Settings (visible when a provider is selected) -->
      <template v-if="settings.AIProvider">
        <p class="form-hint">
          Provider tests validate the selected AI provider only. Agent chat and image analysis also require the internal agent service to be configured and running.
        </p>
        <hr class="section-divider" />
        <h3 class="subsection-title">Agent Prompts</h3>
        <div class="form-group">
          <div class="prompt-header">
            <label class="form-label">Coin Search Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.CoinSearchPrompt === coinSearchPromptDefault"
              @click="settings.CoinSearchPrompt = coinSearchPromptDefault"
            >
              Revert to Default
            </button>
          </div>
          <textarea
            v-model="settings.CoinSearchPrompt"
            class="form-textarea"
            rows="8"
          />
          <span class="form-hint">Search instructions for the coin search agent (Team 1). Controls which dealer sites to search, availability rules, and search strategy.</span>
        </div>
        <div class="form-group">
          <div class="prompt-header">
            <label class="form-label">Coin Shows Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.CoinShowsPrompt === coinShowsPromptDefault"
              @click="settings.CoinShowsPrompt = coinShowsPromptDefault"
            >
              Revert to Default
            </button>
          </div>
          <textarea
            v-model="settings.CoinShowsPrompt"
            class="form-textarea"
            rows="8"
          />
          <span class="form-hint">Search instructions for the coin shows agent (Team 2). Controls which show directories and organizations to search.</span>
        </div>
        <div class="form-group">
          <div class="prompt-header">
            <label class="form-label">Value Estimator Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.ValuationPrompt === valuationPromptDefault"
              @click="settings.ValuationPrompt = valuationPromptDefault"
            >
              Revert to Default
            </button>
          </div>
          <textarea
            v-model="settings.ValuationPrompt"
            class="form-textarea"
            rows="8"
          />
          <span class="form-hint">System prompt for the AI value estimator. Controls how it researches and estimates coin values.</span>
        </div>
        <h3 class="subsection-title">Analysis Prompts</h3>
        <div class="form-group">
          <div class="prompt-header">
            <label class="form-label">Obverse Analysis Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.ObversePrompt === settingDefaults.ObversePrompt"
              @click="settings.ObversePrompt = settingDefaults.ObversePrompt"
            >
              Revert to Default
            </button>
          </div>
          <textarea
            v-model="settings.ObversePrompt"
            class="form-textarea"
            rows="6"
          />
          <span class="form-hint">Prompt for obverse image analysis. Coin context is appended automatically.</span>
        </div>
        <div class="form-group">
          <div class="prompt-header">
            <label class="form-label">Reverse Analysis Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.ReversePrompt === settingDefaults.ReversePrompt"
              @click="settings.ReversePrompt = settingDefaults.ReversePrompt"
            >
              Revert to Default
            </button>
          </div>
          <textarea
            v-model="settings.ReversePrompt"
            class="form-textarea"
            rows="6"
          />
          <span class="form-hint">Prompt for reverse image analysis. Coin context is appended automatically.</span>
        </div>
        <div class="form-group">
          <div class="prompt-header">
            <label class="form-label">Text Extraction Prompt</label>
            <button
              type="button"
              class="btn btn-ghost btn-xs"
              :disabled="settings.TextExtractionPrompt === settingDefaults.TextExtractionPrompt"
              @click="settings.TextExtractionPrompt = settingDefaults.TextExtractionPrompt"
            >
              Revert to Default
            </button>
          </div>
          <textarea
            v-model="settings.TextExtractionPrompt"
            class="form-textarea"
            rows="6"
          />
          <span class="form-hint">Prompt for extracting text from store card images.</span>
        </div>
      </template>

      <p v-if="settingsMsg" class="msg" :class="{ error: settingsError }">{{ settingsMsg }}</p>
      <div class="ai-actions">
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

<style scoped>
.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: var(--color-negative);
}

.ai-actions {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.prompt-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.25rem;
}

.prompt-header .form-label {
  margin-bottom: 0;
}

.connectivity-result {
  margin-top: 0.75rem;
  padding: 0.6rem 0.8rem;
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.connectivity-result.success {
  background: rgba(46, 204, 113, 0.1);
  border: 1px solid rgba(46, 204, 113, 0.3);
  color: var(--color-positive);
}

.connectivity-result.error {
  background: rgba(231, 76, 60, 0.1);
  border: 1px solid rgba(231, 76, 60, 0.3);
  color: var(--color-negative);
}

.connectivity-icon {
  font-size: 0.7rem;
}

.connectivity-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  margin-bottom: 0.5rem;
}

/* Provider Toggle */
.provider-toggle {
  display: flex;
  gap: 1rem;
  margin-top: 0.5rem;
}

.provider-option {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 1rem;
  border: 2px solid var(--border-subtle, #333);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s;
}

.provider-option:hover {
  border-color: var(--accent-gold, #d4a843);
}

.provider-option.active {
  border-color: var(--accent-gold, #d4a843);
  background: rgba(212, 168, 67, 0.08);
}

.provider-option input[type="radio"] {
  display: none;
}

.provider-label {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--text-primary, #e0e0e0);
}

.provider-desc {
  font-size: 0.8rem;
  color: var(--text-secondary, #999);
}

.provider-warning {
  margin-top: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: rgba(231, 176, 60, 0.1);
  border: 1px solid rgba(231, 176, 60, 0.3);
  border-radius: var(--radius-sm);
  color: #e7b03c;
  font-size: 0.85rem;
}

.section-divider {
  border: none;
  border-top: 1px solid var(--border-subtle, #333);
  margin: 1.5rem 0;
}

.subsection-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--text-primary, #e0e0e0);
}

.form-hint {
  display: block;
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}
</style>
