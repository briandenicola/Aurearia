"""Router tests (T044-T050) — contracts/agent-internal-contract.md §6.

Verifies: `route()` is a pure function with no LLM call, selection is
always a subset of the supplied catalog, `provider_override` is honored
outright, an `automatable: false` catalog entry is never present in
`selected`/`skipped` (i.e. never subject to router reasoning at all),
selection is inclusion-by-default (RD-7), OCRE is skipped only on a
*positive* non-Roman-Imperial era signal, every skip carries a reason, and
identical inputs produce byte-identical decisions (SC-006).
"""

from app.models.hypothesis import CoinHypothesis, HypothesisField
from app.models.requests import DeepIdentifyBounds, DeepProviderCatalogEntry, QuickEvidence
from app.teams.deep_identification.router import route


def _catalog(*entries: tuple[str, bool]) -> list[DeepProviderCatalogEntry]:
    return [DeepProviderCatalogEntry(provider=name, automatable=automatable) for name, automatable in entries]


def _bounds(max_providers: int = 5) -> DeepIdentifyBounds:
    return DeepIdentifyBounds(
        max_providers=max_providers,
        max_concurrency=3,
        provider_timeout_s=30,
        total_timeout_s=300,
        recursion_limit=25,
    )


def _hypothesis(**fields) -> CoinHypothesis:
    return CoinHypothesis(**{k: HypothesisField(value=v, confidence=0.8) for k, v in fields.items()})


def test_route_selects_only_from_catalog():
    catalog = _catalog(("numista", True), ("nomisma", True), ("ngc", False))

    decision = route(catalog, provider_override=[], bounds=_bounds())

    assert set(decision.selected) <= {"numista", "nomisma"}
    assert "ngc" not in decision.selected


def test_route_never_selects_non_automatable_provider():
    catalog = _catalog(("numista", True), ("ngc", False), ("ocre", False), ("rpc", False))

    decision = route(catalog, provider_override=[], bounds=_bounds())

    assert decision.selected == ["numista"]
    assert all(skip["provider"] not in ("ngc", "ocre", "rpc") for skip in decision.skipped)


def test_provider_override_bypasses_selection_reasoning():
    catalog = _catalog(("numista", True), ("nomisma", True))

    decision = route(catalog, provider_override=["nomisma"], bounds=_bounds())

    assert decision.selected == ["nomisma"]
    assert decision.rationale == "provider_override supplied by caller"


def test_provider_override_cannot_add_provider_outside_catalog():
    catalog = _catalog(("numista", True),)

    decision = route(catalog, provider_override=["nomisma"], bounds=_bounds())

    assert decision.selected == []


def test_route_respects_max_providers_truncation():
    catalog = _catalog(("numista", True), ("nomisma", True))

    decision = route(catalog, provider_override=[], bounds=_bounds(max_providers=1))

    assert len(decision.selected) == 1
    assert decision.selected[0] == "numista"  # PROVIDER_RANK tie-break
    assert any(s["provider"] == "nomisma" and "max_providers" in s["reason"] for s in decision.skipped)


def test_route_empty_catalog_selects_nothing():
    decision = route([], provider_override=[], bounds=_bounds())

    assert decision.selected == []
    assert decision.skipped == []


# --- T046 / RD-7: inclusion by default, OCRE skip rules -------------------


def test_empty_evidence_run_selects_all_automatable_providers_including_ocre():
    """The exact Maximinus trap: no evidence at all (no hypothesis fields,
    no quick evidence) must still select every automatable provider,
    including OCRE — absence of a Roman signal is not a non-Roman signal.
    """
    catalog = _catalog(("numista", True), ("nomisma", True), ("ocre", True))

    decision = route(catalog, provider_override=[], bounds=_bounds(), quick_evidence=None, hypothesis=None)

    assert set(decision.selected) == {"numista", "nomisma", "ocre"}
    assert decision.skipped == []


def test_positive_roman_signal_still_selects_ocre():
    catalog = _catalog(("numista", True), ("ocre", True))
    hypothesis = _hypothesis(era="ancient")

    decision = route(catalog, provider_override=[], bounds=_bounds(), hypothesis=hypothesis)

    assert "ocre" in decision.selected


def test_positive_non_roman_era_signal_skips_ocre_with_reason():
    catalog = _catalog(("numista", True), ("ocre", True))
    hypothesis = _hypothesis(era="ancient", coin_type="Greek stater")

    decision = route(catalog, provider_override=[], bounds=_bounds(), hypothesis=hypothesis)

    assert "ocre" not in decision.selected
    ocre_skip = next(s for s in decision.skipped if s["provider"] == "ocre")
    assert "non-Roman-Imperial era signal" in ocre_skip["reason"]
    assert "greek" in ocre_skip["reason"]


def test_non_roman_signal_from_quick_evidence_category_skips_ocre():
    catalog = _catalog(("numista", True), ("ocre", True))
    quick_evidence = QuickEvidence(coin_fields={"category": "Byzantine"})

    decision = route(catalog, provider_override=[], bounds=_bounds(), quick_evidence=quick_evidence)

    assert "ocre" not in decision.selected
    ocre_skip = next(s for s in decision.skipped if s["provider"] == "ocre")
    assert "byzantine" in ocre_skip["reason"]


def test_roman_category_signal_does_not_skip_ocre():
    catalog = _catalog(("numista", True), ("ocre", True))
    quick_evidence = QuickEvidence(coin_fields={"category": "Roman"})

    decision = route(catalog, provider_override=[], bounds=_bounds(), quick_evidence=quick_evidence)

    assert "ocre" in decision.selected


def test_route_never_selects_ocre_when_flag_off():
    # Flag off -> Go marks OCRE automatable=false; it must never appear in
    # selected or skipped (it takes the trivial not_automated fan-out path).
    catalog = _catalog(("numista", True), ("ocre", False))

    decision = route(catalog, provider_override=[], bounds=_bounds())

    assert "ocre" not in decision.selected
    assert all(skip["provider"] != "ocre" for skip in decision.skipped)


def test_ocre_override_cannot_bypass_flag_off():
    catalog = _catalog(("numista", True), ("ocre", False))

    decision = route(catalog, provider_override=["ocre"], bounds=_bounds())

    assert "ocre" not in decision.selected


def test_every_skip_carries_a_reason():
    catalog = _catalog(("numista", True), ("nomisma", True), ("ocre", True))
    hypothesis = _hypothesis(era="modern")

    decision = route(catalog, provider_override=[], bounds=_bounds(max_providers=1), hypothesis=hypothesis)

    assert decision.skipped
    assert all(isinstance(s["reason"], str) and s["reason"] for s in decision.skipped)


# --- T046 (RD-7): provider_override selects/deselects OCRE -----------------


def test_provider_override_forces_ocre_even_with_non_roman_signal():
    """override=["ocre"] selects OCRE and skips evidence reasoning entirely,
    even in a context where the evidence-driven rule would have skipped it.
    """
    catalog = _catalog(("numista", True), ("ocre", True))
    hypothesis = _hypothesis(coin_type="Greek stater")

    decision = route(catalog, provider_override=["ocre"], bounds=_bounds(), hypothesis=hypothesis)

    assert decision.selected == ["ocre"]
    assert any(s["provider"] == "numista" for s in decision.skipped)


def test_provider_override_omitting_ocre_prevents_it_from_running():
    catalog = _catalog(("numista", True), ("ocre", True))

    decision = route(catalog, provider_override=["numista"], bounds=_bounds())

    assert decision.selected == ["numista"]
    assert "ocre" not in decision.selected
    assert any(s["provider"] == "ocre" and "provider_override" in s["reason"] for s in decision.skipped)


# --- T048: determinism (SC-006) --------------------------------------------


def test_route_is_deterministic_across_identical_runs():
    catalog = _catalog(("numista", True), ("nomisma", True), ("ocre", True))
    hypothesis = _hypothesis(coin_type="Byzantine follis")
    quick_evidence = QuickEvidence(coin_fields={"category": "Byzantine"})

    first = route(catalog, provider_override=[], bounds=_bounds(max_providers=2),
                  quick_evidence=quick_evidence, hypothesis=hypothesis)
    second = route(catalog, provider_override=[], bounds=_bounds(max_providers=2),
                   quick_evidence=quick_evidence, hypothesis=hypothesis)

    assert first.selected == second.selected
    assert first.skipped == second.skipped
    assert first.rationale == second.rationale


# --- T049 (B4): router_selected frame carries a non-empty skipped[] --------


def test_router_decision_skipped_is_populated_when_a_provider_is_skipped():
    catalog = _catalog(("numista", True), ("ocre", True))
    hypothesis = _hypothesis(coin_type="Islamic dirham")

    decision = route(catalog, provider_override=[], bounds=_bounds(), hypothesis=hypothesis)

    assert decision.skipped != []
    assert decision.skipped[0]["provider"] == "ocre"
    assert decision.skipped[0]["reason"]
