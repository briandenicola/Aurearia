"""`CoinHypothesis` (contracts/vision-hypothesis.md §1) — the vision node's
typed output.

Today this is produced by a deterministic, LLM-free adapter over
`quick_evidence` (`app/teams/deep_identification/hypothesis.py`) — the
source the 351 spec explicitly designs as swappable. Phase 3/4 will replace
*that adapter's body* with the real single-vision-LLM-call output; every
consumer wired against this model / the `hypothesis` state key is
unaffected by that swap.
"""

from typing import Annotated

from pydantic import BaseModel, ConfigDict, Field, StringConstraints


class HypothesisField(BaseModel):
    """One typed, bounded-confidence value the hypothesis source supports
    for a single coin field. A field the source does not support is
    **omitted** entirely — never emitted with a guessed value at low
    confidence (spec FR-003).
    """

    model_config = ConfigDict(extra="forbid")

    value: Annotated[str, StringConstraints(min_length=1, max_length=1000)]
    confidence: float = Field(ge=0.0, le=1.0)


class CoinHypothesis(BaseModel):
    """Coin-field vocabulary shared with `proposed_fields` / the Go
    proposal allowlist (contract §1), so an image-only field lands in the
    draft through the *existing* allowlist — no new write surface. Carries
    no citation and is never converted into a `ProviderClaim` (FR-004).
    """

    model_config = ConfigDict(extra="forbid")

    ruler: HypothesisField | None = None
    denomination: HypothesisField | None = None
    material: HypothesisField | None = None
    mint: HypothesisField | None = None
    dateRange: HypothesisField | None = None
    era: HypothesisField | None = None
    obverseInscription: HypothesisField | None = None
    reverseInscription: HypothesisField | None = None
    obverseDescription: HypothesisField | None = None
    reverseDescription: HypothesisField | None = None
    diameterMm: HypothesisField | None = None
    weightGrams: HypothesisField | None = None

    # Short bounded prose for the narrative writer only — never itself a
    # proposed field value (contract §1).
    observations: Annotated[str, StringConstraints(max_length=500)] = ""
    legible: bool = False

    def fields(self) -> dict[str, HypothesisField]:
        """Every populated coin-field entry, excluding `observations`/`legible`."""
        return {
            name: value
            for name, value in (
                ("ruler", self.ruler),
                ("denomination", self.denomination),
                ("material", self.material),
                ("mint", self.mint),
                ("dateRange", self.dateRange),
                ("era", self.era),
                ("obverseInscription", self.obverseInscription),
                ("reverseInscription", self.reverseInscription),
                ("obverseDescription", self.obverseDescription),
                ("reverseDescription", self.reverseDescription),
                ("diameterMm", self.diameterMm),
                ("weightGrams", self.weightGrams),
            )
            if value is not None
        }

    def is_empty(self) -> bool:
        """True when the hypothesis supports nothing at all — the only
        state in which the no-evidence fallback narrative may fire
        (spec FR-020), together with zero provider contributions.
        """
        return not self.fields() and not self.observations
