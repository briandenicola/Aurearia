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

          <div v-if="coin.obverseInscription || coin.reverseInscription || coin.obverseDescription || coin.reverseDescription" class="inscription-section">
            <h3>Inscription</h3>
            <div class="section-content-card">
              <div class="inscription-grid">
                <div v-if="coin.obverseInscription || coin.obverseDescription" class="inscription-side">
                  <h4 class="side-heading">Obverse</h4>
                  <div v-if="coin.obverseInscription" class="inscription-line">
                    <span class="inscription-label">Inscription:</span>
                    <span class="inscription-text">{{ coin.obverseInscription }}</span>
                  </div>
                  <p v-if="coin.obverseDescription" class="description-text">{{ coin.obverseDescription }}</p>
                </div>
                <div v-if="coin.reverseInscription || coin.reverseDescription" class="inscription-side">
                  <h4 class="side-heading">Reverse</h4>
                  <div v-if="coin.reverseInscription" class="inscription-line">
                    <span class="inscription-label">Inscription:</span>
                    <span class="inscription-text">{{ coin.reverseInscription }}</span>
                  </div>
                  <p v-if="coin.reverseDescription" class="description-text">{{ coin.reverseDescription }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- T014-T016: Metadata table -->
          <div v-if="metadataRows.length" class="metadata-section">
            <h3>Details</h3>
            <CoinDetailMetadataTable :rows="metadataRows" />
          </div>

          <CoinReferencesSection
            :coin-id="coin.id"
            :references="coin.references ?? []"
            @changed="refreshCoin"
          />

          <CoinTagsSection
            :tags="coin.tags ?? []"
            :sets="coin.sets ?? []"
            :coin-id="coin.id"
            @tags-changed="refreshCoin"
          />

          <CoinListingStatus
            :coin-id="coin.id"
            :listing-status="coin.listingStatus"
            :listing-check-reason="coin.listingCheckReason"
            :listing-checked-at="coin.listingCheckedAt"
            @dismissed="refreshCoin"
          />

          <!-- T019: Settings-style section links -->
          <div class="sections-list">
            <h3>Additional Details</h3>
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
import { deleteCoin, purchaseCoin, sellCoin } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
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

.inscription-section,
.metadata-section,
.sections-list {
  margin-bottom: 1.5rem;
}

.inscription-section h3,
.metadata-section h3,
.sections-list h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.section-content-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 1rem;
}

.inscription-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
}

.inscription-side {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.side-heading {
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-heading);
  margin: 0;
}

.inscription-line {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.inscription-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  font-weight: 500;
}

.inscription-text {
  font-style: italic;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.description-text {
  font-size: 0.9rem;
  color: var(--text-secondary);
  line-height: 1.5;
  margin: 0;
}

@media (max-width: 768px) {
  .inscription-grid {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }
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
