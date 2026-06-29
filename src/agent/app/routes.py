"""API routes for the agent service."""

import logging

from fastapi import APIRouter
from fastapi.responses import StreamingResponse

from app.config import settings
from app.models.requests import (
    AlertDiscoveryRequest,
    AnalyzeRequest,
    AvailabilityCheckRequest,
    CoinSearchRequest,
    CoinShowSearchRequest,
    IntakeDraftRequest,
    PortfolioReviewRequest,
)
from app.models.responses import (
    AgentResponse,
    AlertDiscoveryResponse,
    AvailabilityCheckResponse,
    AvailabilityVerdict,
    IntakeDraftResponse,
)
from app.streaming import stream_graph_events
from app.supervisor import create_supervisor
from app.teams.availability_check import (
    AvailabilityVerdictParseError,
    create_availability_check_team,
    parse_verdicts,
)
from app.teams.coin_analysis import create_coin_analysis_team
from app.teams.coin_intake import generate_intake_draft
from app.teams.coin_search import discover_alert_candidates

logger = logging.getLogger(__name__)

_RECURSION_CONFIG = {"recursion_limit": settings.max_supervisor_iterations}

router = APIRouter(prefix="/api")

# Wishlist search alert discovery route anchor:
# specs/337-wishlist-search-alerts/contracts/agent-discovery-contract.md


@router.post("/search/alerts", response_model=AlertDiscoveryResponse)
async def search_alerts(request: AlertDiscoveryRequest):
    """Discover source-backed wishlist search alert candidates. Stateless; no persistence."""
    logger.info(
        "POST /search/alerts — provider=%s, model=%s, alert_id=%s, max=%d",
        request.llm.provider,
        request.llm.model,
        request.alert.alert_id,
        request.alert.max_candidates,
    )
    return await discover_alert_candidates(request)


def _build_messages(message: str, history: list | None = None, system_prompt: str = ""):
    """Convert request data into LangChain messages."""
    from langchain_core.messages import AIMessage, HumanMessage, SystemMessage

    messages = []
    if system_prompt:
        messages.append(SystemMessage(content=system_prompt))
    for msg in (history or []):
        if msg.role == "user":
            messages.append(HumanMessage(content=msg.content))
        elif msg.role == "assistant":
            messages.append(AIMessage(content=msg.content))
    messages.append(HumanMessage(content=message))
    return messages


@router.post("/search/coins")
async def search_coins(request: CoinSearchRequest):
    """Search for coins using multi-agent pipeline with verification. Streams SSE."""
    logger.info(
        "POST /search/coins — provider=%s, model=%s, message=%.80s",
        request.llm.provider, request.llm.model, request.message,
    )
    messages = _build_messages(request.message, request.history)
    graph = create_supervisor(
        request.llm,
        user_message=request.message,
        coin_search_prompt=request.coin_search_prompt,
        coin_shows_prompt=request.coin_shows_prompt,
        user_context=request.user,
        portfolio=request.portfolio,
        app_context=request.app_context,
        tools_base_url=request.tools_base_url,
        internal_token=request.internal_token,
    )

    async def event_stream():
        async for chunk in stream_graph_events(graph, {"messages": messages}, config=_RECURSION_CONFIG):
            yield chunk

    return StreamingResponse(event_stream(), media_type="text/event-stream")


@router.post("/search/shows")
async def search_shows(request: CoinShowSearchRequest):
    """Search for upcoming coin shows with date verification. Streams SSE."""
    logger.info(
        "POST /search/shows — provider=%s, model=%s, message=%.80s",
        request.llm.provider, request.llm.model, request.message,
    )
    messages = _build_messages(request.message, request.history)
    graph = create_supervisor(
        request.llm,
        user_message=request.message,
        coin_search_prompt=request.coin_search_prompt,
        coin_shows_prompt=request.coin_shows_prompt,
        user_context=request.user,
    )

    async def event_stream():
        async for chunk in stream_graph_events(graph, {"messages": messages}, config=_RECURSION_CONFIG):
            yield chunk

    return StreamingResponse(event_stream(), media_type="text/event-stream")


@router.post("/analyze", response_model=AgentResponse)
async def analyze_coin(request: AnalyzeRequest):
    """Analyze coin images using vision model. Returns structured response."""
    logger.info(
        "POST /analyze — provider=%s, model=%s, images=%d, side=%s",
        request.llm.provider, request.llm.model,
        len(request.images or []), request.side or "general",
    )
    graph = create_coin_analysis_team(
        llm_config=request.llm,
        coin=request.coin,
        images=request.images,
        side=request.side,
        custom_prompt=request.prompt,
    )
    # Don't pass image_contents, coin_context, or analysis_prompt in the
    # initial state — the team constructor captures them via closure and
    # the nodes use state.get(key, closure_default). Passing empty values
    # here would override the closure defaults.
    try:
        result = await graph.ainvoke({
            "messages": [],
            "raw_analysis": "",
            "formatted_analysis": "",
        })
    except Exception:
        logger.exception("Coin analysis graph execution failed")
        return AgentResponse(
            analysis="Unable to complete the analysis. Please try again with clearer, well-lit images of the coin."
        )
    analysis_text = result.get("formatted_analysis", "")
    if not analysis_text:
        # Fall back to messages
        msgs = result.get("messages", [])
        if msgs:
            analysis_text = msgs[-1].content if hasattr(msgs[-1], "content") else str(msgs[-1])

    return AgentResponse(analysis=analysis_text)


@router.post("/intake/draft", response_model=IntakeDraftResponse)
async def intake_draft(request: IntakeDraftRequest):
    """Generate a structured coin intake draft from image evidence."""
    logger.info(
        "POST /intake/draft — provider=%s, model=%s, images=%d, coin_card=%s",
        request.llm.provider,
        request.llm.model,
        len(request.images),
        "yes" if request.coin_card_image else "no",
    )
    return await generate_intake_draft(
        llm_config=request.llm,
        images=request.images,
        coin_card_image=request.coin_card_image,
    )


@router.post("/portfolio/review")
async def review_portfolio(request: PortfolioReviewRequest):
    """Review portfolio with live valuation. Streams SSE."""
    logger.info(
        "POST /portfolio/review — provider=%s, model=%s, message=%.80s",
        request.llm.provider, request.llm.model,
        request.message or "Analyze my portfolio",
    )
    prompt = request.valuation_prompt or "You are a numismatic portfolio analyst."
    messages = _build_messages(request.message or "Analyze my portfolio", request.history, prompt)
    graph = create_supervisor(
        request.llm,
        user_message=request.message or "Analyze my portfolio",
        user_context=request.user,
        portfolio=request.portfolio,
    )

    async def event_stream():
        async for chunk in stream_graph_events(graph, {"messages": messages}, config=_RECURSION_CONFIG):
            yield chunk

    return StreamingResponse(event_stream(), media_type="text/event-stream")


@router.post("/check-availability", response_model=AvailabilityCheckResponse)
async def check_availability(request: AvailabilityCheckRequest):
    """Check listing availability for multiple coin URLs. Returns structured verdicts."""
    logger.info(
        "POST /check-availability — provider=%s, model=%s, items=%d",
        request.llm.provider, request.llm.model, len(request.items),
    )

    if not request.items:
        return AvailabilityCheckResponse(results=[])

    graph = create_availability_check_team(request.llm)
    items_data = [{"url": item.url, "coin_name": item.coin_name} for item in request.items]

    try:
        result = await graph.ainvoke({
            "messages": [],
            "items": items_data,
            "raw_checks": "",
            "verdicts": "",
        })
    except Exception:
        logger.exception("Availability check graph execution failed")
        return AvailabilityCheckResponse(
            results=[
                AvailabilityVerdict(
                    url=item.url,
                    coin_name=item.coin_name,
                    status="unknown",
                    reason="Agent check failed",
                    confidence="low",
                )
                for item in request.items
            ]
        )

    verdicts_raw = result.get("verdicts", "")
    try:
        verdicts = parse_verdicts(verdicts_raw, expected_items=items_data)
    except AvailabilityVerdictParseError as exc:
        logger.error(
            "Availability verdict parse failure: %s. Raw snippet=%r",
            exc,
            verdicts_raw[:300],
        )
        verdicts = [
            AvailabilityVerdict(
                url=item.url,
                coin_name=item.coin_name,
                status="unknown",
                reason="Agent response failed schema validation",
                confidence="low",
            )
            for item in request.items
        ]

    return AvailabilityCheckResponse(results=verdicts)
