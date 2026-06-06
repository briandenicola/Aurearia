# AI Coin Analysis

> Use vision models to analyze uploaded coin photos and generate comprehensive numismatic assessments of obverse and reverse sides.

## Overview

AI Coin Analysis provides intelligent image-based analysis of your coins using either Anthropic Claude or Ollama vision models. Upload photos of obverse and reverse sides to receive detailed numismatic insights including identification, grade estimates, condition assessment, historical context, and market value insights.

## Supported Providers

| Provider | Model | Web Search | Setup |
|----------|-------|-----------|-------|
| **Anthropic Claude** | Configured Claude model | ✅ Built-in | API key required |
| **Ollama** | Llava, other vision models | ❌ Requires SearXNG | Self-hosted |

## Analysis Workflow

### Starting an Analysis

1. Go to a coin's **detail page** or create a new coin
2. Scroll to the **Images** section
3. Upload photos of the **obverse** and **reverse**
4. Click **Analyze with AI**

### Analysis Process

1. **Provider Selection** — Auto-uses configured default (Admin → AI Configuration)
2. **Separate Streams** — Each side (obverse/reverse) is analyzed independently
3. **Tailored Prompts** — Specialized prompts for each side (configurable)
4. **Streaming Results** — Analysis streams back in real-time
5. **Review & Accept** — Preview results and choose to save or discard

### Analysis Output

Each analysis includes:

- **Identification** — Proposed ruler, denomination, era, category
- **Obverse/Reverse Descriptions** — What appears on each side
- **Inscriptions** — Readable text on the coin
- **Condition Assessment**:
  - Surface wear patterns
  - Strike quality
  - Detail preservation
  - Estimated Sheldon-scale grade
- **Historical Context** — Significance, rarity, known varieties
- **Estimated Market Value** — Price range based on grade and rarity
- **Recommendations** — Suggestions for references or further authentication

## Enabling AI Analysis

### Prerequisites
- Admin access to **Admin → AI Configuration**
- Valid API key for chosen provider (Anthropic or Ollama endpoint)
- Vision model available on chosen provider

### Configuration Steps

1. Go to **Admin → AI Configuration**
2. **Choose Provider**:
   - **Anthropic**: Paste API key from [console.anthropic.com](https://console.anthropic.com/)
   - **Ollama**: Ensure server is running (`ollama serve`) and provide URL (default: `http://localhost:11434`)

3. **Select Model**:
   - **Anthropic**: Dropdown auto-populated from Anthropic API
   - **Ollama**: Manually enter model name (e.g., `llava`, `llama2-vision`)

4. **Set Timeouts**:
   - **Ollama Timeout** — Max seconds to wait for analysis (default: 300)

5. **(Optional) Customize Prompts**:
   - **Obverse Prompt** — Custom instructions for front-side analysis
   - **Reverse Prompt** — Custom instructions for back-side analysis
   - Leave blank to use defaults

### Status Indicator

After configuration:
- **Admin Dashboard** shows provider status: ✅ Connected, ❌ Error, ⏳ Validating
- Error messages indicate configuration issues (invalid key, unreachable endpoint)

## Analysis Quality

### Best Practices for Photos

1. **Clean Coins** — Remove fingerprints and dust before photography
2. **Lighting** — Use even, diffuse lighting (avoid harsh shadows)
3. **Focus** — Ensure coin is in sharp focus
4. **Framing** — Capture full coin with no cropping
5. **Angle** — Flat, directly above the coin

### Photo Guide Feature

The **Coin Photography Guide** (Chat Agent Team 10) reviews your photos and provides specific critiques and improvement tips.

## Storage & Reuse

### Saved Analysis
- Analysis is saved to the coin record (separate obverse/reverse fields)
- Stored as Markdown for readability
- Available in coin detail, collection statistics, and PDF exports

### AI Coverage Metrics
- **Coverage Score** — Calculated as:
  - Both obverse & reverse: 100%
  - One side: 50%
  - Neither: 0%
- **Collection Health** — Tracks % of collection with AI analysis
- **Gap Checklist** — Identifies which coins need analysis

## Advanced: Multi-Agent Analysis

Beyond basic image analysis, the chat agent includes specialized analysis teams:

| Team | Focus | Trigger |
|------|-------|---------|
| **Coin Analysis** | General analysis | Upload coin photos |
| **Coin Grading** | Grade estimation | "Grade this coin" |
| **Coin Photography** | Photo critique | "Review my photography" |
| **Coin Search** | Dealer discovery | "Find coins like this" |
| **Price Trends** | Market analysis | "What's trending?" |

## Limitations

- **Accuracy** — Vision models may misidentify coins, especially rare varieties
- **Inscriptions** — OCR-based text extraction may have errors, especially for worn coins
- **Market Values** — Estimates are based on training data and may lag current prices
- **Authentication** — Not forensic-grade; use for reference, not expert appraisal

## API Endpoints

```
POST   /api/coins/:id/analyze       # Analyze obverse/reverse images for a coin
DELETE /api/coins/:id/analyze       # Delete saved AI analysis
GET    /api/ai-status               # Check AI provider status
```

## Configuration Reference

### Environment Variables (Admin UI)

- `AIProvider` — `anthropic` or `ollama` (required)
- `AnthropicAPIKey` — API key for Claude
- `AnthropicModel` — Claude model name
- `OllamaURL` — Self-hosted Ollama endpoint
- `OllamaModel` — Vision model name
- `OllamaTimeout` — Request timeout in seconds
- `ObversePrompt` — Custom obverse analysis prompt
- `ReversePrompt` — Custom reverse analysis prompt

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Provider shows "Error" | Verify API key / endpoint is accessible |
| Analysis takes very long | Increase `OllamaTimeout` in admin settings |
| Results are inaccurate | Try a different model or provide clearer photos |
| No images to analyze | Upload obverse and reverse photos first |

## Related Features

- [Coin Details](coin-details.md) — Full coin record with analysis storage
- [AI Grading Assistant](ai-grading.md) — Specialized grade estimation
- [Coin Photography Guide](photography-guide.md) — Photo quality feedback
- [Collection Statistics](statistics.md) — AI coverage metrics in health scorecard
- [Collection Showcase](collection-showcase.md) — Share analyzed coins publicly

## See Also

- [AI Coin Search Agent](ai-search-agent.md) — Powered by same multi-agent architecture
- [Price Trend Analysis](price-trends.md) — Market insights using agent
