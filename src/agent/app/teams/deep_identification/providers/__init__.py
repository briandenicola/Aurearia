"""Deep-identification provider nodes (§6 of the internal contract).

Each module exposes a single `async def run(...) -> ProviderEvidence`
node. Automated nodes (numista, nomisma) call `app/tools/provider_tools.py`
only, never the upstream directly. Non-automated nodes (ngc, ocre, rpc)
make no network call at all — they always return a typed
`not_automated`/`unavailable` evidence row synchronously (FR-025).
"""
