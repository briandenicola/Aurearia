"""Contract tests for deep-identification provider-tool HTTP wrappers."""

from unittest.mock import AsyncMock, Mock, patch

import pytest

from app.outbound import settings
from app.tools.provider_tools import ProviderToolsClient


@pytest.mark.asyncio
async def test_ocre_search_posts_job_token_and_maps_response(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(settings, "trusted_outbound_origins", "https://api.example.com")
    client = ProviderToolsClient(
        tools_base_url="https://api.example.com",
        internal_token="test-job-token",
        timeout_s=45,
    )
    response_body = {
        "status": "ok",
        "candidates": [{"id": "ric.2.tr.123", "label": "RIC II Trajan 123"}],
        "attribution": {
            "source": "Online Coins of the Roman Empire",
            "license": "ODbL 1.0",
        },
    }

    with patch("app.tools.provider_tools.httpx.AsyncClient") as mock_client_type:
        response = Mock()
        response.raise_for_status = Mock(return_value=None)
        response.json = Mock(return_value=response_body)

        http_client = AsyncMock()
        http_client.__aenter__.return_value = http_client
        http_client.__aexit__.return_value = None
        http_client.post = AsyncMock(return_value=response)
        mock_client_type.return_value = http_client

        result = await client.ocre_search(
            ruler="trajan",
            denomination="denarius",
            mint="rome",
            material="silver",
            legend_tokens=["IMP", "TRAIANO"],
            limit=3,
        )

    http_client.post.assert_awaited_once_with(
        "https://api.example.com/api/internal/tools/ocre_search",
        json={
            "ruler": "trajan",
            "denomination": "denarius",
            "mint": "rome",
            "material": "silver",
            "legend_tokens": ["IMP", "TRAIANO"],
            "ocre_id": "",
            "limit": 3,
        },
        headers={"Authorization": "Bearer test-job-token"},
    )
    assert result == response_body
