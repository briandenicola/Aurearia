<template>
  <div class="container">
    <header class="page-header">
      <h1>Review Agentic Set</h1>
      <RouterLink class="btn btn-secondary btn-sm" to="/sets">Back to Sets</RouterLink>
    </header>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading proposal...</p>
    </div>

    <div v-else-if="error" class="empty-state card">
      <h3>Proposal unavailable</h3>
      <p>{{ error }}</p>
    </div>

    <div v-else-if="proposal" class="proposal-layout">
      <section class="card proposal-hero">
        <div class="proposal-title-row">
          <div>
            <span class="section-label">Proposed set</span>
            <h2>{{ proposal.proposedName }}</h2>
          </div>
          <span class="badge" :style="{ borderColor: proposal.color || 'var(--accent-gold)' }">
            {{ proposal.status }}
          </span>
        </div>
        <p class="prompt">{{ proposal.originalPrompt }}</p>
        <p v-if="proposal.description" class="description">{{ proposal.description }}</p>

        <div class="info-grid">
          <div class="info-card">
            <span class="info-label">Scope</span>
            <strong>{{ proposal.selectedScope || 'Review required' }}</strong>
          </div>
          <div class="info-card">
            <span class="info-label">Slots</span>
            <strong>{{ slots.length }}</strong>
          </div>
          <div class="info-card">
            <span class="info-label">Estimated filled</span>
            <strong>{{ estimatedFilled }} / {{ estimatedTotal }}</strong>
          </div>
        </div>
      </section>

      <section v-if="proposalIssue" class="card issue-card">
        <span class="section-label">Workflow issue</span>
        <p class="description">{{ proposalIssue }}</p>
      </section>

      <section v-if="scopeSummary || scopeOptions.length" class="card">
        <span class="section-label">Scope interpretation</span>
        <p v-if="scopeSummary" class="description">{{ scopeSummary }}</p>
        <div v-if="scopeOptions.length" class="scope-options">
          <div v-for="option in scopeOptions" :key="option.label" class="scope-option">
            <div class="scope-option-title">
              <strong>{{ option.label }}</strong>
              <span v-if="option.recommended" class="chip-sm">Recommended</span>
            </div>
            <p v-if="option.description">{{ option.description }}</p>
            <small v-if="option.estimated_slot_count">Estimated slots: {{ option.estimated_slot_count }}</small>
          </div>
        </div>
      </section>

      <section class="card">
        <span class="section-label">Collection match insight</span>
        <div class="info-grid mt-3">
          <div class="info-card">
            <span class="info-label">Matched estimate</span>
            <strong>{{ estimatedFilled }} / {{ estimatedTotal }}</strong>
          </div>
          <div class="info-card">
            <span class="info-label">Verified slots</span>
            <strong>{{ verifiedSlotCount }}</strong>
          </div>
          <div class="info-card">
            <span class="info-label">Needs review</span>
            <strong>{{ unverifiedSlotCount }}</strong>
          </div>
        </div>
        <p v-if="preMatchNotes" class="description mt-3">{{ preMatchNotes }}</p>
        <p v-else class="description mt-3">
          The workflow uses existing collection data to estimate which proposed slots are already represented before you approve the set.
        </p>
      </section>

      <section class="card">
        <div class="section-heading">
          <span class="section-label">Proposed roster</span>
          <span class="text-sm text-text-muted">{{ groupByLabel }}</span>
        </div>
        <div class="slot-groups">
          <div v-for="group in groupedSlots" :key="group.name" class="slot-group">
            <h3>{{ group.name }}</h3>
            <div class="slot-list">
              <article v-for="slot in group.slots" :key="slot.id" class="slot-card">
                <div>
                  <strong>{{ slot.label }}</strong>
                  <p v-if="slot.sourceNote">{{ slot.sourceNote }}</p>
                  <p v-else-if="slot.validationNote">{{ slot.validationNote }}</p>
                </div>
                <span class="chip-sm" :class="slot.verificationStatus === 'verified' ? 'verified' : 'unverified'">
                  {{ slot.verificationStatus }}
                </span>
              </article>
            </div>
          </div>
        </div>
      </section>

      <section v-if="transcriptSummary" class="card">
        <span class="section-label">Agent transcript summary</span>
        <p class="description">{{ transcriptSummary }}</p>
      </section>

      <section class="card action-card">
        <div>
          <span class="section-label">Human review</span>
          <p>Approve creates the Agentic set roster. Matching is automatic from your collection; no manual coin add/remove is used.</p>
        </div>
        <div class="action-buttons">
          <button class="btn btn-primary" :disabled="proposal.status !== 'pending' || approving" @click="approveProposal">
            {{ approving ? 'Approving...' : 'Approve' }}
          </button>
          <button class="btn btn-secondary" :disabled="proposal.status !== 'pending' || savingEdit" @click="openEditModal">Edit</button>
          <button class="btn btn-secondary" :disabled="proposal.status !== 'pending' || regenerating" @click="openRegenerateModal">Regenerate</button>
          <button class="btn btn-danger" :disabled="proposal.status !== 'pending' || rejecting" @click="rejectProposal">
            {{ rejecting ? 'Rejecting...' : 'Reject' }}
          </button>
        </div>
      </section>
    </div>

    <div v-if="showEditModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60 p-4" @click.self="showEditModal = false">
      <div class="card modal-card">
        <h2 class="mt-0">Edit Proposal</h2>
        <form class="modal-form" @submit.prevent="saveProposalEdits">
          <div class="form-group">
            <label for="proposalName" class="form-label">Name</label>
            <input id="proposalName" v-model="editForm.proposedName" class="form-input" maxlength="80" required />
          </div>
          <div class="form-group">
            <label for="proposalDescription" class="form-label">Description</label>
            <textarea id="proposalDescription" v-model="editForm.description" class="form-input" rows="3" />
          </div>
          <div class="form-group">
            <label for="proposalScope" class="form-label">Selected scope</label>
            <input id="proposalScope" v-model="editForm.selectedScope" class="form-input" />
          </div>
          <div class="form-group">
            <label for="proposalColor" class="form-label">Color</label>
            <input id="proposalColor" v-model="editForm.color" type="color" class="form-input h-11 cursor-pointer p-1" />
          </div>

          <div class="slot-editor">
            <span class="section-label">Roster slots</span>
            <article v-for="(slot, index) in editForm.slots" :key="slot.localId" class="slot-edit-card">
              <div class="form-group">
                <label :for="`slotLabel-${slot.localId}`" class="form-label">Slot {{ index + 1 }}</label>
                <input :id="`slotLabel-${slot.localId}`" v-model="slot.label" class="form-input" required />
              </div>
              <div class="slot-edit-grid">
                <div class="form-group">
                  <label :for="`slotGroup-${slot.localId}`" class="form-label">Group</label>
                  <input :id="`slotGroup-${slot.localId}`" v-model="slot.group" class="form-input" />
                </div>
                <div class="form-group">
                  <label :for="`slotStatus-${slot.localId}`" class="form-label">Verification</label>
                  <select :id="`slotStatus-${slot.localId}`" v-model="slot.verificationStatus" class="form-select">
                    <option value="verified">Verified</option>
                    <option value="unverified">Unverified</option>
                  </select>
                </div>
              </div>
              <div class="form-group">
                <label :for="`slotNote-${slot.localId}`" class="form-label">Review note</label>
                <input :id="`slotNote-${slot.localId}`" v-model="slot.validationNote" class="form-input" />
              </div>
            </article>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showEditModal = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="savingEdit">{{ savingEdit ? 'Saving...' : 'Save Edits' }}</button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="showRegenerateModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60 p-4" @click.self="showRegenerateModal = false">
      <div class="card modal-card">
        <h2 class="mt-0">Regenerate Proposal</h2>
        <form class="modal-form" @submit.prevent="regenerateProposal">
          <div class="form-group">
            <label for="regenerateFeedback" class="form-label">Feedback for the agentic workflow</label>
            <textarea
              id="regenerateFeedback"
              v-model="regenerateFeedback"
              class="form-input"
              rows="5"
              required
              placeholder="Tell the agents what to change before they build a new proposal."
            />
          </div>
          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showRegenerateModal = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="regenerating || !regenerateFeedback.trim()">
              {{ regenerating ? 'Submitting...' : 'Regenerate' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { approveSetProposal, getApiErrorMessage, getSetProposal, regenerateSetProposal as regenerateSetProposalApi, rejectSetProposal as rejectSetProposalApi, updateSetProposal } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import type { SetProposal, SetProposalSlot, UpdateSetProposalRequest } from '@/types'

const route = useRoute()
const router = useRouter()
const { showAlert, showConfirm } = useDialog()
const loading = ref(true)
const approving = ref(false)
const rejecting = ref(false)
const savingEdit = ref(false)
const regenerating = ref(false)
const error = ref('')
const proposal = ref<SetProposal | null>(null)
const showEditModal = ref(false)
const showRegenerateModal = ref(false)
const regenerateFeedback = ref('')
const editForm = ref<{
  proposedName: string
  description: string
  color: string
  selectedScope: string
  slots: Array<{
    localId: number
    label: string
    criteria: Record<string, unknown> | null
    group: string
    sortOrder: number
    verificationStatus: 'verified' | 'unverified'
    sourceNote: string
    validationNote: string
  }>
}>({
  proposedName: '',
  description: '',
  color: '#6b7280',
  selectedScope: '',
  slots: [],
})

const slots = computed(() => [...(proposal.value?.slots ?? [])].sort((a, b) => (a.sortOrder ?? 0) - (b.sortOrder ?? 0)))
const scopeSummary = computed(() => proposal.value?.scopeOptions?.scopeSummary ?? '')
const scopeOptions = computed(() => proposal.value?.scopeOptions?.options ?? [])
const groupByLabel = computed(() => proposal.value?.scopeOptions?.groupBy ? `Grouped by ${proposal.value.scopeOptions.groupBy}` : 'Grouped by proposal')
const estimatedFilled = computed(() => proposal.value?.preMatchSummary?.estimatedFilled ?? 0)
const estimatedTotal = computed(() => proposal.value?.preMatchSummary?.estimatedTotal ?? slots.value.length)
const preMatchNotes = computed(() => proposal.value?.preMatchSummary?.notes ?? '')
const transcriptSummary = computed(() => proposal.value?.rosterPayload?.transcriptSummary ?? proposal.value?.run?.transcriptSummary ?? '')
const proposalIssue = computed(() => proposal.value?.errorMessage ?? proposal.value?.run?.errorMessage ?? '')
const verifiedSlotCount = computed(() => slots.value.filter((slot) => slot.verificationStatus === 'verified').length)
const unverifiedSlotCount = computed(() => Math.max(0, slots.value.length - verifiedSlotCount.value))

const groupedSlots = computed(() => {
  const groups = new Map<string, SetProposalSlot[]>()
  for (const slot of slots.value) {
    const name = slot.group || 'Roster'
    groups.set(name, [...(groups.get(name) ?? []), slot])
  }
  return Array.from(groups.entries()).map(([name, groupSlots]) => ({ name, slots: groupSlots }))
})

async function loadProposal() {
  loading.value = true
  error.value = ''
  try {
    const id = Number(route.params.id)
    const res = await getSetProposal(id)
    proposal.value = res.data
  } catch (err) {
    error.value = getApiErrorMessage(err) || 'This proposal could not be loaded.'
  } finally {
    loading.value = false
  }
}

async function rejectProposal() {
  if (!proposal.value) return
  const confirmed = await showConfirm('Reject this Agentic set proposal? No set will be created.', {
    title: 'Reject Proposal',
    confirmLabel: 'Reject',
    variant: 'danger',
  })
  if (!confirmed) return
  rejecting.value = true
  try {
    await rejectSetProposalApi(proposal.value.id, 'Rejected during human review')
    await showAlert('The proposal was rejected. No set was created.', { title: 'Proposal Rejected' })
    router.push('/sets')
  } catch (err) {
    await showAlert(getApiErrorMessage(err) || 'Failed to reject proposal.', { title: 'Reject Failed' })
  } finally {
    rejecting.value = false
  }
}

function openEditModal() {
  if (!proposal.value) return
  editForm.value = {
    proposedName: proposal.value.proposedName,
    description: proposal.value.description ?? '',
    color: proposal.value.color || '#6b7280',
    selectedScope: proposal.value.selectedScope ?? '',
    slots: slots.value.map((slot, index) => ({
      localId: slot.id || index,
      label: slot.label,
      criteria: slot.criteria ?? null,
      group: slot.group ?? '',
      sortOrder: slot.sortOrder ?? index,
      verificationStatus: slot.verificationStatus,
      sourceNote: slot.sourceNote ?? '',
      validationNote: slot.validationNote ?? '',
    })),
  }
  showEditModal.value = true
}

async function saveProposalEdits() {
  if (!proposal.value) return
  savingEdit.value = true
  try {
    const payload: UpdateSetProposalRequest = {
      proposedName: editForm.value.proposedName,
      description: editForm.value.description,
      color: editForm.value.color,
      selectedScope: editForm.value.selectedScope,
      slots: editForm.value.slots.map((slot, index) => ({
        label: slot.label,
        criteria: slot.criteria,
        group: slot.group,
        sortOrder: slot.sortOrder || index,
        verificationStatus: slot.verificationStatus,
        sourceNote: slot.sourceNote,
        validationNote: slot.validationNote,
      })),
    }
    const res = await updateSetProposal(proposal.value.id, payload)
    proposal.value = res.data
    showEditModal.value = false
    await showAlert('Your proposal edits were saved.', { title: 'Proposal Updated' })
  } catch (err) {
    await showAlert(getApiErrorMessage(err) || 'Failed to update proposal.', { title: 'Update Failed' })
  } finally {
    savingEdit.value = false
  }
}

function openRegenerateModal() {
  regenerateFeedback.value = ''
  showRegenerateModal.value = true
}

async function regenerateProposal() {
  if (!proposal.value) return
  regenerating.value = true
  try {
    await regenerateSetProposalApi(proposal.value.id, { feedback: regenerateFeedback.value })
    showRegenerateModal.value = false
    await showAlert('A new Agentic set proposal is being prepared. You will receive a notification when it is ready.', { title: 'Regeneration Submitted' })
    router.push('/sets')
  } catch (err) {
    await showAlert(getApiErrorMessage(err) || 'Failed to regenerate proposal.', { title: 'Regenerate Failed' })
  } finally {
    regenerating.value = false
  }
}

async function approveProposal() {
  if (!proposal.value) return
  const confirmed = await showConfirm('Approve this Agentic set proposal and create the set roster?', {
    title: 'Approve Proposal',
    confirmLabel: 'Approve',
  })
  if (!confirmed) return
  approving.value = true
  try {
    const res = await approveSetProposal(proposal.value.id)
    await showAlert('The Agentic set has been created.', { title: 'Set Created' })
    router.push(`/sets/${res.data.set.id}`)
  } catch (err) {
    await showAlert(getApiErrorMessage(err) || 'Failed to approve proposal.', { title: 'Approve Failed' })
  } finally {
    approving.value = false
  }
}

onMounted(loadProposal)
</script>

<style scoped>
.proposal-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.proposal-hero {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.proposal-title-row,
.section-heading,
.scope-option-title,
.action-card,
.action-buttons {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.proposal-title-row h2 {
  margin: 0.25rem 0 0;
}

.prompt,
.description,
.scope-option p,
.action-card p,
.slot-card p {
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.5;
}

.prompt {
  color: var(--text-primary);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 0.75rem;
}

.info-card,
.scope-option,
.issue-card,
.slot-card {
  padding: 0.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card-hover);
}

.info-card {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.issue-card {
  border-color: var(--border-accent);
}

.scope-options,
.slot-groups,
.slot-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.slot-group h3 {
  margin: 0 0 0.75rem;
}

.slot-card {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.verified {
  color: var(--accent-gold);
  border-color: var(--accent-gold);
}

.unverified {
  color: var(--text-muted);
}

.action-card {
  align-items: flex-start;
}

.modal-card {
  width: min(92vw, 760px);
  max-height: 88vh;
  overflow-y: auto;
  padding: 1.5rem;
}

.modal-form,
.slot-editor {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.slot-edit-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card-hover);
}

.slot-edit-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.75rem;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

@media (max-width: 640px) {
  .proposal-title-row,
  .section-heading,
  .action-card,
  .action-buttons,
  .slot-card,
  .modal-actions {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
