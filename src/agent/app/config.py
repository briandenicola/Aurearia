"""Configuration loaded from environment variables.

Note: Most config (API keys, model, prompts) arrives per-request from the Go API.
These are service-level settings only.
"""

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    debug: bool = False
    log_level: str = "INFO"  # TRACE/DEBUG/INFO/WARN/ERROR
    internal_service_token: str = ""
    searxng_url: str = ""  # External SearXNG instance URL (required for Ollama mode)
    trusted_outbound_origins: str = ""
    allow_local_outbound: bool = False
    max_search_results: int = 10
    verification_timeout: int = 10
    max_supervisor_iterations: int = 25

    # 344-deep-agentic-coin-identification pipeline bounds (contracts/
    # agent-internal-contract.md §2, research.md Performance table). These
    # are service-level ceilings; Go additionally passes a per-request
    # `bounds` object that must never exceed these env-configured maximums.
    deep_max_concurrency: int = 2
    deep_max_providers: int = 4
    deep_provider_timeout: int = 45
    deep_total_timeout: int = 280
    deep_recursion_limit: int = 12

    model_config = {"env_prefix": "AGENT_"}


settings = Settings()
