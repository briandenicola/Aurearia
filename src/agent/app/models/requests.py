"""Request models received from the Go API proxy.

The Go API enriches each request with settings, user context, and data
so this service remains stateless with no direct DB access.
"""

from pydantic import BaseModel, field_validator


class LLMConfig(BaseModel):
    """LLM configuration passed per-request from Go."""

    provider: str  # "anthropic" or "ollama"
    api_key: str = ""  # Anthropic API key (empty for Ollama)
    model: str = ""  # Model name
    ollama_url: str = ""  # Ollama base URL (empty for Anthropic)
    searxng_url: str = ""  # SearXNG URL (for Ollama web search)


class UserContext(BaseModel):
    """User context for personalizing agent behavior."""

    user_id: int
    zip_code: str = ""


class ChatMessage(BaseModel):
    """A single message in conversation history."""

    role: str  # "user" or "assistant"
    content: str


class PortfolioCoin(BaseModel):
    """Summarized coin for portfolio review."""

    name: str
    category: str = ""
    material: str = ""
    era: str = ""
    ruler: str = ""
    grade: str = ""
    purchase_price: float = 0
    current_value: float = 0


class PortfolioSummary(BaseModel):
    """Portfolio summary data passed from Go."""

    total_coins: int = 0
    total_value: float = 0
    total_invested: float = 0
    categories: dict[str, int] = {}
    materials: dict[str, int] = {}
    eras: list[dict] = []
    rulers: list[dict] = []
    top_coins: list[PortfolioCoin] = []

    @field_validator("categories", "materials", mode="before")
    @classmethod
    def none_to_dict(cls, v: dict | None) -> dict:
        """Go serializes nil maps as null — convert to empty dict."""
        return v if v is not None else {}

    @field_validator("eras", "rulers", "top_coins", mode="before")
    @classmethod
    def none_to_list(cls, v: list | None) -> list:
        """Go serializes nil slices as null — convert to empty list."""
        return v if v is not None else []


class CoinSearchRequest(BaseModel):
    """Request to search for coins."""

    llm: LLMConfig
    user: UserContext
    message: str
    history: list[ChatMessage] = []
    coin_search_prompt: str = ""
    coin_shows_prompt: str = ""
    portfolio: PortfolioSummary | None = None


class CoinShowSearchRequest(BaseModel):
    """Request to search for coin shows."""

    llm: LLMConfig
    user: UserContext
    message: str
    history: list[ChatMessage] = []
    coin_search_prompt: str = ""
    coin_shows_prompt: str = ""


class CoinData(BaseModel):
    """Coin data passed from Go for analysis or valuation."""

    id: int
    name: str = ""
    ruler: str = ""
    era: str = ""
    denomination: str = ""
    material: str = ""
    category: str = ""
    grade: str = ""
    purchase_price: float = 0
    current_value: float = 0
    notes: str = ""


class AnalyzeRequest(BaseModel):
    """Request to analyze coin images."""

    llm: LLMConfig
    coin: CoinData
    images: list[str] = []  # Base64-encoded images
    side: str = ""  # "obverse", "reverse", or "" for both
    prompt: str = ""  # Analysis prompt from admin settings


class PortfolioReviewRequest(BaseModel):
    """Request to review a portfolio."""

    llm: LLMConfig
    user: UserContext
    portfolio: PortfolioSummary
    message: str = ""
    history: list[ChatMessage] = []
    valuation_prompt: str = ""

    @field_validator("history", mode="before")
    @classmethod
    def none_to_list(cls, v: list | None) -> list:
        """Go serializes nil slices as null — convert to empty list."""
        return v if v is not None else []


class AvailabilityCheckItem(BaseModel):
    """A single coin URL to check for availability."""

    url: str
    coin_name: str = ""


class AvailabilityCheckRequest(BaseModel):
    """Request to check listing availability for multiple URLs."""

    llm: LLMConfig
    items: list[AvailabilityCheckItem] = []
