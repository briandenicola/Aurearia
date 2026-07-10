<template>
  <div class="container">
    <div v-if="store.loading && !coin" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="coin">
      <div class="md:sticky md:top-[61px] md:z-10 md:border-b md:border-border-subtle md:bg-surface md:px-0 md:py-3 md:shadow-[0_4px_12px_rgba(0,0,0,0.3)]">
        <CoinDetailHeaderActions
          :is-wishlist="coin.isWishlist"
          :is-sold="coin.isSold"
          :coin-id="coin.id"
          :sharing="sharing"
          :duplicating="duplicating"
          @share="handleShare"
          @sell="showSellModal = true"
          @duplicate="handleDuplicate"
          @delete="handleDelete"
        />
      </div>

      <div class="mx-auto grid max-w-[1400px] grid-cols-1 items-start gap-8 md:grid-cols-[400px_minmax(0,1fr)]">
        <!-- T009-T011: Dual-side hero media -->
        <div class="order-1 self-start md:sticky md:top-[125px] md:h-fit">
          <div class="grid grid-cols-2 gap-4">
            <div class="relative aspect-square w-full overflow-hidden rounded-md border border-border-subtle bg-card">
              <AuthenticatedImage
                v-if="obverseImage"
                :media-path="obverseImage.filePath"
                alt="Obverse"
                class="h-full w-full cursor-pointer object-contain scale-[1.28] transition-[opacity,transform] duration-200 hover:scale-[1.32] hover:opacity-[0.85]"
                @click="openLightbox(obverseImage)"
              />
              <div v-else class="flex h-full w-full flex-col items-center justify-center gap-2">
                <span class="text-body font-medium text-text-secondary">Obverse</span>
                <span class="text-sm italic text-text-muted">No image</span>
              </div>
            </div>
            <div class="relative aspect-square w-full overflow-hidden rounded-md border border-border-subtle bg-card">
              <AuthenticatedImage
                v-if="reverseImage"
                :media-path="reverseImage.filePath"
                alt="Reverse"
                class="h-full w-full cursor-pointer object-contain scale-[1.28] transition-[opacity,transform] duration-200 hover:scale-[1.32] hover:opacity-[0.85]"
                @click="openLightbox(reverseImage)"
              />
              <div v-else class="flex h-full w-full flex-col items-center justify-center gap-2">
                <span class="text-body font-medium text-text-secondary">Reverse</span>
                <span class="text-sm italic text-text-muted">No image</span>
              </div>
            </div>
          </div>
          <div v-if="coin.isWishlist" class="mt-3">
            <button class="btn btn-primary w-full justify-center" @click="showPurchaseModal = true">
              Mark as Purchased
            </button>
          </div>
        </div>

        <!-- Info -->
        <div class="order-2">
          <!-- T012: Title hierarchy -->
          <div class="mb-6">
            <h1 class="mt-2 font-display text-2xl font-semibold text-heading">{{ coin.name }}</h1>
            <p v-if="coin.ruler" class="mt-1 text-[1.1rem] text-text-secondary">{{ coin.ruler }}</p>
            <div v-if="coin.category" class="mt-3 flex flex-wrap gap-2">
              <span class="badge" :class="`badge-${coin.category.toLowerCase()}`">{{ coin.category }}</span>
              <span v-if="coin.isWishlist" class="chip-sm">Wishlist</span>
              <span v-if="coin.isSold" class="chip-sm">Sold</span>
            </div>
          </div>

          <div v-if="coin.obverseInscription || coin.reverseInscription || coin.obverseDescription || coin.reverseDescription" class="mb-6">
            <h3 class="mb-3 font-display text-base font-medium text-heading">Inscription</h3>
            <div class="rounded-sm border border-border-subtle bg-card p-4">
              <div class="grid gap-6 md:grid-cols-2">
                <div v-if="coin.obverseInscription || coin.obverseDescription" class="flex flex-col gap-3">
                  <h4 class="m-0 text-base font-medium text-heading">Obverse</h4>
                  <div v-if="coin.obverseInscription" class="flex flex-col gap-1">
                    <span class="section-label">Inscription:</span>
                    <span class="text-base italic text-text-secondary">{{ coin.obverseInscription }}</span>
                  </div>
                  <p v-if="coin.obverseDescription" class="m-0 text-base leading-6 text-text-secondary">{{ coin.obverseDescription }}</p>
                </div>
                <div v-if="coin.reverseInscription || coin.reverseDescription" class="flex flex-col gap-3">
                  <h4 class="m-0 text-base font-medium text-heading">Reverse</h4>
                  <div v-if="coin.reverseInscription" class="flex flex-col gap-1">
                    <span class="section-label">Inscription:</span>
                    <span class="text-base italic text-text-secondary">{{ coin.reverseInscription }}</span>
                  </div>
                  <p v-if="coin.reverseDescription" class="m-0 text-base leading-6 text-text-secondary">{{ coin.reverseDescription }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- T014-T016: Metadata table -->
          <div v-if="metadataRows.length" class="mb-6">
            <h3 class="mb-3 font-display text-base font-medium text-heading">Details</h3>
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
          <div class="mb-6">
            <h3 class="mb-3 font-display text-base font-medium text-heading">Additional Details</h3>
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
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import CoinDetailHeaderActions from '@/components/coin/CoinDetailHeaderActions.vue'
import CoinTagsSection from '@/components/coin/CoinTagsSection.vue'
import CoinDetailMetadataTable from '@/components/coin/CoinDetailMetadataTable.vue'
import CoinDetailSectionLinks from '@/components/coin/CoinDetailSectionLinks.vue'
import CoinListingStatus from '@/components/coin/CoinListingStatus.vue'
import CoinReferencesSection from '@/components/coin/CoinReferencesSection.vue'
import { deleteCoin, duplicateCoin, purchaseCoin, sellCoin } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import { useCoinDetailMetadataRows } from '@/composables/useCoinDetailMetadataRows'
import { useCoinShareCard } from '@/composables/useCoinShareCard'
import type { CoinImage } from '@/types'

const { showConfirm, showAlert } = useDialog()
const route = useRoute()
const router = useRouter()
const store = useCoinsStore()

const showSellModal = ref(false)
const showPurchaseModal = ref(false)
const lightboxImage = ref<CoinImage | null>(null)
const { sharing, shareCoinCard } = useCoinShareCard()
const duplicating = ref(false)

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

async function handleShare() {
  if (!coin.value) return
  await shareCoinCard(coin.value)
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

async function handleDuplicate() {
  if (!coin.value || duplicating.value) return
  duplicating.value = true
  try {
    const res = await duplicateCoin(coin.value.id)
    router.push(`/coin/${res.data.id}`)
  } catch {
    await showAlert('Failed to duplicate coin', { title: 'Error' })
  } finally {
    duplicating.value = false
  }
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
