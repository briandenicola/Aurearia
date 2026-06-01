<template>
  <div class="ai-analysis-section">
    <div class="ai-analysis-content">
      <div class="ai-buttons">
        <button
          class="btn btn-primary btn-sm"
          :disabled="analyzing || !hasObverse || !aiAvailable"
          :title="!aiAvailable ? aiMessage : !hasObverse ? 'No obverse image' : ''"
          @click="handleAnalyze('obverse')"
        >
          {{ analyzingSide === 'obverse' ? 'Analyzing...' : 'Analyze Obverse' }}
        </button>
        <button
          class="btn btn-primary btn-sm"
          :disabled="analyzing || !hasReverse || !aiAvailable"
          :title="!aiAvailable ? aiMessage : !hasReverse ? 'No reverse image' : ''"
          @click="handleAnalyze('reverse')"
        >
          {{ analyzingSide === 'reverse' ? 'Analyzing...' : 'Analyze Reverse' }}
        </button>
      </div>
      <p v-if="!aiAvailable" class="ai-unavailable">{{ aiMessage || 'AI unavailable — configure a provider in Admin → AI Configuration' }}</p>

      <div v-if="obverseAnalysis" class="ai-result-section">
        <div class="ai-result-header">
          <h5 class="ai-result-heading">Obverse Analysis</h5>
          <button class="btn btn-ghost btn-xs" @click="handleDeleteAnalysis('obverse')">Remove</button>
        </div>
        <div class="ai-content" v-html="renderedObverse"></div>
      </div>

      <div v-if="reverseAnalysis" class="ai-result-section">
        <div class="ai-result-header">
          <h5 class="ai-result-heading">Reverse Analysis</h5>
          <button class="btn btn-ghost btn-xs" @click="handleDeleteAnalysis('reverse')">Remove</button>
        </div>
        <div class="ai-content" v-html="renderedReverse"></div>
      </div>

      <div v-if="aiAnalysis && !obverseAnalysis && !reverseAnalysis" class="ai-result-section">
        <div class="ai-content" v-html="renderedLegacy"></div>
      </div>

      <p v-if="!obverseAnalysis && !reverseAnalysis && !aiAnalysis && aiAvailable" class="ai-empty">
        Upload images and click an analyze button to get an expert assessment.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { analyzeCoin, deleteAnalysis, getAIStatus } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import MarkdownIt from 'markdown-it'
import DOMPurify from 'dompurify'

const props = defineProps<{
  coinId: number
  obverseAnalysis?: string | null
  reverseAnalysis?: string | null
  aiAnalysis?: string | null
  hasObverse: boolean
  hasReverse: boolean
}>()

const emit = defineEmits<{
  analysisUpdated: []
}>()

const { showConfirm, showAlert } = useDialog()
const md = new MarkdownIt({ html: false })

const analyzing = ref(false)
const analyzingSide = ref<string | null>(null)
const aiAvailable = ref(true)
const aiMessage = ref('')

const renderedObverse = computed(() => (props.obverseAnalysis ? DOMPurify.sanitize(md.render(props.obverseAnalysis)) : ''))
const renderedReverse = computed(() => (props.reverseAnalysis ? DOMPurify.sanitize(md.render(props.reverseAnalysis)) : ''))
const renderedLegacy = computed(() => (props.aiAnalysis ? DOMPurify.sanitize(md.render(props.aiAnalysis)) : ''))

onMounted(async () => {
  try {
    const res = await getAIStatus()
    aiAvailable.value = res.data.available
    aiMessage.value = res.data.message
  } catch {
    aiAvailable.value = false
    aiMessage.value = 'Unable to check AI status'
  }
})

async function handleAnalyze(side: 'obverse' | 'reverse') {
  analyzing.value = true
  analyzingSide.value = side
  try {
    await analyzeCoin(props.coinId, side)
    emit('analysisUpdated')
  } catch {
    await showAlert(`AI analysis failed for ${side}. Check the configured AI provider in Admin → AI Configuration.`, { title: 'Analysis Failed' })
  } finally {
    analyzing.value = false
    analyzingSide.value = null
  }
}

async function handleDeleteAnalysis(side: 'obverse' | 'reverse') {
  if (!await showConfirm(`Delete the ${side} analysis?`, { title: 'Delete Analysis', variant: 'danger' })) return
  try {
    await deleteAnalysis(props.coinId, side)
    emit('analysisUpdated')
  } catch {
    await showAlert(`Failed to delete ${side} analysis`, { title: 'Error' })
  }
}
</script>

<style scoped>
.ai-analysis-section {
  margin-bottom: 1.5rem;
}

.ai-analysis-content {
  padding: 0.75rem 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.ai-buttons {
  display: flex;
  gap: 0.4rem;
  margin-bottom: 0.75rem;
}

.ai-result-section {
  margin-bottom: 1.25rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-subtle);
}

.ai-result-section:last-of-type {
  border-bottom: none;
  margin-bottom: 0;
  padding-bottom: 0;
}

.ai-result-heading {
  font-size: 0.8rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--accent-gold);
  margin-bottom: 0.5rem;
}

.ai-result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ai-result-header .ai-result-heading {
  margin-bottom: 0;
}

.ai-content {
  font-size: 0.85rem;
  line-height: 1.7;
  color: var(--text-secondary);
}

.ai-content :deep(h1),
.ai-content :deep(h2),
.ai-content :deep(h3) {
  color: var(--accent-gold);
  margin-top: 1rem;
  margin-bottom: 0.5rem;
}

.ai-content :deep(strong) {
  color: var(--text-primary);
}

.ai-content :deep(ul),
.ai-content :deep(ol) {
  padding-left: 1.25rem;
}

.ai-empty {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-style: italic;
}

.ai-unavailable {
  font-size: 0.85rem;
  color: #e67e22;
  font-style: italic;
  margin-bottom: 0.5rem;
}
</style>
