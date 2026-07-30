<template>
  <div class="container py-4 pb-20 md:py-8 md:pb-20">
    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading showcase...</p>
    </div>

    <div v-else-if="notFound" class="empty-state">
      <h3>Showcase not found</h3>
      <p>This showcase may have been removed or the link is incorrect.</p>
    </div>

    <template v-else-if="showcase">
      <div class="mb-8 text-center">
        <h1 class="mb-2 mt-0 text-2xl text-text-primary">{{ showcase.title }}</h1>
        <p v-if="showcase.ownerName" class="mb-2 mt-0 text-base text-gold">Curated by {{ showcase.ownerName }}</p>
        <p v-if="showcase.description" class="mx-auto m-0 max-w-[600px] text-base leading-6 text-text-secondary">{{ showcase.description }}</p>
      </div>

      <div v-if="trayCoins.length" class="flex flex-col gap-4 pb-20">
        <div class="flex justify-center md:justify-end">
          <label class="inline-flex w-full items-center justify-between gap-3 rounded-full border border-border-subtle bg-[rgba(255,255,255,0.04)] px-3 py-2 text-[0.8rem] font-semibold uppercase tracking-[0.04em] text-text-secondary md:w-auto md:justify-start">
            <span>Coin size</span>
            <input
              id="showcase-tray-size-slider"
              v-model.number="traySizeScale"
              class="w-[140px] accent-[var(--accent-gold)]"
              type="range"
              min="0.75"
              max="1.4"
              step="0.05"
            />
            <span class="min-w-[3.2rem] text-right text-text-primary">{{ traySizeScale.toFixed(2) }}x</span>
          </label>
        </div>
        <MuseumTray
          :coins="currentDrawerCoins"
          :felt-theme="feltColor"
          :size-scale="traySizeScale"
          :image-src-resolver="imageUrl"
          :interactive="false"
        />
        <TrayControls
          v-if="totalDrawers > 1"
          :drawer-index="drawerIndex"
          :total-drawers="totalDrawers"
          @prev="handlePrevDrawer"
          @next="handleNextDrawer"
        />
      </div>

      <div v-else class="empty-state">
        <p>This showcase has no coins yet.</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getPublicShowcase } from '@/api/client'
import MuseumTray from '@/components/tray/MuseumTray.vue'
import TrayControls from '@/components/tray/TrayControls.vue'
import { useTrayPreference } from '@/composables/useTrayPreference'
import { publicShowcaseMediaUrl } from '@/utils/media'
import { getDrawerCoins, getTotalDrawers, type TrayCoin } from '@/utils/trayLayout'

interface PublicCoinImage {
  id: number
  filePath: string
  imageType: string
  isPrimary?: boolean
}

interface PublicCoin {
  id: number
  name?: string
  diameterMm?: number | null
  era?: string
  category?: string
  grade?: string
  images?: PublicCoinImage[]
}

interface PublicShowcase {
  title: string
  description?: string
  ownerName?: string
}

const route = useRoute()
const { feltColor } = useTrayPreference()
const loading = ref(true)
const notFound = ref(false)
const showcase = ref<PublicShowcase | null>(null)
const coins = ref<PublicCoin[]>([])
const drawerIndex = ref(0)
const coinsPerDrawer = 12
const traySizeScale = ref(1)

const trayCoins = computed((): TrayCoin[] => coins.value.map(coin => ({
  id: coin.id,
  name: coin.name ?? 'Untitled',
  diameterMm: coin.diameterMm ?? null,
  images: coin.images ?? [],
})))

const currentDrawerCoins = computed(() => getDrawerCoins(trayCoins.value, drawerIndex.value, coinsPerDrawer))
const totalDrawers = computed(() => getTotalDrawers(trayCoins.value.length, coinsPerDrawer))

watch(totalDrawers, (drawers) => {
  if (drawers === 0) {
    drawerIndex.value = 0
    return
  }
  drawerIndex.value = Math.min(drawerIndex.value, drawers - 1)
})

watch(traySizeScale, (value) => {
  const normalizedValue = Math.min(1.4, Math.max(0.75, Number(value) || 1))
  traySizeScale.value = normalizedValue
  localStorage.setItem('tray:sizeScale', normalizedValue.toString())
})

function imageUrl(filePath: string): string {
  return publicShowcaseMediaUrl(route.params.slug as string, filePath)
}

function handlePrevDrawer() {
  drawerIndex.value = Math.max(0, drawerIndex.value - 1)
}

function handleNextDrawer() {
  drawerIndex.value = Math.min(totalDrawers.value - 1, drawerIndex.value + 1)
}

async function loadShowcase() {
  loading.value = true
  const slug = route.params.slug as string
  try {
    const res = await getPublicShowcase(slug)
    showcase.value = res.data?.showcase ?? null
    coins.value = res.data?.coins ?? []
    drawerIndex.value = 0
    if (!showcase.value) notFound.value = true
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const storedScale = Number(localStorage.getItem('tray:sizeScale'))
  if (Number.isFinite(storedScale)) {
    traySizeScale.value = Math.min(1.4, Math.max(0.75, storedScale))
  }
  loadShowcase()
})
</script>
