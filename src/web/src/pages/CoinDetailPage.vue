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
        <!-- T009-T011: Dual-side hero media -->
        <div class="detail-hero-media">
          <div class="hero-media-grid">
            <div class="hero-slot">
              <img
                v-if="obverseImage"
                :src="`/uploads/${obverseImage.filePath}`"
                alt="Obverse"
                class="hero-image"
                @click="openLightbox(obverseImage)"
              />
              <div v-else class="hero-placeholder">
                <span class="placeholder-label">Obverse</span>
                <span class="placeholder-text">No image</span>
              </div>
            </div>
            <div class="hero-slot">
              <img
                v-if="reverseImage"
                :src="`/uploads/${reverseImage.filePath}`"
                alt="Reverse"
                class="hero-image"
                @click="openLightbox(reverseImage)"
              />
              <div v-else class="hero-placeholder">
                <span class="placeholder-label">Reverse</span>
                <span class="placeholder-text">No image</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Info -->
        <div class="detail-info">
          <!-- T012: Title hierarchy -->
          <div class="detail-title-section">
            <h1>{{ coin.name }}</h1>
            <p v-if="coin.ruler" class="detail-ruler">{{ coin.ruler }}</p>
            <div v-if="coin.category" class="title-badges">
              <span class="badge" :class="`badge-${coin.category.toLowerCase()}`">{{ coin.category }}</span>
              <span v-if="coin.isWishlist" class="chip-sm">Wishlist</span>
              <span v-if="coin.isSold" class="chip-sm">Sold</span>
            </div>
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

          <!-- T014-T016: Metadata table -->
          <div v-if="metadataRows.length" class="metadata-section">
            <h3>Details</h3>
            <CoinDetailMetadataTable :rows="metadataRows" />
          </div>

          <CoinTagsSection
            :tags="coin.tags ?? []"
            :coin-id="coin.id"
            @tags-changed="refreshCoin"
          />

          <CoinReferencesSection
            :coin-id="coin.id"
            :references="coin.references ?? []"
            @changed="refreshCoin"
          />

          <div v-if="coin.obverseDescription || coin.reverseDescription" class="descriptions-section">
            <h3>Description</h3>
            <div class="section-content-card">
              <p v-if="coin.obverseDescription"><strong>Obverse:</strong> {{ coin.obverseDescription }}</p>
              <p v-if="coin.reverseDescription"><strong>Reverse:</strong> {{ coin.reverseDescription }}</p>
            </div>
          </div>

          <CoinListingStatus
            :coin-id="coin.id"
            :listing-status="coin.listingStatus"
            :listing-check-reason="coin.listingCheckReason"
            :listing-checked-at="coin.listingCheckedAt"
            @dismissed="refreshCoin"
          />

          <!-- T019: Settings-style section links -->
          <div class="sections-list">
            <h3>Details</h3>
            <CoinDetailSectionLinks :coin-id="coin.id" />
          </div>
        </div>
      </div>
    </div>

    <SellModal v-if="showSellModal && coin" :coin="coin" @close="showSellModal = false" @confirm="confirmSell" />
    <PurchaseModal v-if="showPurchaseModal && coin" :coin="coin" @close="showPurchaseModal = false" @confirm="confirmPurchase" />
    <ImageLightbox
      v-if="lightboxImage && coin"
      :coin-id="coin.id"
      :image-id="lightboxImage.id"
      :image-path="lightboxImage.filePath"
      :image-type="lightboxImage.imageType"
      @close="lightboxImage = null"
      @saved="handleImageSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useCoinsStore } from '@/stores/coins'
import SellModal from '@/components/SellModal.vue'
import PurchaseModal from '@/components/PurchaseModal.vue'
import ImageLightbox from '@/components/ImageLightbox.vue'
import CoinDetailHeaderActions from '@/components/coin/CoinDetailHeaderActions.vue'
import CoinTagsSection from '@/components/coin/CoinTagsSection.vue'
import CoinDetailMetadataTable from '@/components/coin/CoinDetailMetadataTable.vue'
import CoinDetailSectionLinks from '@/components/coin/CoinDetailSectionLinks.vue'
import CoinListingStatus from '@/components/coin/CoinListingStatus.vue'
import CoinReferencesSection from '@/components/coin/CoinReferencesSection.vue'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { deleteCoin, purchaseCoin, sellCoin } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'
import { useCoinDetailMetadataRows } from '@/composables/useCoinDetailMetadataRows'
import type { CoinImage } from '@/types'

const { showConfirm, showAlert } = useDialog()
const route = useRoute()
const router = useRouter()
const store = useCoinsStore()

const showSellModal = ref(false)
const showPurchaseModal = ref(false)
const lightboxImage = ref<CoinImage | null>(null)

const coin = computed(() => store.currentCoin)
const safeReferenceUrl = computed(() => sanitizeExternalUrl(coin.value?.referenceUrl))

// T010: Deterministic media slot logic
const obverseImage = computed(() => coin.value?.images?.find(i => i.imageType === 'obverse') ?? null)
const reverseImage = computed(() => coin.value?.images?.find(i => i.imageType === 'reverse') ?? null)

// T015: Metadata rows
const metadataRows = computed(() => {
  if (!coin.value) return []
  return useCoinDetailMetadataRows(coin.value).rows.value
})

onMounted(() => {
  const id = Number(route.params.id)
  store.fetchCoin(id)
})

function refreshCoin() {
  if (coin.value) {
    store.fetchCoin(coin.value.id)
  }
}

function openLightbox(image: CoinImage) {
  lightboxImage.value = image
}

function handleImageSaved() {
  refreshCoin()
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

.detail-hero-media {
  align-self: start;
}

/* T011: Hero media grid for dual-side default display */
.hero-media-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.hero-slot {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
}

.hero-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
  cursor: pointer;
  transition: opacity var(--transition-fast);
}

.hero-image:hover {
  opacity: 0.85;
}

.hero-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  gap: 0.5rem;
}

.placeholder-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.placeholder-text {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-style: italic;
}

/* T012: Title hierarchy */
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

.title-badges {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}

.purchase-meta {
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem;
  font-style: italic;
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
.metadata-section,
.sections-list {
  margin-bottom: 1.5rem;
}

.inscriptions-section h3,
.descriptions-section h3,
.metadata-section h3,
.sections-list h3 {
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

/* T013: Desktop-only sticky behavior */
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

  .detail-hero-media {
    position: sticky;
    top: 125px;
    height: fit-content;
  }
}

/* T013: Mobile - single-column, no sticky */
@media (max-width: 768px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }
  .detail-hero-media { order: 1; }
  .detail-info { order: 2; }
}
</style>
