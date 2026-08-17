/**
 * Feature 351 RD-3: draft-field acceptance is **confidence-driven, not
 * source-driven**. A proposed field defaults to accepted once its confidence
 * reaches this threshold, regardless of whether the value came from the
 * image-only vision hypothesis or a provider citation (FR-021, FR-026).
 * This reverses the earlier "image-only fields opt-in" default.
 *
 * T120: this is the single named constant every acceptance call site must
 * consume — it must never appear as a bare literal anywhere else.
 */
export const DEEP_PROPOSAL_ACCEPTANCE_THRESHOLD = 0.7

export interface DeepProposalAcceptanceInput {
  accepted: boolean | null
  confidence?: number
}

/** True once `confidence` alone crosses the default-acceptance threshold. */
export function isDeepProposalConfidenceAccepted(confidence?: number): boolean {
  return typeof confidence === 'number' && confidence >= DEEP_PROPOSAL_ACCEPTANCE_THRESHOLD
}

/**
 * The effective accepted state to render (and ultimately persist): an
 * explicit owner decision (`true`/`false`) always wins over the default.
 * Only while no decision has been recorded yet (`accepted === null`) does
 * confidence decide the default (RD-3) — source never gates it.
 */
export function effectiveDeepProposalAcceptance(entry: DeepProposalAcceptanceInput): boolean {
  if (entry.accepted !== null && entry.accepted !== undefined) return entry.accepted
  return isDeepProposalConfidenceAccepted(entry.confidence)
}
