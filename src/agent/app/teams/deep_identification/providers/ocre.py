"""OCRE provider node (T062) — MVP-only typed stub.

Online Coins of the Roman Empire (numismatics.org) is explicitly deferred
(spec T155, "Later Provider Gates"): this node makes NO upstream client
call, NO SPARQL query, and NO scraping. It only ever emits a typed
`not_automated` evidence row with the reason Go supplied in the provider
catalog (e.g. "pending_license_validation").
"""

from app.models.requests import DeepProviderCatalogEntry
from app.models.responses import ProviderEvidence


def run(catalog_entry: DeepProviderCatalogEntry) -> ProviderEvidence:
    return ProviderEvidence(
        provider="ocre",
        status="not_automated",
        automatable=False,
        call_count=0,
        link_out=catalog_entry.link_out or "",
        attribution="OCRE (Online Coins of the Roman Empire) — not yet automated",
    )
