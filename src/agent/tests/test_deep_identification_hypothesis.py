"""Hypothesis seam tests (Phase 8 — 351-vision-first-deep-identification).

Covers the deterministic quick-evidence -> `CoinHypothesis` adapter, the
`_build_proposed_fields` consumer this seam feeds (contracts/
vision-hypothesis.md §1, spec FR-019 through FR-025), and the Phase 3/4
structured vision path with its full degrade ladder (T019-T033).
"""

import asyncio

import pytest
from pydantic import ValidationError

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import DeepProviderCatalogEntry, LLMConfig, QuickEvidence, QuickEvidenceNGC
from app.models.responses import DeepSynthesis, ProviderCoverageEntry
from app.teams.deep_identification import hypothesis as hypothesis_module
from app.teams.deep_identification.hypothesis import build_hypothesis_from_quick_evidence, build_hypothesis_from_vision


def test_absent_quick_evidence_yields_empty_hypothesis():
    hypothesis = build_hypothesis_from_quick_evidence(None)

    assert hypothesis.is_empty()
    assert hypothesis.legible is False
    assert hypothesis.fields() == {}


def test_coin_fields_are_mapped_onto_the_shared_vocabulary():
    quick_evidence = QuickEvidence(
        confidence="high",
        coin_fields={
            "ruler": "Maximinus I",
            "denomination": "Denarius",
            "material": "Silver",
            "era": "ancient",
            "mint": "Rome",
        },
    )

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.legible is True
    assert hypothesis.ruler == HypothesisField(value="Maximinus I", confidence=0.75)
    assert hypothesis.denomination.value == "Denarius"
    assert hypothesis.material.value == "Silver"
    assert hypothesis.era.value == "ancient"
    assert hypothesis.mint.value == "Rome"


def test_era_and_material_values_that_do_not_canonicalize_are_dropped():
    """Go casts a proposed `era`/`material` value straight into
    `models.Era`/`models.Material` with no validation of its own
    (deep_identification_proposal.go::setCoinFieldFromProposalValue), so a
    value outside those fixed enum sets must never survive this adapter —
    it would silently write an invalid enum string into the coin row.
    """
    quick_evidence = QuickEvidence(
        confidence="medium",
        coin_fields={"era": "Roman Imperial", "material": "Billon"},
    )

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.era is None
    assert hypothesis.material is None
    # Nothing else supported either, and no NGC data -> genuinely empty.
    assert hypothesis.is_empty()


def test_material_canonicalizes_case_insensitively():
    quick_evidence = QuickEvidence(confidence="low", coin_fields={"material": "silver"})

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.material.value == "Silver"


def test_ngc_cert_and_grade_become_observations_even_without_coin_fields():
    quick_evidence = QuickEvidence(ngc=QuickEvidenceNGC(cert_number="1234567-001", grade="MS 65"))

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert "1234567-001" in hypothesis.observations
    assert "MS 65" in hypothesis.observations
    assert hypothesis.legible is True
    # Cert/grade are never proposed as coin fields themselves (no allowlist
    # entry for them) — they only ever reach the narrative via observations.
    assert hypothesis.fields() == {}


def test_blank_and_unknown_coin_fields_are_ignored():
    quick_evidence = QuickEvidence(coin_fields={"ruler": "   ", "grade": "MS65", "name": "Denarius"})

    hypothesis = build_hypothesis_from_quick_evidence(quick_evidence)

    assert hypothesis.is_empty()


def test_coin_hypothesis_fields_helper_excludes_observations_and_legible():
    hypothesis = CoinHypothesis(
        ruler=HypothesisField(value="Trajan", confidence=0.6),
        observations="some notes",
        legible=True,
    )

    fields = hypothesis.fields()

    assert set(fields) == {"ruler"}
    assert fields["ruler"].value == "Trajan"


# --- Vision path (Phase 3/4, T031-T033) -----------------------------------

_LLM_CONFIG = LLMConfig(provider="anthropic", api_key="k", model="claude-sonnet-5")
_IMAGE_CONTENTS = [{"type": "image", "source_type": "base64", "data": "abc", "mime_type": "image/png"}]


class _StructuredOK:
    """Fake structured runnable: first call returns a schema-conformant
    `CoinHypothesis` — the happy path, exactly one LLM call."""

    def __init__(self, hypothesis: CoinHypothesis):
        self.hypothesis = hypothesis
        self.calls = 0

    async def ainvoke(self, messages, **kwargs):
        self.calls += 1
        return {"raw": type("Resp", (), {"content": "ignored"})(), "parsed": self.hypothesis, "parsing_error": None}


class _StructuredAlwaysBadJSON:
    """Fake structured runnable: every call fails schema validation but
    returns malformed/prose-shaped raw content — exercises the
    retry-once -> prose-extraction -> deterministic-fallback rungs."""

    def __init__(self, raw_text: str):
        self.raw_text = raw_text
        self.calls = 0

    async def ainvoke(self, messages, **kwargs):
        self.calls += 1
        return {
            "raw": type("Resp", (), {"content": self.raw_text})(),
            "parsed": None,
            "parsing_error": ValueError("bad"),
        }


class _StructuredRaises:
    """Fake structured runnable: every call raises (LLM/network failure)."""

    def __init__(self):
        self.calls = 0

    async def ainvoke(self, messages, **kwargs):
        self.calls += 1
        raise RuntimeError("provider unavailable")


def test_vision_call_schema_conformant_result_is_used_directly(monkeypatch):
    parsed = CoinHypothesis(
        ruler=HypothesisField(value="Maximinus I (Thrax)", confidence=0.86),
        denomination=HypothesisField(value="Denarius", confidence=0.9),
        era=HypothesisField(value="ancient", confidence=0.9),
        legible=True,
    )
    fake = _StructuredOK(parsed)
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence=None))

    assert result.ruler.value == "Maximinus I (Thrax)"
    assert result.denomination.value == "Denarius"
    assert result.era.value == "ancient"
    assert fake.calls == 1  # happy path: exactly one LLM call, no retry


def test_vision_call_drops_unsupported_fields_never_guesses(monkeypatch):
    """FR-003: a field the images do not support must be absent — the
    schema itself enforces omission (fields are `| None`), so this asserts
    the normalization pass does not fabricate anything for an omitted key.
    """
    parsed = CoinHypothesis(ruler=HypothesisField(value="Trajan", confidence=0.7), legible=True)
    fake = _StructuredOK(parsed)
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence=None))

    assert set(result.fields()) == {"ruler"}
    assert result.mint is None
    assert result.denomination is None


def test_vision_call_confidence_bounds_are_enforced_by_the_schema():
    """`HypothesisField.confidence` is `Field(ge=0.0, le=1.0)` — a
    structured parse that violates that bound is a schema-validation
    failure by definition, which is exactly the failure mode the degrade
    ladder exists to handle (spec FR-006), never a value this module
    silently forwards.
    """
    with pytest.raises(ValidationError):
        HypothesisField(value="Trajan", confidence=1.5)


def test_vision_call_normalizes_era_and_material_dropping_non_canonical(monkeypatch):
    parsed = CoinHypothesis(
        era=HypothesisField(value="Roman Imperial", confidence=0.8),
        material=HypothesisField(value="silver", confidence=0.7),
        ruler=HypothesisField(value="Trajan", confidence=0.6),
        legible=True,
    )
    fake = _StructuredOK(parsed)
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence=None))

    # "Roman Imperial" does not canonicalize onto {ancient, medieval,
    # modern} and must be dropped, never forwarded as a Go enum value.
    assert result.era is None
    assert result.material.value == "Silver"
    assert result.ruler.value == "Trajan"


def test_vision_call_schema_failure_retries_once_then_prose_then_deterministic(monkeypatch):
    """T020/T032: schema-validation failure retries once (bounded — not a
    second *vision* call in the normal-cost sense, since it only fires on
    failure), then attempts prose extraction, then falls back to the
    deterministic quick-evidence hypothesis. The job-level exception count
    must stay zero in every branch (T021/T032)."""
    raw_text = 'Here is my analysis: {"ruler": "Hadrian", "denomination": {"value": "Sestertius", "confidence": 0.6}}'
    fake = _StructuredAlwaysBadJSON(raw_text)
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence=None))

    assert fake.calls == 2  # first attempt + the one bounded retry
    # Prose extraction recovered a usable hypothesis from the malformed JSON.
    assert result.ruler.value == "Hadrian"
    assert result.denomination.value == "Sestertius"


def test_vision_call_llm_raise_degrades_to_deterministic_quick_evidence(monkeypatch):
    fake = _StructuredRaises()
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    quick_evidence = QuickEvidence(confidence="high", coin_fields={"ruler": "Maximinus I"})
    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence))

    assert fake.calls == 1  # raise breaks the retry loop immediately
    assert result.ruler.value == "Maximinus I"  # deterministic fallback, not an exception


def test_vision_call_timeout_degrades_to_deterministic_quick_evidence(monkeypatch):
    class _StructuredTimesOut:
        async def ainvoke(self, messages, **kwargs):
            raise TimeoutError("vision call timed out")

    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: _StructuredTimesOut())

    quick_evidence = QuickEvidence(confidence="medium", coin_fields={"denomination": "Denarius"})
    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence))

    assert result.denomination.value == "Denarius"
    assert result.legible is True


def test_vision_call_empty_content_degrades_to_deterministic_quick_evidence(monkeypatch):
    fake = _StructuredAlwaysBadJSON(raw_text="")  # empty raw content, no prose to recover
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    quick_evidence = QuickEvidence(confidence="low", coin_fields={"mint": "Rome"})
    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence))

    assert result.mint.value == "Rome"


def test_vision_call_malformed_json_and_no_quick_evidence_yields_typed_empty(monkeypatch):
    fake = _StructuredAlwaysBadJSON(raw_text="not json at all, sorry")
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence=None))

    assert result.is_empty()
    assert result.legible is False


def test_no_image_contents_never_invokes_the_structured_model(monkeypatch):
    """When there is nothing to look at, don't even attempt the vision
    call — go straight to the deterministic fallback."""
    fake = _StructuredOK(CoinHypothesis(ruler=HypothesisField(value="should not be used", confidence=0.9)))
    monkeypatch.setattr(hypothesis_module, "get_structured_model", lambda config, schema: fake)

    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, image_contents=[], quick_evidence=None))

    assert fake.calls == 0
    assert result.is_empty()


def test_get_structured_model_bind_failure_degrades_to_deterministic(monkeypatch):
    def _raise(config, schema):
        raise ValueError("unknown provider")

    monkeypatch.setattr(hypothesis_module, "get_structured_model", _raise)

    quick_evidence = QuickEvidence(confidence="high", coin_fields={"ruler": "Trajan"})
    result = asyncio.run(build_hypothesis_from_vision(_LLM_CONFIG, _IMAGE_CONTENTS, quick_evidence))

    assert result.ruler.value == "Trajan"


# --- T033: `image` is not a provider --------------------------------------


def test_hypothesis_field_carries_no_citation():
    """The hypothesis is never converted into a `ProviderClaim` (spec
    FR-004) — structurally enforced by `HypothesisField` having no
    citation field at all, unlike `ProviderClaim`."""
    assert "citation" not in HypothesisField.model_fields


def test_image_is_not_a_valid_provider_name_in_coverage_or_catalog():
    """`image` may appear only as an `EvidenceRef.provider` string — it
    must be rejected everywhere `ProviderName` (a closed literal union) is
    used, including `ProviderCoverageEntry` and the Go-facing provider
    catalog/override lists.
    """
    with pytest.raises(ValidationError):
        ProviderCoverageEntry(provider="image", status="contributed")

    with pytest.raises(ValidationError):
        DeepProviderCatalogEntry(provider="image", automatable=True)


def test_image_hypothesis_is_additive_and_separate_from_provider_surfaces():
    """`DeepSynthesis.image_hypothesis` is the only place the hypothesis
    is carried in the terminal report — `coverage`/`attributions` never
    contain it, and it is never itself a `ProviderCoverageEntry`/
    `ProviderAttribution` (spec FR-004, contract §4)."""
    hypothesis = CoinHypothesis(ruler=HypothesisField(value="Trajan", confidence=0.7), legible=True)

    report = DeepSynthesis(
        narrative="test",
        coverage=[ProviderCoverageEntry(provider="numista", status="no_match")],
        image_hypothesis=hypothesis,
    )

    assert report.image_hypothesis is not None
    assert report.image_hypothesis.ruler.value == "Trajan"
    assert all(entry.provider != "image" for entry in report.coverage)
