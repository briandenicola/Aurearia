"""Request models received from the Go API proxy.

The Go API enriches each request with settings, user context, and data
so this service remains stateless with no direct DB access.
"""

from typing import Annotated, Any, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, ValidationError, field_validator, model_validator

from app.outbound import validate_outbound_url
from app.teams.specialist_contracts import SpecialistResult

MAX_MESSAGE_LENGTH = 4000
MAX_HISTORY_MESSAGE_LENGTH = 20000
MAX_HISTORY_MESSAGES = 50
MAX_HISTORY_TOTAL_CHARS = 100000
MAX_PROMPT_LENGTH = 12000
MAX_IMAGE_COUNT = 20
MAX_IMAGE_BASE64_LENGTH = 10 * 1024 * 1024
MAX_URL_LENGTH = 2048
MAX_NAME_LENGTH = 300
MAX_NOTES_LENGTH = 10000
MAX_PORTFOLIO_MAP_ITEMS = 200
MAX_PORTFOLIO_LIST_ITEMS = 200
MAX_TOP_COINS = 100
MAX_AVAILABILITY_ITEMS = 10
MAX_ALERT_CANDIDATES = 50
MAX_WISHLIST_FEATURED_SUMMARY_LENGTH = 500
MAX_SET_BUILDER_PROMPT_LENGTH = 500
MAX_SET_BUILDER_FEEDBACK_LENGTH = 1000
MAX_SET_BUILDER_MAX_TURNS = 8
MAX_SET_BUILDER_MAX_SLOTS = 300
MAX_COPILOT_GOAL_LENGTH = 4000
MAX_COPILOT_PLAN_ITEMS = 12
MAX_COPILOT_PLAN_TITLE_LENGTH = 200
MAX_COPILOT_CLARIFICATION_LENGTH = 500
MAX_COPILOT_CLARIFICATION_CHOICES = 10
MAX_COPILOT_ALLOWED_TOOLS = 10

COPILOT_ALLOWED_TOOLS = frozenset(
    {
        "search_my_collection",
        "get_coin",
        "collection_summary",
        "top_coins_by_value",
        "portfolio_review",
        "gap_analysis",
        "market_search",
        "auction_search",
        "price_trends",
        "similar_lots",
    }
)
COPILOT_SPECIALIST_TOOLS = frozenset(
    {
        "market_search",
        "auction_search",
        "price_trends",
        "similar_lots",
    }
)

# 344-deep-agentic-coin-identification (contracts/agent-internal-contract.md §2)
MAX_DEEP_IMAGE_DATA_URI_LENGTH = 10 * 1024 * 1024
MAX_DEEP_IMAGES = 4  # obverse + reverse + up to MaxDeepIdentificationHintArtifacts (3) hints, capped generously
MAX_DEEP_NOTES_LENGTH = 10000
MAX_DEEP_PROVIDER_CATALOG_ENTRIES = 10
MAX_DEEP_PROVIDER_OVERRIDE_ENTRIES = 10
DEEP_PROVIDER_NAMES = {"numista", "nomisma", "ngc", "ocre", "rpc"}

BoundedMessage = Annotated[str, StringConstraints(max_length=MAX_MESSAGE_LENGTH)]
BoundedHistoryMessage = Annotated[str, StringConstraints(max_length=MAX_HISTORY_MESSAGE_LENGTH)]
BoundedPrompt = Annotated[str, StringConstraints(max_length=MAX_PROMPT_LENGTH)]
BoundedName = Annotated[str, StringConstraints(max_length=MAX_NAME_LENGTH)]
BoundedNotes = Annotated[str, StringConstraints(max_length=MAX_NOTES_LENGTH)]
BoundedOptionalURL = Annotated[str, StringConstraints(max_length=MAX_URL_LENGTH)]
BoundedURL = Annotated[str, StringConstraints(min_length=1, max_length=MAX_URL_LENGTH)]
BoundedImageBase64 = Annotated[str, StringConstraints(max_length=MAX_IMAGE_BASE64_LENGTH)]
BoundedSetBuilderPrompt = Annotated[str, StringConstraints(min_length=1, max_length=MAX_SET_BUILDER_PROMPT_LENGTH)]
BoundedSetBuilderFeedback = Annotated[str, StringConstraints(max_length=MAX_SET_BUILDER_FEEDBACK_LENGTH)]
BoundedWishlistFeaturedSummary = Annotated[str, StringConstraints(max_length=MAX_WISHLIST_FEATURED_SUMMARY_LENGTH)]


class StrictRequestModel(BaseModel):
    """Base model for Go-to-agent DTOs with drift detection."""

    model_config = ConfigDict(extra="forbid")


CopilotThreadID = Annotated[str, StringConstraints(pattern=r"^cct_[A-Za-z0-9_-]+$")]
CopilotRunID = Annotated[str, StringConstraints(pattern=r"^ccr_[A-Za-z0-9_-]+$")]
CopilotExecutionID = Annotated[str, StringConstraints(pattern=r"^cce_[A-Za-z0-9_-]+$")]
CopilotToolCallID = Annotated[str, StringConstraints(min_length=1, max_length=200)]


def _validate_history_total_chars(history: list["ChatMessage"]) -> list["ChatMessage"]:
    total_chars = sum(len(msg.content) for msg in history)
    if total_chars > MAX_HISTORY_TOTAL_CHARS:
        raise ValueError(
            f"history content exceeds {MAX_HISTORY_TOTAL_CHARS} total characters",
        )
    return history


class LLMConfig(StrictRequestModel):
    """LLM configuration passed per-request from Go."""

    provider: str  # "anthropic" or "ollama"
    api_key: str = ""  # Anthropic API key (empty for Ollama)
    model: str = ""  # Model name
    ollama_url: str = ""  # Ollama base URL (empty for Anthropic)
    searxng_url: str = ""  # SearXNG URL (for Ollama web search)

    @model_validator(mode="after")
    def validate_provider_urls(self) -> "LLMConfig":
        if self.provider != "ollama":
            self.ollama_url = ""
            self.searxng_url = ""
            return self

        self.ollama_url = validate_outbound_url(self.ollama_url, "ollama_url")
        self.searxng_url = validate_outbound_url(self.searxng_url, "searxng_url")
        return self


class UserContext(StrictRequestModel):
    """User context for personalizing agent behavior."""

    user_id: int
    zip_code: Annotated[str, StringConstraints(max_length=32)] = ""


class ChatMessage(StrictRequestModel):
    """A single message in conversation history."""

    role: Literal["user", "assistant"]
    content: BoundedHistoryMessage


class CopilotPlanItem(StrictRequestModel):
    """Public, resumable plan state. It never contains model reasoning."""

    id: Annotated[str, StringConstraints(min_length=1, max_length=64)]
    title: Annotated[str, StringConstraints(min_length=1, max_length=MAX_COPILOT_PLAN_TITLE_LENGTH)]
    status: Literal["pending", "in_progress", "completed", "skipped", "failed"]


class CopilotUsage(StrictRequestModel):
    """Cumulative execution counters supplied by and returned to Go."""

    iterations: int = Field(default=0, ge=0)
    tool_calls: int = Field(default=0, ge=0)
    input_tokens: int = Field(default=0, ge=0)
    output_tokens: int = Field(default=0, ge=0)


class CopilotBoundedToolResult(StrictRequestModel):
    """Exact Feature 359 fallback emitted when a persisted result exceeds its byte limit."""

    truncated: Literal[True]
    original_bytes: int = Field(ge=0)
    digest: Annotated[str, StringConstraints(pattern=r"^[0-9a-f]{64}$")]
    summary: Literal["Tool result exceeded the persisted-result limit."]


class CopilotCompletedTool(StrictRequestModel):
    """A bounded, sanitized tool fact from the latest Go checkpoint."""

    tool_call_id: CopilotToolCallID
    tool_name: str
    result_digest: Annotated[str, StringConstraints(max_length=64)] = ""
    result: SpecialistResult | dict[str, Any]
    original_bytes: int = Field(default=0, ge=0)
    persisted_bytes: int = Field(default=0, ge=0)
    truncated: bool = False

    @model_validator(mode="after")
    def validate_tool_result(self) -> "CopilotCompletedTool":
        if self.tool_name not in COPILOT_ALLOWED_TOOLS:
            raise ValueError("tool_name is not in the Coin Copilot allowlist")
        if self.tool_name in COPILOT_SPECIALIST_TOOLS:
            if self.truncated:
                try:
                    fallback = CopilotBoundedToolResult.model_validate(self.result)
                except ValidationError:
                    raise ValueError("truncated specialist results require the bounded fallback envelope")
                self.result = fallback.model_dump(mode="json")
                return self
            try:
                result = SpecialistResult.model_validate(self.result)
            except ValueError as exc:
                raise ValueError("specialist result is invalid") from exc
            if result.capability != self.tool_name:
                raise ValueError("specialist result capability does not match tool_name")
            self.result = result
        elif isinstance(self.result, SpecialistResult):
            raise ValueError("specialist result requires a specialist tool_name")
        return self


class CopilotClarification(StrictRequestModel):
    """Typed clarification state that can be resumed by Go."""

    question: Annotated[str, StringConstraints(min_length=1, max_length=MAX_COPILOT_CLARIFICATION_LENGTH)]
    input_type: Literal["text", "single_choice", "boolean"]
    choices: list[Annotated[str, StringConstraints(min_length=1, max_length=200)]] = Field(
        default_factory=list,
        max_length=MAX_COPILOT_CLARIFICATION_CHOICES,
    )

    @model_validator(mode="after")
    def validate_choices(self) -> "CopilotClarification":
        if self.input_type == "single_choice" and not self.choices:
            raise ValueError("single_choice clarification requires choices")
        if self.input_type != "single_choice" and self.choices:
            raise ValueError("choices are only valid for single_choice clarification")
        return self


class CopilotCheckpoint(StrictRequestModel):
    """Latest durable continuation state supplied by Go."""

    version: int = Field(ge=0)
    plan: list[CopilotPlanItem] = Field(default_factory=list, max_length=MAX_COPILOT_PLAN_ITEMS)
    completed_tools: list[CopilotCompletedTool] = Field(default_factory=list, max_length=40)
    pending_clarification: CopilotClarification | None = None
    next_action: Literal["continue", "await_clarification", "finish"] = "continue"
    counters: CopilotUsage = Field(default_factory=CopilotUsage)

    @field_validator("completed_tools")
    @classmethod
    def validate_unique_tool_calls(
        cls,
        tools: list[CopilotCompletedTool],
    ) -> list[CopilotCompletedTool]:
        ids = [tool.tool_call_id for tool in tools]
        if len(ids) != len(set(ids)):
            raise ValueError("completed_tools contains duplicate tool_call_id values")
        return tools


class CopilotLimits(StrictRequestModel):
    """Snapshotted limits for one stateless execution."""

    max_iterations: int = Field(ge=1, le=20)
    max_tool_calls: int = Field(ge=1, le=40)
    max_concurrent_tools: Literal[1]
    hard_timeout_seconds: int = Field(ge=15, le=600)
    max_persisted_tool_result_bytes: int = Field(ge=4096, le=131072)


class CopilotAppContext(StrictRequestModel):
    """Bounded, non-authoritative UI context."""

    route: Annotated[str, StringConstraints(max_length=MAX_URL_LENGTH)] = ""
    active_coin_id: int | None = Field(default=None, alias="activeCoinId", ge=1)


class CopilotCapabilityRequest(StrictRequestModel):
    """Provider configuration used only to verify fixed Copilot tool binding."""

    llm: LLMConfig


class CopilotExecuteRequest(StrictRequestModel):
    """Complete stateless Go-to-Python Coin Copilot execution request."""

    schema_version: Literal[1]
    thread_id: CopilotThreadID
    run_id: CopilotRunID
    execution_id: CopilotExecutionID
    goal: Annotated[str, StringConstraints(min_length=1, max_length=MAX_COPILOT_GOAL_LENGTH)]
    messages: list[ChatMessage] = Field(min_length=1, max_length=MAX_HISTORY_MESSAGES)
    checkpoint: CopilotCheckpoint
    app_context: CopilotAppContext | None = None
    llm: LLMConfig
    limits: CopilotLimits
    tools_base_url: BoundedURL
    execution_token: Annotated[str, StringConstraints(min_length=1, max_length=8192)]
    allowed_tools: list[str] = Field(min_length=1, max_length=MAX_COPILOT_ALLOWED_TOOLS)

    @field_validator("messages")
    @classmethod
    def validate_messages_total_chars(cls, messages: list[ChatMessage]) -> list[ChatMessage]:
        return _validate_history_total_chars(messages)

    @field_validator("allowed_tools")
    @classmethod
    def validate_allowed_tools(cls, tools: list[str]) -> list[str]:
        if len(tools) != len(set(tools)):
            raise ValueError("allowed_tools contains duplicates")
        if not set(tools).issubset(COPILOT_ALLOWED_TOOLS):
            raise ValueError("allowed_tools contains an unsupported capability")
        return tools

    @model_validator(mode="after")
    def validate_checkpoint_budgets(self) -> "CopilotExecuteRequest":
        counters = self.checkpoint.counters
        if counters.iterations > self.limits.max_iterations:
            raise ValueError("checkpoint exceeds max_iterations")
        if counters.tool_calls > self.limits.max_tool_calls:
            raise ValueError("checkpoint exceeds max_tool_calls")
        return self


class AppContext(StrictRequestModel):
    """Frontend route context proxied by Go for collection-aware chat."""

    route: Annotated[str, StringConstraints(max_length=MAX_URL_LENGTH)] = ""
    active_coin_id: int | None = Field(default=None, alias="activeCoinId", ge=1)


class PortfolioCoin(StrictRequestModel):
    """Summarized coin for portfolio review."""

    name: BoundedName
    category: BoundedName = ""
    material: BoundedName = ""
    era: BoundedName = ""
    ruler: BoundedName = ""
    grade: Annotated[str, StringConstraints(max_length=64)] = ""
    purchase_price: float = 0
    current_value: float = 0


class PortfolioSummary(StrictRequestModel):
    """Portfolio summary data passed from Go."""

    total_coins: int = 0
    total_value: float = 0
    total_invested: float = 0
    categories: dict[str, int] = Field(default_factory=dict, max_length=MAX_PORTFOLIO_MAP_ITEMS)
    materials: dict[str, int] = Field(default_factory=dict, max_length=MAX_PORTFOLIO_MAP_ITEMS)
    eras: list[dict[str, Any]] = Field(default_factory=list, max_length=MAX_PORTFOLIO_LIST_ITEMS)
    rulers: list[dict[str, Any]] = Field(default_factory=list, max_length=MAX_PORTFOLIO_LIST_ITEMS)
    top_coins: list[PortfolioCoin] = Field(default_factory=list, max_length=MAX_TOP_COINS)
    missing_fields: dict[str, int] = Field(default_factory=dict, max_length=MAX_PORTFOLIO_MAP_ITEMS)

    @field_validator("categories", "materials", "missing_fields", mode="before")
    @classmethod
    def none_to_dict(cls, v: dict | None) -> dict:
        """Go serializes nil maps as null — convert to empty dict."""
        return v if v is not None else {}

    @field_validator("eras", "rulers", "top_coins", mode="before")
    @classmethod
    def none_to_list(cls, v: list | None) -> list:
        """Go serializes nil slices as null — convert to empty list."""
        return v if v is not None else []


class CoinSearchRequest(StrictRequestModel):
    """Request to search for coins."""

    llm: LLMConfig
    user: UserContext
    message: BoundedMessage
    history: list[ChatMessage] = Field(default_factory=list, max_length=MAX_HISTORY_MESSAGES)
    app_context: AppContext | None = None
    coin_search_prompt: BoundedPrompt = ""
    coin_shows_prompt: BoundedPrompt = ""
    portfolio: PortfolioSummary | None = None
    internal_token: str = ""
    tools_base_url: BoundedOptionalURL = ""

    @field_validator("history")
    @classmethod
    def validate_history_total_chars(cls, history: list[ChatMessage]) -> list[ChatMessage]:
        return _validate_history_total_chars(history)


class CoinShowSearchRequest(StrictRequestModel):
    """Request to search for coin shows."""

    llm: LLMConfig
    user: UserContext
    message: BoundedMessage
    history: list[ChatMessage] = Field(default_factory=list, max_length=MAX_HISTORY_MESSAGES)
    coin_search_prompt: BoundedPrompt = ""
    coin_shows_prompt: BoundedPrompt = ""

    @field_validator("history")
    @classmethod
    def validate_history_total_chars(cls, history: list[ChatMessage]) -> list[ChatMessage]:
        return _validate_history_total_chars(history)


class CoinData(StrictRequestModel):
    """Coin data passed from Go for analysis or valuation."""

    id: int
    name: BoundedName = ""
    ruler: BoundedName = ""
    era: BoundedName = ""
    denomination: BoundedName = ""
    material: BoundedName = ""
    category: BoundedName = ""
    grade: Annotated[str, StringConstraints(max_length=64)] = ""
    purchase_price: float = 0
    current_value: float = 0
    notes: BoundedNotes = ""


class AnalyzeRequest(StrictRequestModel):
    """Request to analyze coin images."""

    llm: LLMConfig
    coin: CoinData
    images: list[BoundedImageBase64] = Field(default_factory=list, max_length=MAX_IMAGE_COUNT)
    side: Annotated[str, StringConstraints(max_length=16)] = ""  # "obverse", "reverse", or "" for both
    prompt: BoundedPrompt = ""  # Analysis prompt from admin settings
    format_output: bool = True  # False returns raw model output for structured lookup flows


class GradeRequest(StrictRequestModel):
    """Request to estimate a coin grade from owner-scoped coin images."""

    llm: LLMConfig
    coin: CoinData
    images: list[BoundedImageBase64] = Field(default_factory=list, max_length=MAX_IMAGE_COUNT)

    @field_validator("images")
    @classmethod
    def validate_images_present(cls, images: list[str]) -> list[str]:
        if not images:
            raise ValueError("at least one coin image is required for grading")
        return images


class BidMarketSignalRequest(StrictRequestModel):
    """Request for a structured market-trend signal for a described auction lot."""

    llm: LLMConfig
    coin: CoinData


class WishlistFeaturedSummaryCoin(StrictRequestModel):
    """Wishlist coin context passed from Go for featured-summary drafting."""

    name: BoundedName
    era: BoundedName = ""
    category: BoundedName = ""
    denomination: BoundedName = ""
    ruler: BoundedName = ""
    mint: BoundedName = ""
    obverse_analysis: BoundedNotes = ""
    reverse_analysis: BoundedNotes = ""
    ai_analysis: BoundedNotes = ""


class WishlistFeaturedSummaryRequest(StrictRequestModel):
    """Request to produce a concise, factual wishlist featured-coin rationale."""

    llm: LLMConfig
    coin: WishlistFeaturedSummaryCoin
    user_display_name: BoundedName = ""


class IntakeDraftRequest(StrictRequestModel):
    """Request to generate an intake draft from observation images."""

    llm: LLMConfig
    images: list[BoundedImageBase64] = Field(default_factory=list, max_length=MAX_IMAGE_COUNT)
    coin_card_image: BoundedImageBase64 = ""

    @field_validator("images")
    @classmethod
    def validate_images_present(cls, images: list[str]) -> list[str]:
        if not images:
            raise ValueError("at least one observation image is required")
        return images


class PortfolioReviewRequest(StrictRequestModel):
    """Request to review a portfolio."""

    llm: LLMConfig
    user: UserContext
    portfolio: PortfolioSummary
    message: BoundedMessage = ""
    history: list[ChatMessage] = Field(default_factory=list, max_length=MAX_HISTORY_MESSAGES)
    valuation_prompt: BoundedPrompt = ""

    @field_validator("history", mode="before")
    @classmethod
    def none_to_list(cls, v: list | None) -> list:
        """Go serializes nil slices as null — convert to empty list."""
        return v if v is not None else []

    @field_validator("history")
    @classmethod
    def validate_history_total_chars(cls, history: list[ChatMessage]) -> list[ChatMessage]:
        return _validate_history_total_chars(history)


class AvailabilityCheckItem(StrictRequestModel):
    """A single coin URL to check for availability."""

    url: BoundedURL
    coin_name: BoundedName = ""


class AvailabilityCheckRequest(StrictRequestModel):
    """Request to check listing availability for multiple URLs."""

    llm: LLMConfig
    items: list[AvailabilityCheckItem] = Field(default_factory=list, max_length=MAX_AVAILABILITY_ITEMS)

    @field_validator("items")
    @classmethod
    def validate_unique_urls(cls, items: list[AvailabilityCheckItem]) -> list[AvailabilityCheckItem]:
        urls = [item.url for item in items]
        if len(set(urls)) != len(urls):
            raise ValueError("items contain duplicate URLs")
        return items


# Wishlist search alert discovery DTOs.
# Contract anchor: specs/337-wishlist-search-alerts/contracts/agent-discovery-contract.md
class AlertDiscoveryCriteriaSnapshot(StrictRequestModel):
    """Immutable criteria snapshot supplied by the Go API."""

    name: BoundedName
    ruler_or_issuer: BoundedName = ""
    coin_type: BoundedName = ""
    date_from: int | None = None
    date_to: int | None = None
    mint: BoundedName = ""
    material: BoundedName = ""
    grade_or_condition: BoundedName = ""
    price_min: float | None = Field(default=None, ge=0)
    price_max: float | None = Field(default=None, ge=0)
    currency: Annotated[str, StringConstraints(max_length=3)] = "USD"
    dealer_preference: BoundedName = ""
    source_filters: list[Annotated[str, StringConstraints(max_length=253)]] = Field(default_factory=list, max_length=20)
    keywords: Annotated[str, StringConstraints(max_length=500)] = ""
    notes: BoundedNotes = ""

    @model_validator(mode="after")
    def validate_ranges(self) -> "AlertDiscoveryCriteriaSnapshot":
        if self.price_min is not None and self.price_max is not None and self.price_min > self.price_max:
            raise ValueError("price_min must be less than or equal to price_max")
        if self.date_from is not None and self.date_to is not None and self.date_from > self.date_to:
            raise ValueError("date_from must be less than or equal to date_to")
        self.currency = self.currency.upper()
        return self


class AlertDiscoveryDetail(StrictRequestModel):
    """Alert discovery request details supplied by Go."""

    alert_id: int = Field(ge=1)
    criteria_snapshot: AlertDiscoveryCriteriaSnapshot
    max_candidates: int = Field(default=20, ge=1, le=MAX_ALERT_CANDIDATES)


class AlertDiscoveryRequest(StrictRequestModel):
    """Stateless alert discovery request. Python never persists or scopes users."""

    llm: LLMConfig
    alert: AlertDiscoveryDetail


# Dynamic Set Builder workflow DTOs.
# Contract anchor: specs/011-dynamic-set-builder-correction-plan.md (Phase 2)
class SetBuilderRequest(StrictRequestModel):
    """Stateless set-builder workflow request.

    Python never creates or modifies sets — it only proposes structured
    roster data. Go owns persistence, approval, and set creation (Phase 3+).
    """

    llm: LLMConfig
    user: UserContext
    run_id: int | None = Field(default=None, ge=1)
    prompt: BoundedSetBuilderPrompt
    # Optional summary of the user's existing collection, passed from Go so
    # the Collection Matcher role can estimate filled/likely-matched slots
    # without Python ever touching the database directly.
    collection: PortfolioSummary | None = None
    max_turns: int = Field(default=4, ge=1, le=MAX_SET_BUILDER_MAX_TURNS)
    max_slots: int = Field(default=200, ge=1, le=MAX_SET_BUILDER_MAX_SLOTS)
    enable_external_lookup: bool = True
    # Optional feedback text for a regenerate-with-feedback request (US2 Phase 4).
    feedback: BoundedSetBuilderFeedback = ""


# Deep Agentic Coin Identification DTOs (344-deep-agentic-coin-identification).
# Contract anchor: specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md §2
BoundedDeepDataURI = Annotated[str, StringConstraints(max_length=MAX_DEEP_IMAGE_DATA_URI_LENGTH)]
BoundedDeepNotes = Annotated[str, StringConstraints(max_length=MAX_DEEP_NOTES_LENGTH)]


class DeepIdentifyImage(StrictRequestModel):
    """A single obverse/reverse/hint image passed as a data URI.

    Hint images are marked by role="hint" and are used only as router/LLM
    context — they never enter the coin-face vision-prompt slots (FR-004).
    """

    role: Literal["obverse", "reverse", "hint"]
    data_uri: BoundedDeepDataURI
    # Wire contract §2 forward-compatibility placeholder (contracts/agent-
    # internal-contract.md §2, e.g. "label"/"slab"): Go populates this today
    # but no Python provider/router/hypothesis logic reads it yet — kept per
    # T102 rather than dropped, since removing it would silently break the
    # Go↔Python wire contract for a field Go already sends on every hint
    # image.
    hint_kind: Annotated[str, StringConstraints(max_length=40)] = ""


class DeepProviderCatalogEntry(StrictRequestModel):
    """One entry of the per-run provider catalog Go supplies (§2/§7).

    `automatable: false` entries are never contacted upstream by Python;
    the router short-circuits them into a typed `not_automated`/
    `unavailable` evidence row (FR-025).
    """

    provider: Literal["numista", "nomisma", "ngc", "ocre", "rpc"]
    automatable: bool
    # Wire contract §2 forward-compatibility placeholder (contracts/agent-
    # internal-contract.md §2): Go already sends a per-provider call budget
    # on every catalog entry, but no Python provider node currently reads or
    # enforces it — kept documented per T102 rather than removed, since
    # deleting it here would silently drop a field Go still populates on
    # every request.
    call_budget: int = Field(default=0, ge=0)
    reason: Annotated[str, StringConstraints(max_length=100)] = ""
    link_out: BoundedOptionalURL = ""


class DeepIdentifyBounds(StrictRequestModel):
    """Per-run bounds Go supplies; the graph must never exceed these
    (contracts/agent-internal-contract.md §2/§6).
    """

    max_providers: int = Field(ge=1, le=10)
    max_concurrency: int = Field(ge=1, le=10)
    provider_timeout_s: int = Field(ge=1, le=120)
    total_timeout_s: int = Field(ge=1, le=900)
    recursion_limit: int = Field(ge=1, le=50)


class QuickEvidenceNGC(StrictRequestModel):
    """Optional NGC cert data already extracted upstream (F341 Quick
    Identify). Never re-extracted or re-OCR'd by this pipeline.
    """

    cert_number: Annotated[str, StringConstraints(max_length=40)] = ""
    grade: Annotated[str, StringConstraints(max_length=32)] = ""
    lookup_url: BoundedOptionalURL = ""


class QuickEvidence(StrictRequestModel):
    """Optional normalized output of CoinLookupService.Lookup, passed
    through as additional router/synthesis context. Never required —
    absent when Go has no prior quick-lookup result for this job.
    """

    label_text: Annotated[str, StringConstraints(max_length=2000)] = ""
    coin_fields: dict[str, str] = Field(default_factory=dict, max_length=50)
    confidence: Annotated[str, StringConstraints(max_length=16)] = ""
    ngc: QuickEvidenceNGC | None = None
    numista_query: Annotated[str, StringConstraints(max_length=300)] = ""


class DeepIdentifyRequest(StrictRequestModel):
    """Go → Python deep-identification pipeline request (§2). Python holds
    no database handle, no API keys beyond `llm`, and no persistent state
    (Principle II, FR-035) — every field needed to run the pipeline for
    this one job is supplied here.
    """

    job_id: int = Field(ge=1)
    # Wire contract §2 forward-compatibility placeholder (contracts/agent-
    # internal-contract.md §2): accepted from Go on every request but not
    # yet branched on by any Python logic — no request currently sends a
    # value other than the default. Kept documented per T102 rather than
    # removed so future contract revisions can rely on Python already
    # accepting/round-tripping this field.
    schema_version: int = 1
    llm: LLMConfig
    images: list[DeepIdentifyImage] = Field(default_factory=list, max_length=MAX_DEEP_IMAGES)
    notes: BoundedDeepNotes = ""
    quick_evidence: QuickEvidence | None = None
    provider_override: list[Literal["numista", "nomisma", "ngc", "ocre", "rpc"]] = Field(
        default_factory=list, max_length=MAX_DEEP_PROVIDER_OVERRIDE_ENTRIES
    )
    provider_catalog: list[DeepProviderCatalogEntry] = Field(
        default_factory=list, max_length=MAX_DEEP_PROVIDER_CATALOG_ENTRIES
    )
    bounds: DeepIdentifyBounds
    tools_base_url: BoundedOptionalURL = ""
    internal_token: str = ""

    @field_validator("images")
    @classmethod
    def validate_required_faces(cls, images: list[DeepIdentifyImage]) -> list[DeepIdentifyImage]:
        roles = [img.role for img in images]
        if roles.count("obverse") != 1 or roles.count("reverse") != 1:
            raise ValueError("exactly one obverse and one reverse image are required")
        return images

    @field_validator("provider_catalog")
    @classmethod
    def validate_unique_providers(cls, catalog: list[DeepProviderCatalogEntry]) -> list[DeepProviderCatalogEntry]:
        names = [entry.provider for entry in catalog]
        if len(set(names)) != len(names):
            raise ValueError("provider_catalog contains duplicate providers")
        return catalog
