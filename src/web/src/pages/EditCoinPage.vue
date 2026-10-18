<template>
  <div class="container">
    <div class="form-wrapper">
      <DesktopPageContext title="Edit Coin">
        <div class="page-header">
          <h1>Edit Coin</h1>
        </div>
      </DesktopPageContext>
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
import { useQuickAccess } from '@/composables/useQuickAccess'
import DesktopPageContext from '@/components/DesktopPageContext.vue'

const { showAlert } = useDialog()
const { refresh: refreshQuickAccess } = useQuickAccess()
const route = useRoute()
const router = useRouter()
const loading = ref(true)
const saving = ref(false)
const coinFormRef = ref<InstanceType<typeof CoinForm> | null>(null)

const form = reactive<Partial<Coin>>({})
let uploadedCard: File | null = null
type CoinFace = 'obverse' | 'reverse'
const savedFaces = new Map<CoinFace, { file: File; imageId: number; supersededId: number | null }>()
const removedImages = new Set<number>()
let cleanupPending = false

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
    async function removeImage(imageId: number) {
      if (removedImages.has(imageId)) return
      const response = await deleteImage(coinId, imageId)
      removedImages.add(imageId)
      cleanupPending ||= response.data.cleanupPending === true
    }

    // Replace a side by uploading the new image FIRST, then deleting the one
    // it supersedes. Deleting first means a failed upload leaves the coin with
    // no image at all, which is how a dropped upload turned into lost data.
    // There is no unique constraint on (coin_id, image_type), so the coin
    // holding both briefly is fine, and uploading the obverse as primary
    // clears the old primary flag before the old row goes away.
    async function replaceSide(
      imageType: CoinFace,
      file: File,
      isPrimary: boolean,
      removedId: number | null,
    ) {
      const saved = savedFaces.get(imageType)
      if (saved?.supersededId) {
        await removeImage(saved.supersededId)
        saved.supersededId = null
      }
      if (saved?.file === file) return
      const superseded = saved?.imageId ?? removedId ?? form.images?.find((i) => i.imageType === imageType)?.id ?? null
      const response = await uploadImage(coinId, file, imageType, isPrimary)
      const replacement = { file, imageId: response.data.id, supersededId: superseded }
      savedFaces.set(imageType, replacement)
      if (superseded) {
        await removeImage(superseded)
        replacement.supersededId = null
      }
    }

    async function removeSide(imageType: CoinFace, removedId: number) {
      const saved = savedFaces.get(imageType)
      if (saved?.supersededId) await removeImage(saved.supersededId)
      await removeImage(saved?.imageId ?? removedId)
      savedFaces.delete(imageType)
    }

    const obverseRemovalId = savedFaces.get('obverse')?.imageId ?? formComp?.removedObverseId
    const reverseRemovalId = savedFaces.get('reverse')?.imageId ?? formComp?.removedReverseId
    if (formComp?.obverseFile) {
      await replaceSide('obverse', formComp.obverseFile, true, formComp.removedObverseId)
    } else if (obverseRemovalId) {
      await removeSide('obverse', obverseRemovalId)
    }

    if (formComp?.reverseFile) {
      await replaceSide('reverse', formComp.reverseFile, false, formComp.removedReverseId)
    } else if (reverseRemovalId) {
      await removeSide('reverse', reverseRemovalId)
    }

    if (formComp?.cardFile) {
      if (uploadedCard !== formComp.cardFile) {
        await uploadImage(coinId, formComp.cardFile, 'card', false)
        uploadedCard = formComp.cardFile
      }
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
      } catch (error) {
        console.error('Attribution card notes could not be saved', error)
        await showAlert('The original attribution card was saved privately, but its text could not be added to Notes.', { title: 'Card notes unavailable' })
      }
    }

    await refreshQuickAccess()
    if (cleanupPending) {
      await showAlert('Your changes were saved. Old image files are awaiting cleanup; the API will retry on startup. Do not upload the replacement again.', { title: 'Image cleanup pending' })
    }
    router.back()
  } catch (error: unknown) {
    const code = (error as { response?: { data?: { code?: string } } })?.response?.data?.code
    if (!coinSaved && code === 'slot_occupied') {
      await coinFormRef.value?.refreshStorageOccupancy()
      await showAlert('That tray slot was just taken. Choose another available slot; your form has been preserved.', { title: 'Slot unavailable' })
      return
    }
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
