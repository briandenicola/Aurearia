"""Tests for LLM retry utility."""

from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from app.llm.retry import ainvoke_with_retry


@pytest.fixture(autouse=True)
def _no_retry_wait(monkeypatch):
    """Eliminate tenacity wait time so tests run instantly."""
    from app.llm.retry import ainvoke_with_retry as fn

    fn.retry.wait = lambda *a, **kw: 0


@pytest.mark.asyncio
async def test_ainvoke_success_no_retry():
    """Successful call should not retry."""
    model = AsyncMock()
    model.ainvoke.return_value = "response"
    result = await ainvoke_with_retry(model, ["hello"])
    assert result == "response"
    assert model.ainvoke.call_count == 1


@pytest.mark.asyncio
async def test_ainvoke_retries_on_rate_limit():
    """Should retry on rate limit errors."""
    model = AsyncMock()
    model.ainvoke.side_effect = [
        Exception("rate limit exceeded"),
        "success",
    ]
    result = await ainvoke_with_retry(model, ["hello"])
    assert result == "success"
    assert model.ainvoke.call_count == 2


@pytest.mark.asyncio
async def test_ainvoke_retries_on_503():
    """Should retry on 503 service unavailable."""
    model = AsyncMock()
    model.ainvoke.side_effect = [
        Exception("503 service unavailable"),
        "ok",
    ]
    result = await ainvoke_with_retry(model, ["hello"])
    assert result == "ok"
    assert model.ainvoke.call_count == 2


@pytest.mark.asyncio
async def test_ainvoke_no_retry_on_non_transient_error():
    """Should NOT retry on non-transient errors."""
    model = AsyncMock()
    model.ainvoke.side_effect = ValueError("invalid prompt")
    with pytest.raises(ValueError, match="invalid prompt"):
        await ainvoke_with_retry(model, ["hello"])
    assert model.ainvoke.call_count == 1


@pytest.mark.asyncio
async def test_ainvoke_exhausts_retries():
    """Should raise after max 3 attempts on persistent transient errors."""
    model = AsyncMock()
    model.ainvoke.side_effect = Exception("503 service unavailable")
    with pytest.raises(Exception, match="503"):
        await ainvoke_with_retry(model, ["hello"])
    assert model.ainvoke.call_count == 3


@pytest.mark.asyncio
async def test_ainvoke_passes_kwargs():
    """Should forward kwargs to model.ainvoke."""
    model = AsyncMock()
    model.ainvoke.return_value = "result"
    await ainvoke_with_retry(model, ["msg"], temperature=0.5)
    model.ainvoke.assert_called_once_with(["msg"], temperature=0.5)


@pytest.mark.asyncio
async def test_ainvoke_logs_cache_usage(caplog):
    """Should log usage metrics including Anthropic prompt-cache fields."""
    model = AsyncMock()
    model.ainvoke.return_value = SimpleNamespace(
        response_metadata={
            "usage": {
                "input_tokens": 100,
                "output_tokens": 40,
                "cache_creation_input_tokens": 80,
                "cache_read_input_tokens": 20,
            }
        }
    )

    with caplog.at_level("INFO"):
        await ainvoke_with_retry(model, ["hello"])

    assert "cache_creation_input_tokens=80" in caplog.text
    assert "cache_read_input_tokens=20" in caplog.text
