"""Tests for collection tools HTTP wrappers."""

from unittest.mock import AsyncMock, Mock, patch

import httpx
import pytest
from langchain_core.tools import StructuredTool

from app.tools.collection_tools import build_collection_tools


@pytest.fixture
def mock_tools():
    """Build collection tools with test credentials."""
    return build_collection_tools(
        tools_base_url="http://test-api:8080",
        internal_token="test-token-12345",
    )


@pytest.mark.asyncio
async def test_build_collection_tools_returns_six_tools(mock_tools):
    """Verify build_collection_tools returns 6 StructuredTools."""
    assert len(mock_tools) == 6
    assert all(isinstance(tool, StructuredTool) for tool in mock_tools)

    tool_names = {tool.name for tool in mock_tools}
    expected = {
        "search_my_collection",
        "get_coin",
        "collection_summary",
        "top_coins_by_value",
        "propose_update",
        "commit_update",
    }
    assert tool_names == expected


@pytest.mark.asyncio
async def test_search_my_collection_makes_correct_request(mock_tools):
    """Verify search_my_collection tool makes correct HTTP request."""
    search_tool = next(t for t in mock_tools if t.name == "search_my_collection")

    with patch("app.tools.collection_tools.httpx.AsyncClient") as MockClient:
        # Response methods are SYNC in httpx, so use regular Mock
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json = Mock(return_value={"coins": [{"id": 1, "name": "Test Coin"}]})
        mock_response.raise_for_status = Mock(return_value=None)

        mock_client = AsyncMock()
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None
        mock_client.post = AsyncMock(return_value=mock_response)
        MockClient.return_value = mock_client

        result = await search_tool.ainvoke({"query": "Roman silver", "limit": 5})

        # Verify HTTP call
        mock_client.post.assert_called_once()
        call_args = mock_client.post.call_args
        assert call_args[0][0] == "http://test-api:8080/api/internal/tools/search_my_collection"
        assert call_args[1]["json"] == {"query": "Roman silver", "limit": 5}
        assert call_args[1]["headers"]["Authorization"] == "Bearer test-token-12345"

        # Verify result
        assert "Found 1 coin(s)" in result


@pytest.mark.asyncio
async def test_get_coin_handles_not_found(mock_tools):
    """Verify get_coin tool handles 404 correctly."""
    get_coin_tool = next(t for t in mock_tools if t.name == "get_coin")

    mock_response = httpx.Response(404, json={"error": "Coin not found"})
    mock_response.status_code = 404

    with patch("httpx.AsyncClient.post", new_callable=AsyncMock) as mock_post:
        mock_post.side_effect = httpx.HTTPStatusError(
            "404", request=None, response=mock_response
        )
        result = await get_coin_tool.ainvoke({"coin_id": 9999})

        # Should return error string, not raise
        assert "Error:" in result or "HTTP 404" in result


@pytest.mark.asyncio
async def test_collection_summary_no_args(mock_tools):
    """Verify collection_summary tool works with no arguments."""
    summary_tool = next(t for t in mock_tools if t.name == "collection_summary")

    with patch("app.tools.collection_tools.httpx.AsyncClient") as MockClient:
        # Response methods are SYNC in httpx, so use regular Mock
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json = Mock(return_value={"summary": {"total_coins": 10, "total_value": 1000.0}})
        mock_response.raise_for_status = Mock(return_value=None)

        mock_client = AsyncMock()
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None
        mock_client.post = AsyncMock(return_value=mock_response)
        MockClient.return_value = mock_client

        result = await summary_tool.ainvoke({})

        # Verify empty body
        call_args = mock_client.post.call_args
        assert call_args[1]["json"] == {}

        # Verify result
        assert "Collection summary:" in result


@pytest.mark.asyncio
async def test_propose_update_returns_proposal(mock_tools):
    """Verify propose_update returns proposal with token."""
    propose_tool = next(t for t in mock_tools if t.name == "propose_update")

    with patch("app.tools.collection_tools.httpx.AsyncClient") as MockClient:
        # Response methods are SYNC in httpx, so use regular Mock
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json = Mock(return_value={
            "proposal": {
                "proposal_id": "prop-123",
                "token": "token-abc",
                "changes": {"notes": "Updated"},
            }
        })
        mock_response.raise_for_status = Mock(return_value=None)

        mock_client = AsyncMock()
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None
        mock_client.post = AsyncMock(return_value=mock_response)
        MockClient.return_value = mock_client

        result = await propose_tool.ainvoke({
            "coin_id": 42,
            "changes": {"notes": "Updated"},
        })

        # Verify request body
        call_args = mock_client.post.call_args
        assert call_args[1]["json"] == {
            "coin_id": 42,
            "changes": {"notes": "Updated"},
        }

        # Verify result includes proposal data
        assert "prop-123" in result


@pytest.mark.asyncio
async def test_commit_update_requires_confirmation(mock_tools):
    """Verify commit_update requires explicit confirmation."""
    commit_tool = next(t for t in mock_tools if t.name == "commit_update")

    # Without confirm=True, should return error
    result = await commit_tool.ainvoke({
        "proposal_id": "prop-123",
        "token": "token-abc",
        "confirm": False,
    })

    assert "confirmation required" in result.lower()


@pytest.mark.asyncio
async def test_timeout_handling(mock_tools):
    """Verify tools handle timeout gracefully."""
    search_tool = next(t for t in mock_tools if t.name == "search_my_collection")

    with patch("app.tools.collection_tools.httpx.AsyncClient") as MockClient:
        mock_client = AsyncMock()
        mock_client.__aenter__.return_value = mock_client
        mock_client.__aexit__.return_value = None
        mock_client.post = AsyncMock(side_effect=httpx.TimeoutException("Request timed out"))
        MockClient.return_value = mock_client

        result = await search_tool.ainvoke({"query": "test"})

        # Should return error string, not raise
        assert "timed out" in result.lower()
