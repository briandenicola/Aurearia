from app.models.requests import DeepProviderCatalogEntry, QuickEvidence, QuickEvidenceNGC
from app.teams.deep_identification.providers.ngc import run


def test_ngc_reuses_quick_lookup_certificate_link() -> None:
    evidence = run(
        DeepProviderCatalogEntry(
            provider="ngc",
            automatable=False,
            reason="terms_prohibit_automated_access",
            link_out="https://www.ngccoin.com/verify/",
        ),
        QuickEvidence(
            label_text="Maximinus I Ch VF 8232252-186",
            coin_fields={"ruler": "Maximinus I", "denomination": "Denarius"},
            confidence="high",
            ngc=QuickEvidenceNGC(
                cert_number="8232252-186",
                grade="Ch VF",
                lookup_url="https://www.ngccoin.com/certlookup/8232252186/NGCAncients/",
            ),
        ),
    )

    assert evidence.status == "not_automated"
    assert evidence.call_count == 0
    assert evidence.link_out == "https://www.ngccoin.com/certlookup/8232252186/NGCAncients/"
