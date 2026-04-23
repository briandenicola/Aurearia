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
    return await model.ainvoke(messages, **kwargs)
