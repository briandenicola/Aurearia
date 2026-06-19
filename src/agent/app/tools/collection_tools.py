"""Collection tools — HTTP wrappers for Go internal endpoints.

Each tool makes an authenticated HTTP POST request to the Go API's
/api/internal/tools/{operation} endpoints. The internal token is short-lived
(30s) and provides identity for user-scoped operations — NEVER send a
userID from Python (Principle XI + XII).
"""

import logging
from typing import Any

import httpx
from langchain_core.tools import StructuredTool
from pydantic import BaseModel, Field

from app.outbound import validate_outbound_url

logger = logging.getLogger(__name__)


class SearchMyCollectionInput(BaseModel):
    """Input schema for search_my_collection tool."""

    query: str = Field(description="Search query to filter user's coin collection")
    limit: int = Field(default=10, description="Maximum number of coins to return (default 10)")


class GetCoinInput(BaseModel):
    """Input schema for get_coin tool."""

    coin_id: int = Field(description="Coin ID from the user's collection")


class CollectionSummaryInput(BaseModel):
    """Input schema for collection_summary tool (no args)."""

    pass


class TopCoinsByValueInput(BaseModel):
    """Input schema for top_coins_by_value tool."""

    limit: int = Field(default=3, description="Number of top coins to return (default 3, max 10)")


class ProposeUpdateInput(BaseModel):
    """Input schema for propose_update tool."""

    coin_id: int = Field(description="Coin ID to update")
    changes: dict[str, Any] = Field(
        description="Dictionary of field changes (e.g. {'notes': 'New notes', 'grade': 'VF'})"
    )


class CommitUpdateInput(BaseModel):
    """Input schema for commit_update tool."""

    proposal_id: str = Field(description="Proposal ID returned by propose_update")
    token: str = Field(description="Proposal token returned by propose_update")
    confirm: bool = Field(description="Explicit confirmation (must be True)")


def build_collection_tools(tools_base_url: str, internal_token: str) -> list[StructuredTool]:
    """Build 6 LangChain StructuredTools calling Go internal endpoints.

    Args:
        tools_base_url: Base URL for internal tool endpoints (e.g., "http://localhost:8080")
                        Routes are at /api/internal/tools/{operation}
        internal_token: Short-lived internal JWT token (30s expiry)

    Returns:
        List of 6 StructuredTools for collection operations.
    """
    tools_base_url = validate_outbound_url(tools_base_url, "tools_base_url")
    headers = {"Authorization": f"Bearer {internal_token}"}
    timeout = httpx.Timeout(connect=5.0, read=20.0, write=5.0, pool=5.0)

    async def _http_post(operation: str, body: dict) -> dict:
        """Make authenticated POST request to internal tool endpoint."""
        url = f"{tools_base_url}/api/internal/tools/{operation}"
        try:
            async with httpx.AsyncClient(timeout=timeout) as client:
                logger.debug("[collection_tools] POST %s", url)
                resp = await client.post(url, json=body, headers=headers)
                resp.raise_for_status()
                return resp.json()
        except httpx.HTTPStatusError as e:
            error_detail = e.response.text if e.response else str(e)
            logger.error("[collection_tools] HTTP %d for %s: %s", e.response.status_code, operation, error_detail)
            return {"error": f"HTTP {e.response.status_code}: {error_detail}"}
        except httpx.TimeoutException:
            logger.error("[collection_tools] Timeout for %s", operation)
            return {"error": f"Request to {operation} timed out"}
        except Exception as e:
            logger.error("[collection_tools] Error calling %s: %s", operation, e)
            return {"error": str(e)}

    async def search_my_collection_fn(query: str, limit: int = 10) -> str:
        """Search the user's coin collection with flexible filters.

        Use this to find coins the user already owns. Searches across name,
        ruler, era, denomination, category, material, notes, and tags. Also use
        this for data-quality questions like "missing size", "missing diameter",
        "missing weight", "missing grade", "missing value", or "missing
        metadata"; size means diameter_mm / diameterMm.

        Args:
            query: Search query (e.g., "Roman silver", "Constantine", "denarius")
            limit: Maximum results to return (default 10)

        Returns:
            JSON string with array of matching coins, each with: id, name, category,
            era, ruler, denomination, material, grade, weightGrams, diameterMm,
            purchasePrice, currentValue, missingFields.
        """
        result = await _http_post("search_my_collection", {"query": query, "limit": limit})
        if "error" in result:
            return f"Error: {result['error']}"
        coins = result.get("coins", [])
        if not coins:
            return f"No coins found matching '{query}' in your collection."
        return f"Found {len(coins)} coin(s): {coins}"

    async def get_coin_fn(coin_id: int) -> str:
        """Get detailed information about a single coin from the user's collection.

        Args:
            coin_id: Numeric coin ID

        Returns:
            JSON string with coin details: id, name, category, era, ruler,
            denomination, material, grade, weightGrams, diameterMm, purchasePrice,
            currentValue, missingFields.
        """
        result = await _http_post("get_coin", {"coin_id": coin_id})
        if "error" in result:
            return f"Error: {result['error']}"
        coin = result.get("coin")
        if not coin:
            return f"Coin {coin_id} not found in your collection."
        return f"Coin details: {coin}"

    async def collection_summary_fn() -> str:
        """Get aggregate statistics for the user's entire collection.

        Returns:
            JSON string with summary: total_coins, total_value, total_invested,
            roi_percent, categories (dict of counts), materials (dict of counts),
            eras (list of {name, count}), rulers (list of {name, count}), and
            missingFields counts for data-quality gaps such as diameterMm and
            weightGrams.
        """
        result = await _http_post("collection_summary", {})
        if "error" in result:
            return f"Error: {result['error']}"
        summary = result.get("summary", {})
        return f"Collection summary: {summary}"

    async def top_coins_by_value_fn(limit: int = 3) -> str:
        """Get the top coins in the user's collection by current value.

        Args:
            limit: Number of top coins to return (default 3, max 10)

        Returns:
            JSON string with array of coins sorted by current_value descending.
        """
        result = await _http_post("top_coins_by_value", {"limit": limit})
        if "error" in result:
            return f"Error: {result['error']}"
        coins = result.get("coins", [])
        if not coins:
            return "No coins with values found in your collection."
        return f"Top {len(coins)} coin(s) by value: {coins}"

    async def propose_update_fn(coin_id: int, changes: dict[str, Any]) -> str:
        """Create an update proposal for a coin (allowlisted fields only).

        Use this to propose changes to: notes, grade, tags.
        Returns a proposal preview with proposal_id and token for commit_update.

        Args:
            coin_id: Coin ID to update
            changes: Dictionary of field changes (e.g., {"notes": "Updated note", "grade": "VF"})

        Returns:
            JSON string with proposal: proposal_id, token, coin_id, changes,
            before (original values), after (proposed values), expires_at.
            User MUST confirm before calling commit_update.
        """
        result = await _http_post("propose_update", {"coin_id": coin_id, "changes": changes})
        if "error" in result:
            return f"Error: {result['error']}"
        proposal = result.get("proposal", {})
        return (
            f"Proposal created: {proposal}. "
            "Please confirm these changes with the user before committing."
        )

    async def commit_update_fn(proposal_id: str, token: str, confirm: bool) -> str:
        """Commit a previously created update proposal with user confirmation.

        Args:
            proposal_id: Proposal ID from propose_update
            token: Proposal token from propose_update
            confirm: Must be True (explicit confirmation required)

        Returns:
            JSON string with result: status, coin_id, updated_fields.
        """
        if not confirm:
            return "Error: User confirmation required. Set confirm=True to commit."
        result = await _http_post("commit_update", {
            "proposal_id": proposal_id,
            "token": token,
            "confirm": confirm,
        })
        if "error" in result:
            return f"Error: {result['error']}"
        commit_result = result.get("result", {})
        return f"Update committed: {commit_result}"

    # Build StructuredTools with Pydantic schemas
    return [
        StructuredTool.from_function(
            coroutine=search_my_collection_fn,
            name="search_my_collection",
            description=search_my_collection_fn.__doc__ or "",
            args_schema=SearchMyCollectionInput,
        ),
        StructuredTool.from_function(
            coroutine=get_coin_fn,
            name="get_coin",
            description=get_coin_fn.__doc__ or "",
            args_schema=GetCoinInput,
        ),
        StructuredTool.from_function(
            coroutine=collection_summary_fn,
            name="collection_summary",
            description=collection_summary_fn.__doc__ or "",
            args_schema=CollectionSummaryInput,
        ),
        StructuredTool.from_function(
            coroutine=top_coins_by_value_fn,
            name="top_coins_by_value",
            description=top_coins_by_value_fn.__doc__ or "",
            args_schema=TopCoinsByValueInput,
        ),
        StructuredTool.from_function(
            coroutine=propose_update_fn,
            name="propose_update",
            description=propose_update_fn.__doc__ or "",
            args_schema=ProposeUpdateInput,
        ),
        StructuredTool.from_function(
            coroutine=commit_update_fn,
            name="commit_update",
            description=commit_update_fn.__doc__ or "",
            args_schema=CommitUpdateInput,
        ),
    ]
