---
applyTo: "src/agent/**"
---

# python guidance

Apply the [constitution](../../.specify/memory/constitution.md) and selected approved work. These scoped instructions implement, not replace, that authority.

### Multi-Agent Architecture (Python)

```
Vue SPA → Go API (8080) → Python Agent Service (8081)
```

The Python agent is a **stateless** FastAPI service — no database access. All configuration (API keys, models, prompts, user context) is passed per-request from the Go API. SSE streams flow Python → Go → Vue (Go proxies the byte stream via `services/agent_proxy.go`).

**Team pipelines:**

| Team | Pipeline |
|---|---|
| Coin Search | Search → Fetch dealer pages → Format |
| Coin Shows | Search → Verify dates are future → Format |
| Coin Analysis | Vision model analysis → Format |
| Portfolio Review | Read holdings → Valuate → Analyze |
| Availability Check | Check URLs → Analyze results → Verdict |

**Key design rules:**
- Search agents pass only tool-returned data downstream — never invented details
- Verification agents confirm every URL is live and every date is in the future
- All worker agent outputs conform to a defined Pydantic schema — no free-form text
- Top-level supervisor (`app/supervisor.py`) enforces max iteration count to prevent loops

### AI Provider Configuration

Users choose one provider in Admin Settings (`AIProvider` key):

- **Anthropic** — Claude models. Web search uses Claude's built-in `web_search_20250305` tool.
- **Ollama** — Self-hosted models. Web search uses a `create_react_agent` with SearXNG tool.

**Important:** Anthropic's `web_search` is NOT available by default on `ChatAnthropic`. Use `get_search_model()` from `app/llm/provider.py` (which calls `bind_tools`) for any agent node that needs web search. Use `get_chat_model()` for nodes that don't search.

### Python (Agent)
- Pydantic models for all request/response schemas (in `app/models/`)
- LangGraph `StateGraph` for team pipelines
- `create_react_agent()` for tool-using agents
- Structured logging via `app/logging_config.py` (ring buffer + stdout)

## Completion

task check:agent against the prepared locked environment; affected cross-service evidence remains additional. See [testing](../../docs/testing.md#6-running-tests-locally-vs-ci).
Setup/installations require separate authorization; missing execution is incomplete.
