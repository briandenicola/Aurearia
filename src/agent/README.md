# Ancient Coins — Agent Service

Python 3.12 + FastAPI + LangGraph + LangChain multi-agent service for AI-powered numismatic features. Stateless service with no database access — all configuration and context is passed per-request from the Go API.

## Prerequisites

- **Python** 3.12+
- **uv** 0.11.22 for locked installs

## Install

```bash
pip install uv==0.11.22
uv sync --locked --extra dev    # install locked dependencies with dev tools
```

## Run

```bash
uvicorn app.main:app --port 8081
```

The Go API proxies AI requests to this service at `AGENT_SERVICE_URL` (default `http://localhost:8081`). SSE streams flow Python → Go → Vue.

## Test & Lint

```bash
uv run pytest tests/ -v             # run all tests
uv run pytest tests/test_foo.py -v  # run a specific test file
uv run ruff check app/ tests/       # lint (E, F, I, W rules)
```

Ruff is configured for Python 3.12 with a 120-character line length. Pytest uses `asyncio_mode = "auto"`.

## Dependency Locking

Agent dependencies are locked in `uv.lock`. Refresh the lock after intentional dependency range changes or Dependabot uv PRs:

```bash
pip install uv==0.11.22
uv lock --upgrade
uv sync --locked --extra dev
```

## Architecture

```
Vue SPA → Go API (8080) → Python Agent Service (8081)
```

The service is fully **stateless**: no database, no persistent storage. The Go API sends all necessary context (user message, LLM config, portfolio data, prompts, user context) in each request. Responses stream back as SSE events.

### Supervisor

The top-level supervisor (`app/supervisor.py`) uses a LangGraph `StateGraph` with a lightweight router that classifies user intent and delegates to the appropriate team pipeline:

| Team | Pipeline | Description |
|---|---|---|
| **Coin Search** | Search → Fetch dealer pages → Format | Find coins for sale from dealers |
| **Coin Shows** | Search → Verify dates → Format | Find upcoming coin shows/events |
| **Coin Analysis** | Vision model analysis → Format | Analyze coin images for ID and authenticity |
| **Coin Grading** | Photo analysis → Grade estimation | AI grade estimation from coin photos |
| **Portfolio Review** | Read holdings → Valuate → Analyze | Portfolio analysis and valuation |
| **Gap Analysis** | Summarize collection → Analyze gaps → Suggest | Collection completeness analysis |
| **Price Trends** | Search auctions → Analyze trends | Auction price history and market direction |
| **Similar Lots** | Search → Score similarity → Format | Find similar coins at active auctions |
| **Auction Search** | Search NumisBids → Fetch lots → Format | Search auction lots |
| **General** | Direct LLM response | General numismatic Q&A |

### Key Design Rules

- Search agents pass only tool-returned data downstream — never invented details
- Verification agents confirm every URL is live and every date is in the future
- All worker agent outputs conform to defined Pydantic schemas — no free-form text
- The router classifies intent using recent conversation history for context

## Directory Structure

```
app/
  main.py            # FastAPI entry point and middleware setup
  config.py          # Pydantic Settings configuration
  routes.py          # API route definitions
  supervisor.py      # Top-level LangGraph supervisor and router
  streaming.py       # SSE streaming utilities
  logging_config.py  # Structured logging with ring buffer
  llm/
    provider.py      # get_chat_model() and get_search_model() factory
    retry.py         # LLM invocation with retry logic
  models/
    requests.py      # Pydantic request schemas (LLMConfig, UserContext, etc.)
    responses.py     # Pydantic response schemas
  teams/
    coin_search.py   # Team 1: Coin search pipeline
    coin_shows.py    # Team 2: Coin show search pipeline
    coin_analysis.py # Team 3: Coin image analysis
    coin_grading.py  # Team 6: AI grade estimation
    portfolio_review.py  # Team 4: Portfolio review pipeline
    availability_check.py # Team 5: URL availability checking
    gap_analysis.py  # Team 7: Collection gap analysis
    price_trends.py  # Team 9: Auction price trends
    similar_lots.py  # Team 10: Similar lot finder
    auction_search.py # Team 5: Auction lot search
  tools/
    search.py        # Web search tools (Anthropic web_search, SearXNG)
    numisbids.py     # NumisBids auction search tool
```

## AI Provider Configuration

The provider is selected per-request via `LLMConfig` passed from the Go API (configured in Admin Settings):

| Provider | Chat Model | Web Search |
|---|---|---|
| **Anthropic** | Claude via `ChatAnthropic` | Claude's built-in `web_search_20250305` tool (via `get_search_model()`) |
| **Ollama** | Self-hosted models via `ChatOllama` | `create_react_agent` with SearXNG tool |

Use `get_search_model()` from `app/llm/provider.py` for any agent node that needs web search. Use `get_chat_model()` for nodes that don't search. Anthropic's web search tool is **not** available by default on `ChatAnthropic` — it must be bound via `get_search_model()`.

## Key Dependencies

| Package | Purpose |
|---|---|
| `fastapi` | HTTP framework |
| `uvicorn` | ASGI server |
| `langgraph` | StateGraph-based team pipelines |
| `langchain` | Base LLM abstractions |
| `langchain-anthropic` | Anthropic/Claude provider |
| `langchain-ollama` | Ollama self-hosted provider |
| `langchain-community` | Community tools and integrations |
| `httpx` | Async HTTP client |
| `pydantic` / `pydantic-settings` | Request/response schemas and config |
| `sse-starlette` | Server-Sent Events support |
| `tenacity` | Retry logic for LLM calls |
