"""Request models received from the Go API proxy.

The Go API enriches each request with settings, user context, and data
so this service remains stateless with no direct DB access.
"""

from typing import Annotated, Any, Literal

from pydantic import BaseModel, ConfigDict, Field, StringConstraints, field_validator

from app.outbound import validate_outbound_url

MAX_MESSAGE_LENGTH = 4000
MAX_HISTORY_MESSAGES = 50
MAX_HISTORY_TOTAL_CHARS = 100000
MAX_PROMPT_LENGTH = 12000
MAX_IMAGE_COUNT = 20
MAX_IMAGE_BASE64_LENGTH = 12000000
MAX_URL_LENGTH = 2048
MAX_NAME_LENGTH = 300
MAX_NOTES_LENGTH = 10000
MAX_PORTFOLIO_MAP_ITEMS = 200
MAX_PORTFOLIO_LIST_ITEMS = 200
MAX_TOP_COINS = 100
MAX_AVAILABILITY_ITEMS = 10

BoundedMessage = Annotated[str, StringConstraints(max_length=MAX_MESSAGE_LENGTH)]
BoundedPrompt = Annotated[str, StringConstraints(max_length=MAX_PROMPT_LENGTH)]
BoundedName = Annotated[str, StringConstraints(max_length=MAX_NAME_LENGTH)]
BoundedNotes = Annotated[str, StringConstraints(max_length=MAX_NOTES_LENGTH)]
BoundedURL = Annotated[str, StringConstraints(min_length=1, max_length=MAX_URL_LENGTH)]
BoundedImageBase64 = Annotated[str, StringConstraints(max_length=MAX_IMAGE_BASE64_LENGTH)]


class StrictRequestModel(BaseModel):
    """Base model for Go-to-agent DTOs with drift detection."""

    model_config = ConfigDict(extra="forbid")


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

    @field_validator("ollama_url", "searxng_url")
    @classmethod
    def validate_trusted_urls(cls, value: str, info) -> str:
        return validate_outbound_url(value, info.field_name)


class UserContext(StrictRequestModel):
    """User context for personalizing agent behavior."""

    user_id: int
    zip_code: Annotated[str, StringConstraints(max_length=32)] = ""


class ChatMessage(StrictRequestModel):
    """A single message in conversation history."""

    role: Literal["user", "assistant"]
    content: BoundedMessage


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
    tools_base_url: str = ""

    @field_validator("tools_base_url")
    @classmethod
    def validate_tools_base_url(cls, value: str) -> str:
        return validate_outbound_url(value, "tools_base_url")

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
