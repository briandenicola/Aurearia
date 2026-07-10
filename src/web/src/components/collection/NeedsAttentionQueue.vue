<template>
  <div class="card">
    <div class="mb-5 flex items-center justify-between gap-3">
      <h3 class="m-0 flex items-center gap-2 text-lg text-heading">
        <AlertCircle :size="20" />
        Needs Attention
      </h3>
      <div v-if="total > 0" class="rounded-full border border-border-subtle bg-[var(--accent-gold-glow)] px-[0.7rem] py-1 text-body text-text-secondary">{{ total }} coins</div>
    </div>

    <div v-if="loading" class="flex justify-center py-12">
      <div class="spinner"></div>
    </div>

    <div v-else-if="coins.length === 0" class="flex flex-col items-center gap-3 px-8 py-12 text-center text-text-secondary">
      <CircleCheck :size="32" />
      <p class="m-0 text-base">All coins are in good health</p>
    </div>

    <div v-else class="flex flex-col gap-3">
      <div class="flex flex-col gap-3">
        <div
          v-for="coin in coins"
          :key="coin.coinId"
          class="grid cursor-pointer grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-4 rounded-sm border border-border-subtle bg-input p-3 transition-all hover:border-border-accent hover:bg-card-hover hover:shadow-[var(--shadow-glow)] max-md:grid-cols-1 max-md:gap-3"
          @click="handleCoinClick(coin.coinId)"
        >
          <div class="flex flex-col gap-1">
            <router-link :to="`/coins/${coin.coinId}`" class="text-base font-semibold text-text-primary no-underline transition-colors hover:text-gold">
              {{ coin.title || `Coin #${coin.coinId}` }}
            </router-link>
          </div>

          <div class="flex flex-col items-end gap-1 max-md:w-full max-md:flex-row max-md:items-center max-md:justify-between">
            <div
              class="flex items-center gap-[0.35rem] rounded-full border px-[0.6rem] py-1 text-chip font-semibold"
              :class="coin.grade === 'A'
                ? 'border-[rgba(39,174,96,0.3)] bg-[rgba(39,174,96,0.15)] text-green-400'
                : coin.grade === 'B'
                  ? 'border-[rgba(52,152,219,0.3)] bg-[rgba(52,152,219,0.15)] text-sky-400'
                  : coin.grade === 'C'
                    ? 'border-[rgba(243,156,18,0.3)] bg-[rgba(243,156,18,0.15)] text-amber-400'
                    : coin.grade === 'D'
                      ? 'border-[rgba(230,126,34,0.3)] bg-[rgba(230,126,34,0.15)] text-orange-400'
                      : 'border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.15)] text-red-400'"
            >
              {{ coin.score }}
              <span class="text-label uppercase tracking-[0.05em]">{{ coin.grade }}</span>
            </div>
            <div class="text-sm text-text-muted">{{ coin.missingItems.length }} issues</div>
          </div>

          <div class="flex gap-[0.35rem] max-md:w-full max-md:justify-start">
            <button
              v-for="action in coin.quickActions.slice(0, 2)"
              :key="action"
              class="btn btn-xs btn-ghost"
              @click.stop="handleQuickAction(coin.coinId, action)"
            >
              {{ formatQuickAction(action) }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="total > coins.length" class="mt-4 flex items-center justify-between gap-3 border-t border-border-subtle pt-4 max-md:flex-col">
        <button
          class="btn btn-sm btn-secondary"
          :disabled="page === 1"
          @click="emit('pageChange', page - 1)"
        >
          <ChevronLeft :size="16" /> Previous
        </button>
        <span class="text-body text-text-secondary">
          Page {{ page }} of {{ Math.ceil(total / limit) }}
        </span>
        <button
          class="btn btn-sm btn-secondary"
          :disabled="page * limit >= total"
          @click="emit('pageChange', page + 1)"
        >
          Next <ChevronRight :size="16" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { AlertCircle, CircleCheck, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import type { CoinHealthItem, HealthQuickAction } from '@/types'

defineProps<{
  coins: CoinHealthItem[]
  loading: boolean
  total: number
  page: number
  limit: number
}>()

const emit = defineEmits<{
  quickAction: [coinId: number, action: HealthQuickAction]
  pageChange: [page: number]
}>()

const router = useRouter()

function handleCoinClick(coinId: number) {
  router.push(`/coins/${coinId}`)
}

function formatQuickAction(action: HealthQuickAction): string {
  const labels: Record<HealthQuickAction, string> = {
    edit_metadata: 'Edit',
    upload_images: 'Upload',
    run_valuation: 'Valuate',
    run_ai_analysis: 'Analyze',
  }
  return labels[action] || action
}

function handleQuickAction(coinId: number, action: HealthQuickAction) {
  emit('quickAction', coinId, action)
}
</script>
