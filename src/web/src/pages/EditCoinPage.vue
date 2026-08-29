<template>
  <div class="container">
    <div class="form-wrapper">
      <div class="page-header">
        <h1>Edit Coin</h1>
      </div>
      <div v-if="loading" class="loading-overlay">
        <div class="spinner"></div>
      </div>
      <CoinForm v-else ref="coinFormRef" :form="form" :coin-id="form.id" submit-label="Save Changes" :loading="saving" @submit="handleSubmit" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CoinForm from '@/components/CoinForm.vue'
import { getCoin, updateCoin, uploadImage, deleteImage, extractText } from '@/api/client'
import type { Coin } from '@/types'
import { useDialog } from '@/composables/useDialog'

const { showAlert } = useDialog()
const route = useRoute()
const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const coinFormRef = ref<InstanceType<typeof CoinForm> | null>(null)

const form = reactive<Partial<Coin>>({})

onMounted(async () => {
  const id = Number(route.params['id'])
  try {
    const res = await getCoin(id)
    Object.assign(form, res.data)
    if (form.purchaseDate) {
      form.purchaseDate = form.purchaseDate.substring(0, 10)
    }
  } catch {
    await showAlert('Failed to load coin', { title: 'Error' })
    router.push('/')
  } finally {
    loading.value = false
  }
})

async function handleSubmit() {
  saving.value = true
  // Tracks how far the save got, so a failure reports what actually happened.
  // The coin's fields and its images are saved by separate requests, and the
  // image ones can fail on their own — telling the user the whole edit was
  // lost when only the upload failed sends them back to redo work that saved.
  let coinSaved = false
  try {
    await updateCoin(form.id!, form)
    coinSaved = true

    const formComp = coinFormRef.value
    const coinId = form.id!

    // Replace a side by uploading the new image FIRST, then deleting the one
    // it supersedes. Deleting first means a failed upload leaves the coin with
    // no image at all, which is how a dropped upload turned into lost data.
    // There is no unique constraint on (coin_id, image_type), so the coin
    // holding both briefly is fine, and uploading the obverse as primary
    // clears the old primary flag before the old row goes away.
    async function replaceSide(
      imageType: 'obverse' | 'reverse',
      file: File,
      isPrimary: boolean,
      removedId: number | null,
    ) {
      await uploadImage(coinId, file, imageType, isPrimary)
      const superseded = removedId ?? form.images?.find((i) => i.imageType === imageType)?.id
      if (superseded) {
        await deleteImage(coinId, superseded)
      }
    }

    if (formComp?.obverseFile) {
      await replaceSide('obverse', formComp.obverseFile, true, formComp.removedObverseId)
    } else if (formComp?.removedObverseId) {
      await deleteImage(coinId, formComp.removedObverseId)
    }

    if (formComp?.reverseFile) {
      await replaceSide('reverse', formComp.reverseFile, false, formComp.removedReverseId)
    } else if (formComp?.removedReverseId) {
      await deleteImage(coinId, formComp.removedReverseId)
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
          await updateCoin(coinId, { notes: updatedNotes })
        }
      } catch {
        console.warn('Card text extraction failed – coin saved without card notes')
      }
    }

    router.back()
  } catch {
    await showAlert(
      coinSaved
        ? 'Coin details were saved, but the image could not be uploaded. Try adding the image again.'
        : 'Failed to update coin',
      { title: 'Error' },
    )
  } finally {
    saving.value = false
  }
}
</script>
