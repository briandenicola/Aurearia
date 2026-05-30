<template>
  <div class="container">
    <div v-if="store.loading && !coin" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="coin" class="coin-detail">
      <div class="sticky-action-bar">
        <CoinDetailHeaderActions
          :is-wishlist="coin.isWishlist"
          :is-sold="coin.isSold"
          :coin-id="coin.id"
          @purchase="showPurchaseModal = true"
          @sell="showSellModal = true"
          @delete="handleDelete"
        />
      </div>

      <div class="detail-layout">
        <!-- Images -->
        <div class="detail-images">
          <ImageGallery :images="coin.images || []" :processing="removingBg" @remove-bg="handleRemoveBackground" @delete-image="handleDeleteImage" />
        </div>

        <!-- Info -->
        <div class="detail-info">
          <div class="detail-title-section">
            <h1>{{ coin.name }}</h1>
            <p v-if="coin.ruler" class="detail-ruler">{{ coin.ruler }}</p>
          </div>

          <div v-if="coin.obverseInscription || coin.reverseInscription" class="inscriptions-section">
            <h3>Inscriptions</h3>
            <div v-if="coin.obverseInscription" class="inscription">
              <span class="inscription-label">Obverse:</span>
              <span class="inscription-text">{{ coin.obverseInscription }}</span>
            </div>
            <div v-if="coin.reverseInscription" class="inscription">
              <span class="inscription-label">Reverse:</span>
              <span class="inscription-text">{{ coin.reverseInscription }}</span>
            </div>
          </div>

          <CoinTagsSection
            :tags="coin.tags ?? []"
            :category="coin.category"
            :coin-id="coin.id"
            @tags-changed="refreshCoin"
          />

          <div v-if="coin.purchaseDate || coin.purchaseLocation || safeReferenceUrl" class="purchase-meta">
            <span v-if="coin.purchaseDate">Purchased {{ new Date(coin.purchaseDate).toLocaleDateString() }}</span>
            <template v-if="coin.purchaseLocation">
              <span>{{ coin.purchaseDate ? ' from ' : 'Purchased from ' }}</span>
              <SafeExternalLink v-if="safeReferenceUrl" :href="safeReferenceUrl" target="_blank" rel="noopener" class="store-link">
                {{ coin.purchaseLocation }} ↗
              </SafeExternalLink>
              <span v-else>{{ coin.purchaseLocation }}</span>
            </template>
            <template v-if="safeReferenceUrl && !coin.purchaseLocation">
              <span v-if="coin.purchaseDate"> · </span>
              <SafeExternalLink :href="safeReferenceUrl" target="_blank" rel="noopener" class="store-link">
                {{ coin.referenceText || 'View Listing' }} ↗
              </SafeExternalLink>
            </template>
          </div>

          <CoinInfoGrid
            :purchase-price="coin.purchasePrice"
            :current-value="coin.currentValue"
            :denomination="coin.denomination"
            :era="coin.era"
            :mint="coin.mint"
            :material="coin.material"
            :weight-grams="coin.weightGrams"
            :diameter-mm="coin.diameterMm"
            :grade="coin.grade"
            :rarity-rating="coin.rarityRating"
          />

          <div v-if="coin.obverseDescription || coin.reverseDescription" class="descriptions-section">
            <h3>Description</h3>
            <div class="section-content-card">
              <p v-if="coin.obverseDescription"><strong>Obverse:</strong> {{ coin.obverseDescription }}</p>
              <p v-if="coin.reverseDescription"><strong>Reverse:</strong> {{ coin.reverseDescription }}</p>
            </div>
          </div>

          <div v-if="coin.notes" class="notes-section">
            <h3>Notes</h3>
            <div class="section-content-card">
              <p>{{ coin.notes }}</p>
            </div>
          </div>

          <CoinActivityJournal
            :entries="journalEntries"
            :coin-id="coin.id"
            @add="handleAddJournalEntry"
            @delete="handleDeleteJournalEntry"
          />

          <CoinListingStatus
            :coin-id="coin.id"
            :listing-status="coin.listingStatus"
            :listing-check-reason="coin.listingCheckReason"
            :listing-checked-at="coin.listingCheckedAt"
            @dismissed="refreshCoin"
          />

          <!-- Dashboard: Actions then AI Analysis stacked full-width -->
          <div class="detail-dashboard">
            <CoinActionsPanel
              :coin-id="coin.id"
              :coin-name="coin.name"
              :coin-ruler="coin.ruler"
              :coin-denomination="coin.denomination"
              :image-count="coin.images?.length ?? 0"
              :is-pwa="isPwa"
              @images-changed="refreshCoin"
              @estimate-applied="handleEstimateApplied"
            />

            <!-- Health Checklist (if coin is active and has health data) -->
            <CoinHealthChecklist
              v-if="coinHealth && !coin.isWishlist && !coin.isSold"
              :score="coinHealth.score"
              :grade="coinHealth.grade"
              :missing-items="coinHealth.missingItems"
              @quick-action="handleHealthQuickAction"
            />

            <div class="detail-ai">
              <CoinAIAnalysis
                :coin-id="coin.id"
                :obverse-analysis="coin.obverseAnalysis"
                :reverse-analysis="coin.reverseAnalysis"
                :ai-analysis="coin.aiAnalysis"
                :has-obverse="coin.images?.some(i => i.imageType === 'obverse') ?? false"
                :has-reverse="coin.images?.some(i => i.imageType === 'reverse') ?? false"
                @analysis-updated="refreshCoin"
              />
            </div>
          </div>
        </div>
      </div>
    </div>

    <SellModal v-if="showSellModal && coin" :coin="coin" @close="showSellModal = false" @confirm="confirmSell" />
    <PurchaseModal v-if="showPurchaseModal && coin" :coin="coin" @close="showPurchaseModal = false" @confirm="confirmPurchase" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useCoinsStore } from '@/stores/coins'
import ImageGallery from '@/components/ImageGallery.vue'
import SellModal from '@/components/SellModal.vue'
import PurchaseModal from '@/components/PurchaseModal.vue'
import CoinDetailHeaderActions from '@/components/coin/CoinDetailHeaderActions.vue'
import CoinTagsSection from '@/components/coin/CoinTagsSection.vue'
import CoinInfoGrid from '@/components/coin/CoinInfoGrid.vue'
import CoinActionsPanel from '@/components/coin/CoinActionsPanel.vue'
import CoinAIAnalysis from '@/components/coin/CoinAIAnalysis.vue'
import CoinListingStatus from '@/components/coin/CoinListingStatus.vue'
import CoinActivityJournal from '@/components/coin/CoinActivityJournal.vue'
import CoinHealthChecklist from '@/components/coin/CoinHealthChecklist.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { uploadImage, deleteCoin, deleteImage, purchaseCoin, sellCoin, getJournalEntries, addJournalEntry, deleteJournalEntry, getCoinValueHistory, getCoinHealthList } from '@/api/client'
import { removeBackground as removeBg } from '@imgly/background-removal'
import type { CoinImage, CoinJournal, CoinValueHistory as CoinValueHistoryType, CoinHealthItem, HealthQuickAction } from '@/types'
import { useDialog } from '@/composables/useDialog'
import { usePwa } from '@/composables/usePwa'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'

const { showConfirm, showAlert } = useDialog()
const route = useRoute()
const router = useRouter()
const store = useCoinsStore()
const { isPwa } = usePwa()

const removingBg = ref(false)
const showSellModal = ref(false)
const showPurchaseModal = ref(false)
const journalEntries = ref<CoinJournal[]>([])
const coinValueHistory = ref<CoinValueHistoryType[]>([])
const coinHealth = ref<CoinHealthItem | null>(null)

const coin = computed(() => store.currentCoin)
const safeReferenceUrl = computed(() => sanitizeExternalUrl(coin.value?.referenceUrl))

onMounted(() => {
  const id = Number(route.params['id'])
  store.fetchCoin(id)
  loadJournal(id)
  loadValueHistory(id)
  loadCoinHealth(id)
})

async function loadCoinHealth(coinId: number) {
  try {
    const res = await getCoinHealthList({ page: 1, limit: 1 })
    const match = res.data.coins.find(c => c.coinId === coinId)
    coinHealth.value = match ?? null
  } catch (err) {
    console.error('Failed to load coin health:', err)
  }
}

function handleHealthQuickAction(action: HealthQuickAction) {
  if (!coin.value) return
  
  switch (action) {
    case 'edit_metadata':
      router.push(`/coins/${coin.value.id}/edit`)
      break
    case 'upload_images':
      router.push(`/coins/${coin.value.id}/edit?tab=images`)
      break
    case 'run_valuation':
      // Trigger valuation action (handled by CoinActionsPanel)
      showAlert('Run valuation via the Actions panel')
      break
    case 'run_ai_analysis':
      // Trigger AI analysis action (handled by CoinAIAnalysis)
      showAlert('Run AI analysis via the AI Analysis panel')
      break
  }
}

function refreshCoin() {
  if (coin.value) {
    store.fetchCoin(coin.value.id)
    loadCoinHealth(coin.value.id)
  }
}

function handleEstimateApplied() {
  if (!coin.value) return
  store.fetchCoin(coin.value.id)
  loadValueHistory(coin.value.id)
  loadJournal(coin.value.id)
}

async function loadValueHistory(coinId: number) {
  try {
    const res = await getCoinValueHistory(coinId)
    coinValueHistory.value = res.data || []
  } catch {
    coinValueHistory.value = []
  }
}

async function loadJournal(coinId: number) {
  try {
    const res = await getJournalEntries(coinId)
    journalEntries.value = res.data || []
  } catch {
    journalEntries.value = []
  }
}

async function handleAddJournalEntry(entry: string) {
  if (!coin.value || !entry) return
  try {
    await addJournalEntry(coin.value.id, entry)
    loadJournal(coin.value.id)
  } catch {
    await showAlert('Failed to add journal entry', { title: 'Error' })
  }
}

async function handleDeleteJournalEntry(entryId: number) {
  if (!coin.value) return
  try {
    await deleteJournalEntry(coin.value.id, entryId)
    loadJournal(coin.value.id)
  } catch {
    await showAlert('Failed to delete journal entry', { title: 'Error' })
  }
}

async function handleRemoveBackground(image: CoinImage) {
  if (!coin.value) return
  removingBg.value = true

  try {
    const response = await fetch(`/uploads/${image.filePath}`)
    const srcBlob = await response.blob()
    const resultBlob = await removeBg(srcBlob, {
      output: { format: 'image/png', quality: 1 },
    })
    const file = new File([resultBlob], `${image.imageType}-processed.png`, { type: 'image/png' })
    await uploadImage(coin.value.id, file, image.imageType, image.isPrimary)
    await deleteImage(coin.value.id, image.id)
    store.fetchCoin(coin.value.id)
  } catch (err) {
    console.error('Background removal failed:', err)
  } finally {
    removingBg.value = false
  }
}

async function handleDeleteImage(image: CoinImage) {
  if (!coin.value || !await showConfirm(`Delete this ${image.imageType} image?`, { title: 'Delete Image', variant: 'danger' })) return
  try {
    await deleteImage(coin.value.id, image.id)
    store.fetchCoin(coin.value.id)
  } catch {
    await showAlert('Failed to delete image', { title: 'Error' })
  }
}

async function confirmPurchase(data: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string }) {
  if (!coin.value) return
  try {
    await purchaseCoin(coin.value.id, data)
    showPurchaseModal.value = false
    store.fetchCoin(coin.value.id)
  } catch {
    showPurchaseModal.value = false
  }
}

async function handleDelete() {
  if (!coin.value || !await showConfirm('Delete this coin from your collection?', { title: 'Delete Coin', variant: 'danger' })) return
  await deleteCoin(coin.value.id)
  router.push('/')
}

async function confirmSell(soldPrice: number | null, soldTo: string) {
  if (!coin.value) return
  try {
    await sellCoin(coin.value.id, soldPrice, soldTo)
    showSellModal.value = false
    router.push('/sold')
  } catch {
    await showAlert('Failed to mark as sold', { title: 'Error' })
    showSellModal.value = false
  }
}
</script>

<style scoped>
.detail-layout {
  display: grid;
  grid-template-columns: 400px 1fr;
  gap: 2rem;
  align-items: start;
  max-width: 1400px;
  margin-left: auto;
  margin-right: auto;
}

.detail-images {
  align-self: start;
}

/* Dashboard: Actions then AI Analysis stacked full-width */
.detail-dashboard {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.5rem;
  margin-top: 1.5rem;
}

.detail-title-section {
  margin-bottom: 1.5rem;
}

.detail-title-section h1 {
  margin-top: 0.5rem;
}

.detail-ruler {
  color: var(--text-secondary);
  font-size: 1.1rem;
  margin-top: 0.25rem;
}

.purchase-meta {
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem;
}

.store-link {
  color: var(--accent-gold);
  font-size: 0.85rem;
  text-decoration: none;
  margin-left: 0.5rem;
  white-space: nowrap;
}

.store-link:hover {
  text-decoration: underline;
}

.inscriptions-section,
.descriptions-section,
.notes-section {
  margin-bottom: 1.5rem;
}

.inscriptions-section h3,
.descriptions-section h3,
.notes-section h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.inscription {
  margin-bottom: 0.4rem;
}

.inscription-label {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-right: 0.4rem;
}

.inscription-text {
  font-style: italic;
  color: var(--text-secondary);
}

.descriptions-section p {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin-bottom: 0.4rem;
}

.notes-section p {
  color: var(--text-secondary);
  font-size: 0.9rem;
  white-space: pre-wrap;
  margin-bottom: 0;
}

/* Desktop: sticky action bar below fixed navbar (nav-content is 60px + 1px border = 61px) */
@media (min-width: 769px) {
  .sticky-action-bar {
    position: sticky;
    top: 61px;
    z-index: 10;
    background: var(--bg-primary);
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border-subtle);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  }
}

/* Desktop: sticky image sidebar — clear navbar (61px) + action bar (~60px) */
@media (min-width: 769px) {
  .detail-images {
    position: sticky;
    top: 125px;
    height: fit-content;
  }
}

/* Mobile: single-column, no sticky */
@media (max-width: 768px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }
  .detail-images { order: 1; }
  .detail-info { order: 2; }
}
</style>
