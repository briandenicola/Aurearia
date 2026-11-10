<template>
  <CoinDetailSectionPageShell section-title="Notes">
    <template #default="{ coin, refresh }">
      <div v-if="editing" class="mt-0">
        <textarea
          v-model="draft"
          class="form-input min-h-[12rem] w-full"
          maxlength="5000"
          aria-label="Coin notes"
          data-testid="notes-editor"
        ></textarea>
        <div class="mt-1 text-right text-chip text-text-muted">{{ draft.length }} / 5000</div>
        <p v-if="saveError" class="mt-2 text-body text-loss">{{ saveError }}</p>
        <div class="mt-3 flex justify-end gap-2">
          <button class="btn btn-ghost btn-sm" :disabled="saving" @click="cancelEdit">Cancel</button>
          <button
            class="btn btn-primary btn-sm"
            :disabled="saving || draft === (coin.notes ?? '')"
            data-testid="notes-save"
            @click="saveNotes(coin.id, refresh)"
          >
            {{ saving ? 'Saving...' : 'Save Notes' }}
          </button>
        </div>
      </div>
      <div v-else-if="coin.notes" class="mt-0">
        <div class="mb-3 flex justify-end">
          <button
            class="btn btn-secondary btn-sm inline-flex items-center gap-1"
            data-testid="notes-edit"
            @click="startEdit(coin.notes)"
          >
            <Pencil :size="14" />
            Edit
          </button>
        </div>
        <div class="section-content-card">
          <p class="text-base text-text-secondary whitespace-pre-wrap mb-0">{{ coin.notes }}</p>
        </div>
      </div>
      <div v-else class="empty-state text-center py-12 px-8">
        <p class="text-text-muted mb-6">No notes recorded for this coin.</p>
        <button class="btn btn-secondary btn-sm" data-testid="notes-add" @click="startEdit('')">
          Add Notes
        </button>
      </div>
    </template>
  </CoinDetailSectionPageShell>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Pencil } from 'lucide-vue-next'
import CoinDetailSectionPageShell from '@/components/coin/CoinDetailSectionPageShell.vue'
import { updateCoin } from '@/api/client'

const editing = ref(false)
const draft = ref('')
const saving = ref(false)
const saveError = ref('')

function startEdit(current: string) {
  draft.value = current
  saveError.value = ''
  editing.value = true
}

function cancelEdit() {
  editing.value = false
  saveError.value = ''
}

async function saveNotes(coinId: number, refresh: () => Promise<void>) {
  saving.value = true
  saveError.value = ''
  try {
    await updateCoin(coinId, { notes: draft.value })
    await refresh()
    editing.value = false
  } catch {
    saveError.value = 'Failed to save notes'
  } finally {
    saving.value = false
  }
}
</script>
