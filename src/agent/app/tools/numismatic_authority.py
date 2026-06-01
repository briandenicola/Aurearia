"""Helpers for normalizing catalog references and linking authority records."""

from __future__ import annotations

from urllib.parse import quote_plus


def lookup_authority_uri(catalog: str, volume: str = "", number: str = "", uri: str = "") -> str:
    """Return a best-effort authority URI for a structured reference."""
    existing = (uri or "").strip()
    if existing:
        return existing

    cat = (catalog or "").strip().upper()
    vol = (volume or "").strip()
    num = (number or "").strip()
    if not cat or not num:
        return ""

    lowered_number = num.lower()

    if cat == "RIC":
        if lowered_number.startswith("ric."):
            return f"https://numismatics.org/ocre/id/{lowered_number}"
        query = quote_plus(f"RIC {vol} {num}".strip())
        return f"https://numismatics.org/ocre/results?q={query}"

    if cat == "RPC":
        if lowered_number.startswith("rpc."):
            return f"https://rpc.ashmus.ox.ac.uk/id/{lowered_number}"
        query = quote_plus(f"RPC {vol} {num}".strip())
        return f"https://rpc.ashmus.ox.ac.uk/search?q={query}"

    return ""


def normalize_candidate_references(raw_refs: list[dict] | None) -> list[dict]:
    """Normalize candidate references to the app's expected response schema."""
    refs = raw_refs or []
    normalized: list[dict] = []
    seen: set[tuple[str, str, str]] = set()

    for raw in refs:
        if not isinstance(raw, dict):
            continue

        catalog = str(raw.get("catalog", "")).strip().upper()
        volume = str(raw.get("volume", "")).strip()
        number = str(raw.get("number", "")).strip()
        uri = lookup_authority_uri(
            catalog=catalog,
            volume=volume,
            number=number,
            uri=str(raw.get("uri", "")).strip(),
        )

        if not catalog or not number:
            continue

        key = (catalog, volume, number)
        if key in seen:
            continue
        seen.add(key)

        normalized.append(
            {
                "catalog": catalog,
                "volume": volume,
                "number": number,
                "uri": uri,
            }
        )

    return normalized
