"""RPC provider node (T062) — MVP-only typed stub.

Roman Provincial Coinage Online (rpc.ashmus.ox.ac.uk) is explicitly
deferred (spec T156, "Later Provider Gates"): this node makes NO upstream
client call and NO scraping. It only ever emits a typed `unavailable`
evidence row with the reason Go supplied in the provider catalog (e.g.
"no_public_api").
"""

from app.models.requests import DeepProviderCatalogEntry
from app.models.responses import ProviderEvidence


def run(catalog_entry: DeepProviderCatalogEntry) -> ProviderEvidence:
    return ProviderEvidence(
        provider="rpc",
        status="unavailable",
        automatable=False,
        call_count=0,
        link_out=catalog_entry.link_out or "",
        attribution="RPC Online (Roman Provincial Coinage) — no public API",
    )
