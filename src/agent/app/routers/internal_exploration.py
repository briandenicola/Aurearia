"""Internal-only browser exploration decision route."""

from fastapi import APIRouter, HTTPException, Request

from app.models.browser_exploration import DecisionRequest, DecisionResponse
from app.services.browser_exploration import ExplorationDecisionError, decide_browser_action

router = APIRouter(prefix="/internal/browser-exploration", tags=["internal-browser-exploration"])


def _provider_config(decision: DecisionRequest, http_request: Request) -> dict[str, str]:
    if decision.provider == "anthropic":
        return {
            "provider": decision.provider,
            "model": decision.model,
            "api_key": http_request.headers.get("x-ai-browser-anthropic-api-key", ""),
        }
    return {
        "provider": decision.provider,
        "model": decision.model,
        "ollama_url": http_request.headers.get("x-ai-browser-ollama-url", ""),
    }


@router.post("/decide", response_model=DecisionResponse, response_model_by_alias=True)
async def decide(request: DecisionRequest, http_request: Request) -> DecisionResponse:
    try:
        return await decide_browser_action(request, provider_config=_provider_config(request, http_request))
    except ExplorationDecisionError as exc:
        status = {
            "provider_rate_limited": 429,
            "model_output_invalid": 422,
            "provider_usage_invalid": 502,
            "provider_configuration_invalid": 400,
            "provider_unavailable": 502,
        }.get(exc.code, 502)
        raise HTTPException(status_code=status, detail=exc.code) from exc
