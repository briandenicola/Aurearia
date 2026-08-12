<template>
  <div class="mb-6 grid gap-3">
    <NumistaLookupPanel
      :initial-query="initialQuery"
      :evidence="evidence"
      path="direct"
      :is-admin="auth.isAdmin"
      @confirmed="addSelectedReference"
    />
    <p v-if="saveMessage" class="m-0 text-body" :class="saveError ? 'text-warning' : 'text-gold'" role="status">
      {{ saveMessage }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { createCoinReference, getApiErrorMessage, proposeNumistaQuery } from '@/api/client'
import NumistaLookupPanel from '@/components/numista/NumistaLookupPanel.vue'
import { useAuthStore } from '@/stores/auth'
import { buildDirectNumistaEvidence } from '@/utils/numistaLookup'
import type { Coin, NumistaCandidate } from '@/types'

const props = defineProps<{
  coinId: number
  coinName: string
  coinRuler: string
  coinDenomination: string
  coinMint: string
  coinDateRange: string
  coinMaterial: Coin['material']
  coinObverseInscription: string
  coinReverseInscription: string
}>()

const emit = defineEmits<{ referenceAdded: [candidate: NumistaCandidate] }>()
const auth = useAuthStore()
const saving = ref(false)
const saveMessage = ref('')
const saveError = ref(false)
const initialQuery = ref('')
let proposalRequest = 0

const evidence = computed(() => buildDirectNumistaEvidence({
  name: props.coinName,
  ruler: props.coinRuler,
  denomination: props.coinDenomination,
  mint: props.coinMint,
  dateRange: props.coinDateRange,
  material: props.coinMaterial,
  obverseInscription: props.coinObverseInscription,
  reverseInscription: props.coinReverseInscription,
}))

watch(evidence, async (currentEvidence) => {
  const request = ++proposalRequest
  try {
    const response = await proposeNumistaQuery({ path: 'direct', evidence: currentEvidence })
    if (request === proposalRequest) initialQuery.value = response.data.query
  } catch {
    if (request === proposalRequest) initialQuery.value = ''
  }
}, { immediate: true })

async function addSelectedReference(candidate: NumistaCandidate) {
  if (saving.value) return

  saving.value = true
  saveMessage.value = ''
  saveError.value = false
  try {
    await createCoinReference(props.coinId, {
      catalog: 'Numista',
      number: String(candidate.id),
      uri: candidate.canonicalUrl,
    })
    saveMessage.value = 'Numista reference added.'
    emit('referenceAdded', candidate)
  } catch (error) {
    saveError.value = true
    saveMessage.value = getApiErrorMessage(error) || 'The Numista reference could not be added.'
  } finally {
    saving.value = false
  }
}
</script>
