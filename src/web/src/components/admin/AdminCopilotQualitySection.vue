<template>
  <section class="admin-section card flex flex-col gap-6">
    <div class="flex flex-col gap-2 border-b border-border-subtle pb-3">
      <h2 class="text-xl font-medium text-heading">Copilot Quality</h2>
      <p class="text-base leading-relaxed text-text-secondary">
        Rubric grades for completed Coin Copilot conversations. Grading runs nightly when enabled under Schedules.
        Mark a grade wrong to leave it out of the trend.
      </p>
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <label class="form-label" for="copilot-quality-days">Window</label>
      <select id="copilot-quality-days" v-model.number="days" class="form-input w-auto" @change="loadSummary()">
        <option :value="7">7 days</option>
        <option :value="30">30 days</option>
        <option :value="90">90 days</option>
      </select>
    </div>

    <div v-if="summaryError" class="rounded-sm border border-status-error-border bg-status-error-bg p-4 text-base text-loss" role="alert">
      {{ summaryError }}
    </div>
    <div v-else-if="summary && summary.graded === 0" class="text-base text-text-muted">
      No graded conversations in the last {{ summary.days }} days.
    </div>
    <div v-else-if="summary" class="grid gap-4 md:grid-cols-3">
      <div class="rounded-md border border-border-subtle bg-card p-5">
        <div class="section-label">Average quality</div>
        <div class="font-display text-2xl font-bold text-gold">{{ percent(summary.avgQuality) }}</div>
        <div class="text-base text-text-secondary">
          {{ summary.graded }} graded<span v-if="summary.disputed">, {{ summary.disputed }} disputed and excluded</span>
        </div>
      </div>

      <div class="rounded-md border border-border-subtle bg-card p-5 md:col-span-2">
        <div class="section-label mb-2">Daily average</div>
        <div class="flex h-24 items-end gap-1" role="img" :aria-label="trendLabel">
          <div
            v-for="day in summary.trend"
            :key="day.day"
            class="flex-1 rounded-t-sm bg-gold"
            :style="{ height: `${Math.max(4, day.avgQuality * 100)}%` }"
            :title="`${day.day}: ${percent(day.avgQuality)} across ${day.graded}`"
          ></div>
        </div>
      </div>

      <div class="rounded-md border border-border-subtle bg-card p-5 md:col-span-3">
        <div class="section-label mb-3">Share of answers failing each check</div>
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="criterion in criteria" :key="criterion.id" class="flex flex-col gap-1">
            <div class="flex items-baseline justify-between gap-2 text-base">
              <span>{{ criterion.label }}</span>
              <span class="font-semibold" :class="failRate(criterion.id) > 0.2 ? 'text-loss' : 'text-text-secondary'">
                {{ criterion.id in summary.failRates ? percent(failRate(criterion.id)) : 'n/a' }}
              </span>
            </div>
            <div class="h-1.5 rounded-full bg-border-subtle">
              <div class="h-1.5 rounded-full bg-loss" :style="{ width: `${failRate(criterion.id) * 100}%` }"></div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="flex flex-col gap-3">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h3 class="text-lg font-medium text-heading">Graded conversations</h3>
        <div class="flex flex-wrap gap-2">
          <select v-model="filter" class="form-input w-auto" aria-label="Filter grades" @change="reload()">
            <option value="undisputed">Undisputed</option>
            <option value="disputed">Marked wrong</option>
            <option value="all">All</option>
          </select>
          <select v-model="sort" class="form-input w-auto" aria-label="Sort grades" @change="reload()">
            <option value="worst">Lowest score first</option>
            <option value="recent">Most recent first</option>
          </select>
        </div>
      </div>

      <div v-if="listLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
      <div v-else-if="listError" class="text-base text-loss" role="alert">{{ listError }}</div>
      <div v-else-if="grades.length === 0" class="py-8 text-center text-text-muted">No graded conversations to show.</div>
      <ul v-else class="flex flex-col gap-3">
        <li v-for="grade in grades" :key="grade.id" class="rounded-md border border-border-subtle bg-card p-4">
          <button
            class="flex w-full items-start justify-between gap-3 text-left"
            :aria-expanded="expanded === grade.id"
            @click="expanded = expanded === grade.id ? null : grade.id"
          >
            <span class="flex min-w-0 flex-col gap-1">
              <span class="truncate font-medium">{{ grade.question }}</span>
              <span class="text-chip text-text-muted">
                {{ formatDate(grade.gradedAt) }} · {{ failedChecks(grade).length ? failedChecks(grade).join(', ') : 'All checks passed' }}
              </span>
            </span>
            <span class="flex shrink-0 items-center gap-2">
              <BaseStatusBadge v-if="grade.disputed" tone="warning">Marked wrong</BaseStatusBadge>
              <span class="font-display text-lg font-bold" :class="grade.quality < 0.6 ? 'text-loss' : 'text-gold'">{{ percent(grade.quality) }}</span>
            </span>
          </button>

          <div v-if="expanded === grade.id" class="mt-4 flex flex-col gap-4 border-t border-border-subtle pt-4">
            <div>
              <div class="section-label mb-1">Question</div>
              <p class="whitespace-pre-wrap text-base">{{ grade.question }}</p>
            </div>
            <div>
              <div class="section-label mb-1">Answer</div>
              <p class="max-h-80 overflow-y-auto whitespace-pre-wrap text-base text-text-secondary">{{ grade.answer }}</p>
            </div>
            <div>
              <div class="section-label mb-1">Judge ({{ grade.judgeModel }})</div>
              <p class="whitespace-pre-wrap text-base text-text-secondary">{{ grade.explanation }}</p>
            </div>
            <div class="flex flex-col gap-2">
              <template v-if="grade.disputed">
                <p class="text-base">
                  Marked wrong {{ grade.disputedAt ? formatDate(grade.disputedAt) : '' }}<span v-if="grade.disputeNote">: {{ grade.disputeNote }}</span>
                </p>
                <button class="btn btn-secondary btn-sm self-start" :disabled="saving === grade.id" @click="setDispute(grade, false)">
                  Restore grade
                </button>
              </template>
              <template v-else>
                <label class="form-label" :for="`dispute-note-${grade.id}`">Why is this grade wrong? (optional)</label>
                <textarea :id="`dispute-note-${grade.id}`" v-model="notes[grade.id]" class="form-input w-full" rows="2" maxlength="500"></textarea>
                <button class="btn btn-secondary btn-sm self-start" :disabled="saving === grade.id" @click="setDispute(grade, true)">
                  Mark grade wrong
                </button>
              </template>
            </div>
          </div>
        </li>
      </ul>

      <div v-if="total > limit" class="flex items-center justify-center gap-3">
        <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="changePage(-1)">Prev</button>
        <span class="text-chip text-text-secondary">Page {{ page }} of {{ pageCount }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="page >= pageCount" @click="changePage(1)">Next</button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { BaseStatusBadge } from '@/components/ui'
import { disputeCopilotGrade, getCopilotGradeSummary, getCopilotGrades } from '@/api/client'
import type { CopilotGrade, CopilotGradeSummary } from '@/types'

const criteria = [
  { id: 'answersDirectly', label: 'Answers directly' },
  { id: 'grounded', label: 'Grounded in tool results' },
  { id: 'accurate', label: 'Numismatically accurate' },
  { id: 'citesSources', label: 'Cites sources' },
  { id: 'noOfferMenu', label: 'No menu of offers' },
  { id: 'noFalseInability', label: 'No false "can\'t check"' },
] as const

const limit = 20
const days = ref(30)
const summary = ref<CopilotGradeSummary | null>(null)
const summaryError = ref('')
const grades = ref<CopilotGrade[]>([])
const total = ref(0)
const page = ref(1)
const sort = ref<'worst' | 'recent'>('worst')
const filter = ref<'undisputed' | 'disputed' | 'all'>('undisputed')
const listLoading = ref(false)
const listError = ref('')
const expanded = ref<number | null>(null)
const saving = ref<number | null>(null)
const notes = reactive<Record<number, string>>({})

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / limit)))
const trendLabel = computed(() =>
  (summary.value?.trend ?? []).map(day => `${day.day} ${percent(day.avgQuality)}`).join(', '),
)

function percent(value: number) {
  return `${Math.round(value * 100)}%`
}

function failRate(id: string) {
  return summary.value?.failRates[id] ?? 0
}

function failedChecks(grade: CopilotGrade) {
  const failed: string[] = []
  if (!grade.answersDirectly) failed.push('not direct')
  if (!grade.grounded) failed.push('ungrounded')
  if (!grade.accurate) failed.push('inaccurate')
  if (grade.citesSources === 'no') failed.push('no sources')
  if (!grade.noOfferMenu) failed.push('offer menu')
  if (!grade.noFalseInability) failed.push('false inability')
  return failed
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString()
}

async function loadSummary() {
  summaryError.value = ''
  try {
    const res = await getCopilotGradeSummary(days.value)
    summary.value = res.data
  } catch {
    summary.value = null
    summaryError.value = 'Failed to load the Copilot quality summary.'
  }
}

async function loadGrades() {
  listLoading.value = true
  listError.value = ''
  try {
    const disputed = filter.value === 'all' ? undefined : filter.value === 'disputed'
    const res = await getCopilotGrades({ page: page.value, limit, sort: sort.value, disputed })
    grades.value = res.data.grades
    total.value = res.data.total
  } catch {
    grades.value = []
    listError.value = 'Failed to load graded conversations.'
  } finally {
    listLoading.value = false
  }
}

function reload() {
  page.value = 1
  expanded.value = null
  loadGrades()
}

function changePage(delta: number) {
  page.value = Math.min(pageCount.value, Math.max(1, page.value + delta))
  expanded.value = null
  loadGrades()
}

async function setDispute(grade: CopilotGrade, disputed: boolean) {
  saving.value = grade.id
  try {
    const res = await disputeCopilotGrade(grade.id, disputed, disputed ? (notes[grade.id] ?? '') : '')
    Object.assign(grade, res.data)
    await loadSummary()
  } catch {
    listError.value = 'Failed to update the grade.'
  } finally {
    saving.value = null
  }
}

onMounted(() => {
  loadSummary()
  loadGrades()
})
</script>
