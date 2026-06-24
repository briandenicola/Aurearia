<template>
  <div class="container">
    <div v-if="loading" class="loading-state">Loading showcase...</div>

    <div v-else-if="notFound" class="empty-state">
      <h3>Showcase not found</h3>
      <p>This showcase may have been removed or the link is incorrect.</p>
    </div>

    <template v-else-if="showcase">
      <div class="showcase-header">
        <h1>{{ showcase.title }}</h1>
        <p v-if="showcase.ownerName" class="owner">Curated by {{ showcase.ownerName }}</p>
        <p v-if="showcase.description" class="description">{{ showcase.description }}</p>
      </div>

      <div v-if="trayCoins.length" class="public-tray-section">
        <MuseumTray
          :coins="currentDrawerCoins"
          :felt-theme="feltColor"
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

onMounted(loadShowcase)
</script>

<style scoped>
.container { max-width: 1200px; margin: 0 auto; padding: 2rem 1.5rem; }
.public-tray-section { display: flex; flex-direction: column; gap: 1rem; padding-bottom: 5rem; }
.loading-state { text-align: center; padding: 2rem; color: var(--text-secondary); }
.empty-state { text-align: center; padding: 3rem; color: var(--text-secondary); }
.empty-state h3 { color: var(--text-primary); margin-bottom: 0.5rem; }

.showcase-header { text-align: center; margin-bottom: 2rem; }
.showcase-header h1 { font-size: 2rem; color: var(--text-primary); margin: 0 0 0.5rem; }
.owner { color: var(--accent-gold); font-size: 0.9rem; margin: 0 0 0.5rem; }
.description { color: var(--text-secondary); font-size: 1rem; max-width: 600px; margin: 0 auto; line-height: 1.5; }

@media (max-width: 575px) {
  .container { padding: 1rem 0.75rem; }
}
</style>
