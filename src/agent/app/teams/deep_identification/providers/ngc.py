"""NGC provider node (T061) — not_automated, link-out only.

NGC's Terms of Use prohibit automated access, so this node makes no live
NGC API call whatsoever (Principle IV, FR-025). It reuses any cert number
already extracted upstream by the F341 Quick Identify flow
(`quick_evidence.ngc`) purely to build a more specific link-out URL — it
never re-runs OCR or re-derives the cert number itself.
"""

from urllib.parse import urlencode

from app.models.requests import DeepProviderCatalogEntry, QuickEvidence
from app.models.responses import ProviderEvidence


def run(catalog_entry: DeepProviderCatalogEntry, quick_evidence: QuickEvidence | None) -> ProviderEvidence:
    link_out = catalog_entry.link_out or ""
    if link_out and quick_evidence and quick_evidence.ngc and quick_evidence.ngc.cert_number:
        separator = "&" if "?" in link_out else "?"
        link_out = f"{link_out}{separator}{urlencode({'certNumber': quick_evidence.ngc.cert_number})}"
    elif quick_evidence and quick_evidence.ngc and quick_evidence.ngc.lookup_url:
        link_out = link_out or quick_evidence.ngc.lookup_url

    return ProviderEvidence(
        provider="ngc",
        status="not_automated",
        automatable=False,
        call_count=0,
        link_out=link_out,
        attribution="Source: NGC (link-out only, terms prohibit automated access)",
    )
