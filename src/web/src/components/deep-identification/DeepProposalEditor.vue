<template>
  <section class="grid min-w-0 gap-4 overflow-hidden" aria-label="Deep Analysis proposal">
    <h3 class="m-0 text-lg font-semibold text-text-primary">Review and save</h3>
    <p class="m-0 text-sm text-text-secondary">
      Accept the details you want to keep. Nothing is saved until you use the action below.
    </p>
    <ul class="m-0 grid gap-3 p-0" style="list-style: none;">
      <li
        v-for="name in fieldNames"
        :key="name"
        class="grid min-w-0 gap-2 overflow-hidden rounded-sm border border-border-subtle bg-card p-3"
      >
        <div class="flex flex-wrap items-baseline justify-between gap-2">
          <span class="text-sm font-semibold uppercase tracking-[0.04em] text-text-primary">{{ fieldLabel(name) }}</span>
          <span v-if="entryOf(name).ownerEdited" class="text-xs font-semibold uppercase tracking-[0.05em] text-byzantine">
            Edited by you
          </span>
          <span v-else class="text-xs font-semibold uppercase tracking-[0.05em] text-gold">AI proposed</span>
        </div>

        <p class="m-0 break-words text-sm text-text-secondary [overflow-wrap:anywhere]">
          AI value: <span class="font-medium text-text-primary">{{ displayValue(entryOf(name).proposed) }}</span>
        </p>

        <OCREAttribution
          v-if="ocreCitationFor(name)"
          :uri="ocreCitationFor(name)"
        />

        <label class="grid gap-1 text-sm text-text-secondary" :for="`deep-proposal-field-${name}`">
          Your value
          <textarea
            v-if="name === 'notes'"
            :id="`deep-proposal-field-${name}`"
            class="min-h-[132px] min-w-0 resize-y rounded-sm border border-border-subtle bg-background px-2 py-1 text-text-primary"
            :value="ownerValue(name)"
            @input="onOwnerValueInput(name, $event)"
          ></textarea>
          <input
            v-else
            :id="`deep-proposal-field-${name}`"
            type="text"
            class="min-h-[44px] rounded-sm border border-border-subtle bg-background px-2 py-1 text-text-primary"
            :value="ownerValue(name)"
            @input="onOwnerValueInput(name, $event)"
          >
        </label>

        <div class="flex items-center gap-2" role="group" :aria-label="`${fieldLabel(name)} decision`">
          <button
            type="button"
            class="inline-flex min-h-[44px] items-center rounded-full border px-3 py-1 text-sm font-medium transition-colors duration-150"
            :class="entryOf(name).accepted === true ? 'border-gold bg-gold text-white' : 'border-border-subtle text-text-secondary'"
            :aria-pressed="entryOf(name).accepted === true"
            @click="setAccepted(name, true)"
          >
            Accept
          </button>
          <button
            type="button"
            class="inline-flex min-h-[44px] items-center rounded-full border px-3 py-1 text-sm font-medium transition-colors duration-150"
            :class="entryOf(name).accepted === false ? 'border-byzantine bg-byzantine text-white' : 'border-border-subtle text-text-secondary'"
            :aria-pressed="entryOf(name).accepted === false"
            @click="setAccepted(name, false)"
          >
            Reject
          </button>
        </div>
      </li>
    </ul>

    <button
      type="button"
      class="inline-flex min-h-[44px] items-center justify-self-start rounded-full border border-gold bg-gold px-5 py-2 text-sm font-semibold text-white transition-opacity duration-150 disabled:cursor-not-allowed disabled:opacity-50"
      :disabled="confirmDisabled"
      @click="$emit('confirm')"
    >
      {{ applying ? applyingLabel : actionLabel }}
    </button>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { DeepProposal, DeepProposalFieldEntry } from '@/types'
import OCREAttribution from './OCREAttribution.vue'

const props = defineProps<{
  proposal: DeepProposal
  applying?: boolean
  actionLabel?: string
  applyingLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update-field', name: string, edit: { ownerValue?: unknown; accepted?: boolean | null }): void
  (e: 'confirm'): void
}>()

const emptyEntry: DeepProposalFieldEntry = { proposed: null, ownerEdited: false, ownerValue: null, accepted: null }

const fields = computed(() => props.proposal.fields)
const fieldNames = computed(() => Object.keys(props.proposal.fields).sort())
const actionLabel = computed(() => props.actionLabel ?? 'Apply to Coin')
const applyingLabel = computed(() => props.applyingLabel ?? 'Applying...')

function entryOf(name: string): DeepProposalFieldEntry {
  return fields.value[name] ?? emptyEntry
}

const confirmDisabled = computed(() => {
  if (props.applying) return true
  return !fieldNames.value.some((name) => entryOf(name).accepted === true)
})

function fieldLabel(name: string): string {
  return name.replace(/([A-Z])/g, ' $1').replace(/^./, (c) => c.toUpperCase())
}

function displayValue(value: unknown): string {
  if (value === null || value === undefined) return '—'
  return String(value)
}

function ownerValue(name: string): string {
  const entry = entryOf(name)
  const value = entry.ownerEdited ? entry.ownerValue : entry.proposed
  return value === null || value === undefined ? '' : String(value)
}

function onOwnerValueInput(name: string, event: Event) {
  const target = event.target as HTMLInputElement
  emit('update-field', name, { ownerValue: target.value })
}

function setAccepted(name: string, accepted: boolean) {
  emit('update-field', name, { accepted })
}

// Feature 345: an OCRE-sourced field (canonically `coin_type`) carries its
// attribution/license on the claim evidence, not on the coin row. Surface the
// dedicated OCRE attribution whenever a field's evidence cites a canonical
// numismatics.org OCRE type URI, so the ODbL/ANS credit appears exactly when
// (and only when) OCRE actually contributed to that proposed value.
function ocreCitationFor(name: string): string | null {
  const evidence = entryOf(name).evidence ?? []
  for (const claim of evidence) {
    const citation = claim?.citation
    if (!citation) continue
    try {
      const host = new URL(citation).host.toLowerCase()
      if (host === 'numismatics.org' && citation.toLowerCase().includes('/ocre/')) {
        return citation
      }
    } catch {
      // Non-parseable citation is never surfaced as an attribution link.
    }
  }
  return null
}
</script>
