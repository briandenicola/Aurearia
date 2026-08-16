"""Merge/citation tests (T075) — contracts/agent-internal-contract.md §4-5.

Verifies deterministic claim ordering, citation host-allowlist validation,
and that non-automated/unavailable providers never emit `no_match`.
"""

import random

from app.models.responses import ProviderClaim, ProviderEvidence
from app.teams.deep_identification.evaluator import detect_disagreements
from app.teams.deep_identification.merge import sort_claims, validate_citations
from app.teams.deep_identification.providers import ngc, ocre, rpc


def test_validate_citations_drops_off_allowlist_host():
    claims = [
        ProviderClaim(field="mint", value="Rome", confidence=0.8, citation="https://en.numista.com/catalogue/123"),
        ProviderClaim(field="issuer", value="Trajan", confidence=0.6, citation="https://evil.example.com/fake"),
    ]

    valid, dropped = validate_citations("numista", claims)

    assert len(valid) == 1
    assert valid[0].field == "mint"
    assert dropped == 1


def test_validate_citations_allows_all_documented_hosts():
    hosts = {
        "numista": "https://en.numista.com/catalogue/1",
        "nomisma": "https://nomisma.org/id/rome",
        "ngc": "https://www.ngccoin.com/verify/1",
        "ocre": "https://numismatics.org/ocre/id/1",
        "rpc": "https://rpc.ashmus.ox.ac.uk/id/1",
    }
    for provider, citation in hosts.items():
        claims = [ProviderClaim(field="x", value="y", confidence=0.5, citation=citation)]
        valid, dropped = validate_citations(provider, claims)
        assert dropped == 0, f"{provider} should allow {citation}"
        assert len(valid) == 1


def test_sort_claims_is_deterministic_for_identical_inputs_in_any_order():
    row_numista = ProviderEvidence(
        provider="numista",
        status="contributed",
        automatable=True,
        claims=[
            ProviderClaim(field="mint", value="Rome", confidence=0.6, citation="https://en.numista.com/c/1"),
        ],
    )
    row_nomisma = ProviderEvidence(
        provider="nomisma",
        status="contributed",
        automatable=True,
        claims=[
            ProviderClaim(field="mint", value="Rome", confidence=0.9, citation="https://nomisma.org/id/1"),
        ],
    )

    order_a = sort_claims([row_numista, row_nomisma])
    order_b = sort_claims([row_nomisma, row_numista])

    fields_a = [(field, row.provider, claim.value) for field, row, claim in order_a]
    fields_b = [(field, row.provider, claim.value) for field, row, claim in order_b]
    assert fields_a == fields_b
    # Ordering key is (field, provider_rank, -confidence, citation) per §5 —
    # provider_rank takes precedence over confidence, so numista (rank 0)
    # sorts before nomisma (rank 1) even though nomisma's confidence is higher.
    assert fields_a[0][1] == "numista"


def test_sort_claims_shuffle_stability():
    rows = [
        ProviderEvidence(
            provider=p,
            status="contributed",
            automatable=True,
            claims=[ProviderClaim(field="issuer", value=f"v{i}", confidence=0.5, citation=f"https://en.numista.com/c/{i}")],
        )
        for i, p in enumerate(["numista"] * 5)
    ]
    baseline = sort_claims(rows)
    shuffled = list(rows)
    random.Random(42).shuffle(shuffled)
    result = sort_claims(shuffled)
    assert [c.value for _, _, c in baseline] == [c.value for _, _, c in result]


def test_conflicting_provider_claims_remain_an_unresolved_disagreement():
    rows = [
        ProviderEvidence(
            provider="numista",
            status="contributed",
            automatable=True,
            claims=[
                ProviderClaim(
                    field="mint",
                    value="Rome",
                    confidence=0.8,
                    citation="https://en.numista.com/catalogue/1",
                )
            ],
        ),
        ProviderEvidence(
            provider="nomisma",
            status="contributed",
            automatable=True,
            claims=[
                ProviderClaim(
                    field="mint",
                    value="Antioch",
                    confidence=0.7,
                    citation="https://nomisma.org/id/antioch",
                )
            ],
        ),
    ]

    disagreements, resolved_count = detect_disagreements(rows)

    assert resolved_count == 0
    assert len(disagreements) == 1
    assert disagreements[0].field == "mint"
    assert disagreements[0].resolution == "unresolved"
    assert {(ref.provider, ref.claim_index) for ref in disagreements[0].claim_refs} == {
        ("numista", 0),
        ("nomisma", 0),
    }


def test_ngc_ocre_rpc_never_emit_no_match():
    import asyncio

    from app.models.requests import DeepProviderCatalogEntry

    ngc_entry = DeepProviderCatalogEntry(provider="ngc", automatable=False, link_out="https://www.ngccoin.com/verify/")
    ocre_entry = DeepProviderCatalogEntry(provider="ocre", automatable=False, reason="pending_license_validation")
    rpc_entry = DeepProviderCatalogEntry(provider="rpc", automatable=False, reason="no_public_api")

    ngc_result = ngc.run(ngc_entry, None)
    # Feature 345: OCRE is now an automated node, but with its flag off
    # (automatable=False) it still short-circuits to not_automated with zero
    # tool calls and no tools client needed.
    ocre_result = asyncio.run(ocre.run(ocre_entry, None, None, ""))
    rpc_result = rpc.run(rpc_entry)

    assert ngc_result.status == "not_automated"
    assert ocre_result.status == "not_automated"
    assert ocre_result.call_count == 0
    assert rpc_result.status == "unavailable"
    for result in (ngc_result, ocre_result, rpc_result):
        assert result.status != "no_match"
        assert result.automatable is False
        assert result.claims == []
