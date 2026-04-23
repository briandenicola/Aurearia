<template>
  <div class="container">
    <div class="form-wrapper">
      <div class="page-header">
        <h1>Add Coin</h1>
      </div>
      <CoinForm ref="coinFormRef" :form="form" submit-label="Add to Collection" :loading="saving" @submit="handleSubmit" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useCoinsStore } from '@/stores/coins'
import { uploadImage, extractText, updateCoin } from '@/api/client'
import CoinForm from '@/components/CoinForm.vue'
import type { Coin } from '@/types'
import { useDialog } from '@/composables/useDialog'

const { showAlert } = useDialog()
const router = useRouter()
const store = useCoinsStore()
const saving = ref(false)
const coinFormRef = ref<InstanceType<typeof CoinForm> | null>(null)

const form = reactive<Partial<Coin>>({
  name: '',
  category: 'Roman',
  material: 'Silver',
  denomination: '',
  mint: '',
  weightGrams: null,
  diameterMm: null,
  grade: '',
  obverseInscription: '',
  reverseInscription: '',
  obverseDescription: '',
  reverseDescription: '',
  rarityRating: '',
  purchasePrice: null,
  currentValue: null,
  purchaseDate: null,
  purchaseLocation: '',
  notes: '',
  referenceUrl: '',
  referenceText: 'Store Link',
  isWishlist: useRoute().query.wishlist === 'true',
})

async function handleSubmit() {
  saving.value = true
  try {
    const coin = await store.addCoin(form)
    const formComp = coinFormRef.value

    // Upload images if selected
    if (formComp?.obverseFile) {
      await uploadImage(coin.id, formComp.obverseFile, 'obverse', true)
    }
    if (formComp?.reverseFile) {
      await uploadImage(coin.id, formComp.reverseFile, 'reverse', false)
    }

    // Extract text from store card if uploaded
    if (formComp?.cardFile) {
      try {
        const res = await extractText(formComp.cardFile)
        const extractedText = res.data.text
        if (extractedText) {
          const existingNotes = form.notes || ''
          const updatedNotes = existingNotes
            ? `${existingNotes}\n\n--- Store Card ---\n${extractedText}`
            : `--- Store Card ---\n${extractedText}`
          await updateCoin(coin.id, { notes: updatedNotes })
        }
      } catch {
        console.warn('Card text extraction failed – coin saved without card notes')
      }
    }

    router.push(`/coin/${coin.id}`)
  } catch {
    await showAlert('Failed to add coin', { title: 'Error' })
  } finally {
    saving.value = false
  }
}
</script>
