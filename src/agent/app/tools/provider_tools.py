"""Provider tools — HTTP wrappers for the Go deep-identification internal
tool endpoints (contracts/agent-internal-contract.md §7).

Mirrors app/tools/collection_tools.py's authenticated-POST pattern, but
these are called directly by the provider fan-out nodes
(app/teams/deep_identification/providers/) rather than exposed as
LangChain StructuredTools — there is no free-form tool-choice loop here,
only a fixed, closed set of provider calls the router already decided on.
"""

import logging

import httpx

from app.outbound import validate_outbound_url

logger = logging.getLogger(__name__)

_TIMEOUT_BUFFER_S = 5.0


class ProviderToolError(Exception):
    """Raised when a provider-tool HTTP call cannot be completed at all
    (network/timeout/unexpected status). Provider nodes catch this and
    convert it into a typed `failed`/`timed_out` ProviderEvidence row —
    it must never propagate out of a provider node.
    """


class ProviderToolsClient:
    """Authenticated client for the Go `/api/internal/tools/{numista_*,
    nomisma_search}` endpoints. One instance is built per job run from the
    request's `tools_base_url` + `internal_token` (job-scoped token,
    §1/§7) — never a userID, never a database handle.
    """

    def __init__(self, tools_base_url: str, internal_token: str, timeout_s: float) -> None:
        self._base_url = validate_outbound_url(tools_base_url, "tools_base_url")
        self._headers = {"Authorization": f"Bearer {internal_token}"}
        self._timeout = httpx.Timeout(
            connect=5.0, read=timeout_s + _TIMEOUT_BUFFER_S, write=5.0, pool=5.0
        )

    async def _post(self, operation: str, body: dict) -> dict:
        url = f"{self._base_url}/api/internal/tools/{operation}"
        try:
            async with httpx.AsyncClient(timeout=self._timeout) as client:
                resp = await client.post(url, json=body, headers=self._headers)
                resp.raise_for_status()
                return resp.json()
        except httpx.TimeoutException as exc:
            logger.warning("[provider_tools] timeout calling %s", operation)
            raise ProviderToolError(f"{operation} timed out") from exc
        except httpx.HTTPStatusError as exc:
            logger.warning(
                "[provider_tools] %s returned HTTP %d", operation, exc.response.status_code
            )
            raise ProviderToolError(f"{operation} returned HTTP {exc.response.status_code}") from exc
        except Exception as exc:
            logger.warning("[provider_tools] %s failed: %s", operation, type(exc).__name__)
            raise ProviderToolError(f"{operation} failed") from exc

    async def numista_search(self, query: str, limit: int = 5) -> dict:
        """POST /api/internal/tools/numista_search — {status, candidates, attribution}."""
        return await self._post("numista_search", {"query": query, "limit": limit})

    async def numista_detail(self, numista_id: int) -> dict:
        """POST /api/internal/tools/numista_detail — {status, candidate, identifier}."""
        return await self._post("numista_detail", {"id": numista_id})

    async def nomisma_search(self, query: str, limit: int = 5) -> dict:
        """POST /api/internal/tools/nomisma_search — {status, candidates, attribution}."""
        return await self._post("nomisma_search", {"query": query, "limit": limit})
