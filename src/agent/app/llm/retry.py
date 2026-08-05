"""Retry wrapper for LLM invocations."""

import logging

from tenacity import retry, retry_if_exception, stop_after_attempt, wait_exponential

logger = logging.getLogger(__name__)

RETRYABLE_PATTERNS = [
    "rate limit", "rate_limit", "429", "500", "502", "503", "504",
    "timeout", "connection", "overloaded", "capacity",
]


def _is_retryable(exc: BaseException) -> bool:
    """Check if an exception is retryable (transient/rate-limit)."""
    exc_str = str(exc).lower()
    return any(p in exc_str for p in RETRYABLE_PATTERNS)


def _log_cache_usage(response) -> None:
    """Log token/cache usage metrics when present on model responses."""
    usage: dict = {}

    response_metadata = getattr(response, "response_metadata", None)
    if isinstance(response_metadata, dict):
        metadata_usage = response_metadata.get("usage")
        if isinstance(metadata_usage, dict):
            usage.update(metadata_usage)

    usage_metadata = getattr(response, "usage_metadata", None)
    if isinstance(usage_metadata, dict):
        usage.update(usage_metadata)

    if not usage:
        return

    input_tokens = usage.get("input_tokens")
    output_tokens = usage.get("output_tokens")
    cache_write_tokens = usage.get("cache_creation_input_tokens")
    cache_read_tokens = usage.get("cache_read_input_tokens")

    if all(metric is None for metric in (input_tokens, output_tokens, cache_write_tokens, cache_read_tokens)):
        return

    logger.info(
        "LLM usage input_tokens=%s output_tokens=%s cache_creation_input_tokens=%s cache_read_input_tokens=%s",
        input_tokens,
        output_tokens,
        cache_write_tokens,
        cache_read_tokens,
    )


@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=2, max=30),
    retry=retry_if_exception(_is_retryable),
    before_sleep=lambda retry_state: logger.warning(
        "LLM call failed (attempt %d): %s — retrying...",
        retry_state.attempt_number,
        retry_state.outcome.exception(),
    ),
    reraise=True,
)
async def ainvoke_with_retry(model, messages: list, **kwargs):
    """Invoke an LLM model with automatic retry on transient failures."""
    response = await model.ainvoke(messages, **kwargs)
    _log_cache_usage(response)
    return response
