"""Shared agent test environment."""

import os

os.environ.setdefault("AGENT_INTERNAL_SERVICE_TOKEN", "test-agent-service-token")
os.environ.setdefault(
    "AGENT_TRUSTED_OUTBOUND_ORIGINS",
    "http://localhost:11434,http://localhost:8080,http://test-api:8080,http://test:8080",
)
os.environ.setdefault("AGENT_ALLOW_LOCAL_OUTBOUND", "true")
