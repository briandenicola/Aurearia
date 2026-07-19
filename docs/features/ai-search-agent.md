# AI Coin Search Agent

> Chat with an intelligent agent to discover coins matching your description across dealer sites, with real-time search results and structured suggestions.

## Overview

The AI Coin Search Agent is a multi-team LangGraph orchestrator that searches dealer websites, auction sites, and catalogs for coins matching your description. It returns structured results with images, metadata, prices, source links, and candidate catalog references that can be imported directly to your wishlist.

## Architecture

```
Vue Chat UI ──► Go API ──► Python Agent Service
  (streaming)   (proxy)    (multi-team pipeline)
```

The agent runs in a separate Python service (FastAPI + LangGraph) and streams responses back via Server-Sent Events (SSE) for real-time visibility into the search process.

## Supported Providers

| Provider | Search Method | Setup |
|----------|---------------|-------|
| **Anthropic Claude** | Built-in `web_search_20250305` tool | API key required |
| **Ollama** | ReAct agent + SearXNG | Self-hosted Ollama + SearXNG |

**Recommendation**: Use Anthropic Claude for better search quality and built-in web search. Ollama requires additional setup but offers cost savings for self-hosted deployments.

## Agent Teams

The agent uses specialized teams for different tasks:

### Team 1: Coin Search
- Searches dealer websites for coins matching description
- Uses search terms derived from user description
- Fetches listing pages and extracts coin metadata
- Returns: name, denomination, ruler, era, material, estimated price, source link

### Team 2: Coin Shows
- Searches for upcoming coin shows and auction events
- Verifies event dates are in the future
- Returns: event name, date, location, registration link

### Team 3: Coin Analysis
- Analyzes uploaded coin photos using vision models
- Works alongside the search results
- Provides detailed identification and condition assessment

### Team 4: Portfolio Review
- Reads your collection summary
- Analyzes holdings by era, ruler, material
- Identifies gaps and suggests acquisitions
- Returns: gap analysis, targeted recommendations with estimated prices

### Team 5: Auction Search
- Searches supported auction sources for matching lots
- Formats auction lot candidates with provider links and source context

### Team 6: Availability Check
- Verifies URLs are still live and listings active
- Analyzes HTTP status and page content
- Returns: availability verdict, confidence level

### Team 7: Coin Grading
- Estimates grades from uploaded photos
- Provides reasoning and confidence scores
- Uses Sheldon scale (VF, EF, AU, MS, etc.)

### Team 9: Price Trends
- Analyzes historical auction data for coin types
- Identifies market trends (up, down, stable)
- Returns: price history, trend direction

### Team 7: Gap Analysis
- Comprehensive portfolio gap analysis
- Suggests specific coins to fill collection gaps
- Estimates market prices

### Team 10: Coin Photography Guide
- Reviews uploaded coin photos
- Provides critique and improvement tips
- Covers lighting, focus, background, composition

### Team 11: Similar Lot Finder
- Finds active auction lots similar to your coins
- Ranks by relevance
- Useful for valuation research

## Chat Interface

### Opening the Agent

1. Navigate to **Wish List** page
2. Click **Find Coins** button
3. Chat drawer opens (or full-screen on mobile)

### Typing a Description

Enter natural language descriptions:

**Examples**:
- "Roman silver denarii of Julius Caesar under $500"
- "Greek silver tetradrachms from Athens, 5th century"
- "Byzantine gold coins from the reign of Justinian"
- "Find upcoming coin shows in the next 30 days"

### Real-Time Status

During search, see active status indicators:
- 🔍 Searching dealer websites...
- 📥 Fetching lot details...
- 📊 Formatting results...
- ✅ Complete

### Search Results

Each result displays:

```
[COIN IMAGE]
Name: Roman Silver Denarius of Julius Caesar
Denomination: Denarius (silver)
Ruler: Julius Caesar
Era: Ancient (49-44 BC)
Material: Silver (95%+)
Estimated Price: $485

🔗 View on dealer site
➕ Add to Wishlist
```

### Candidate Catalog References

Search results include structured references:
- Catalog (RIC, RPC, SNG, Numista, etc.)
- Volume number (if applicable)
- Catalog entry number
- Optional invoice number and authority URI
- Authority URI (link to source)

These are imported with the coin to your wishlist.

## Streaming & Real-Time Updates

### SSE Streaming
- Responses stream token-by-token as they're generated
- Frontend displays progressive text with cursor animation
- Status events show which agent team is active

### Connection Resilience
- Auto-reconnect on network failure
- Timeout after 5 minutes of inactivity
- Clear error messages if agent service is unreachable

## Saving Conversations

### Save Chat
1. During or after a search, click **Save Conversation**
2. Optionally add a title (e.g., "Julius Caesar Coins - May 2025")
3. Conversation is saved to your account

### Access Saved Chats
1. Go to **Settings → Conversations**
2. Browse list of saved chats
3. Click to reopen and continue searching
4. Delete chats you no longer need

## Configuration

### Admin Settings (`Admin → AI Configuration`)

#### Provider Selection
- **AI Provider** — Choose `anthropic` or `ollama`
- Both require valid configuration before any agent features work

#### Anthropic Setup
1. Get API key: [console.anthropic.com](https://console.anthropic.com/)
2. Paste into **Anthropic API Key** field
3. **Anthropic Model** — Auto-populated dropdown of available models
4. (Optional) Customize system prompt in **Coin Search Prompt**

#### Ollama Setup
1. Ensure Ollama is running: `ollama serve`
2. Set **Ollama URL** (default: `http://localhost:11434`)
3. Set **Ollama Model** (e.g., `llama2`, `mistral`)
4. Set **SearXNG URL** for web search (default: `http://localhost:8888`)
5. Set timeout and custom prompts as needed

## Quality & Best Practices

### Tips for Better Results

1. **Be Specific** — "Roman denarii of Julius Caesar" is better than "old Roman coins"
2. **Include Context** — "Silver coins under $500" is more useful than "cheap coins"
3. **Use Known Terms** — Ruler names, denominations, and eras help narrow search
4. **Check Results** — Review sources; verify dealer reputation if purchasing

### Search Limitations

- Search quality depends on dealer website indexing
- Prices may be outdated (listings change daily)
- Some rare coins may not have public listings
- Results are only as good as the search engine can find

## API Reference

```
POST   /api/agent/chat              # Send message to agent (SSE stream)
GET    /api/agent/conversations     # List saved conversations
POST   /api/agent/conversations     # Save current conversation
DELETE /api/agent/conversations/:id # Delete conversation
GET    /api/ai-status               # Check agent provider status
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Agent service unavailable" | Ensure Python agent is running; check `AGENT_SERVICE_URL` |
| Logs show "Internal service credential is not configured" | Set the same `AGENT_INTERNAL_SERVICE_TOKEN` in both the Go API and Python agent containers, then recreate both containers |
| Chat freezes | Increase timeout in admin settings or try again |
| No search results | Try simpler description; check provider configuration |
| Wrong results | Refine description; provide more context (era, material, etc.) |
| API key error | Verify API key in Admin → AI Configuration |

## Related Features

- [Wish List](wish-list.md) — Add search results to wishlist
- [AI Coin Analysis](ai-analysis.md) — Vision-model analysis of uploaded photos
- [Auction Tracking](auction-tracking.md) — Track NumisBids and CNG Auctions lots found by agent
- [Price Trends](price-trends.md) — Analyze auction market trends

## Architecture Deep Dive

For implementation details, see:
- [Multi-Agent Architecture](../ARCHITECTURE.md#multi-agent-architecture-python)

## See Also

- [Getting Started with AI](../getting-started.md#using-ai-features)
