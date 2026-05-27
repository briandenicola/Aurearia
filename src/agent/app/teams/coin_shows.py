"""Team 2: Coin Shows — multi-agent pipeline with date verification.

Pipeline: Search Agent → Date Verification Agent → Formatter Agent

- Search Agent: finds upcoming coin shows via web search
- Date Verification Agent: confirms show dates are in the future and not cancelled
- Formatter Agent: structures verified shows into CoinShow JSON schema
"""

import json
import logging
import re
from typing import Annotated, TypedDict
from urllib.parse import urlparse

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
from langgraph.graph import END, StateGraph

from app.llm.provider import create_search_agent, get_chat_model, get_search_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig, UserContext
from app.safety import with_safety

logger = logging.getLogger(__name__)

SEARCH_PROMPT = with_safety("""You are a coin show search specialist. Your ONLY job is to search for
upcoming coin shows, numismatic conventions, and coin collecting events.

CRITICAL RULES:
- Use your search tool to find REAL coin shows and events
- Search multiple sources: coinshows.com, coinshows-usa.com, ANA (money.org),
  PNG (pngdealers.org), NYINC, Whitman Expo, local coin club sites, Eventbrite
- Focus on shows within the next 30 days unless the user specifies a different timeframe
- NEVER invent event details from memory

URL RULES (CRITICAL):
- For EACH result, use ONLY URLs that appeared verbatim in your web search results
- NEVER construct, guess, or modify a URL path — copy it exactly as returned
- If you are unsure of the exact URL, use the search results page URL instead
- A real URL from a search result is ALWAYS better than a fabricated "correct-looking" one

IMPORTANT — DO NOT SELF-FILTER:
- Your job is to SEARCH and REPORT, not to verify or validate dates
- A separate verification step handles date and location validation
- INCLUDE recurring/regular shows (e.g. "first weekend of each month") with their
  schedule pattern — the verifier will compute specific upcoming dates
- If search results mention a show but lack an exact date, still include it with
  whatever schedule information is available (e.g. "monthly", "every first Saturday")
- When in doubt, INCLUDE the show — the verifier will remove anything invalid

{location_context}

For each show found, output a JSON object with these fields:
- name: the show/event name
- dates: the show dates as listed, OR the recurring schedule (e.g. "First full weekend
  of each month" or "March 15-17, 2025")
- location: city/state/country
- venue: venue name if available
- url: the EXACT URL from your search results (copy verbatim — do not construct)
- description: brief description
- entryFee: entry fee if listed
- dealers: notable dealers or "bourse" info if listed

Output your results as a JSON array wrapped in ```json and ``` markers.
If you find nothing at all, return an empty array: ```json\n[]\n```""")

DATE_VERIFY_PROMPT = with_safety("""You are a date and location verification specialist for coin show events.
You will receive coin show data and the user's location context. Your job is to verify shows.

Be INCLUSIVE, not exclusive. If there is reasonable evidence a show is happening
(listed on a coin show directory, has a venue, has a recurring schedule), KEEP it.
Only remove shows with clear disqualifying evidence.

Today's date context will be provided. REMOVE a show ONLY if:
- The dates are clearly in the PAST (already happened)
- The show is explicitly marked as CANCELLED or POSTPONED INDEFINITELY
- The user asked for nearby shows and the show is clearly NOT within ~50 miles

KEEP shows where:
- Dates are clearly in the future
- Show has a RECURRING schedule (e.g. "first weekend of each month", "monthly",
  "every third Saturday"). For these, COMPUTE the next occurrence from today's date
  and set the dates field to that computed date (e.g. "April 5-6, 2026")
- Show appears to be actively scheduled (not cancelled)
- Information comes from a reputable source (coinshows.com, coinshows-usa.com,
  money.org, coin club websites, etc.)
- If user asked for nearby shows, the location is within reasonable driving
  distance (~50 miles) of their location or ZIP code

IMPORTANT: Preserve ALL URLs exactly as provided. Do NOT modify, construct, or
"fix" any URL. Copy them character-for-character from the input.

Output the VERIFIED shows as a JSON array with the same fields.
For recurring shows, update the "dates" field with the computed next occurrence.
Wrap in ```json and ``` markers. If none pass verification, return an empty array.""")

FORMAT_PROMPT = with_safety("""You are a formatting specialist for a coin collecting application.
You receive verified coin show data. Structure each into this exact JSON schema:

```json
[
  {
    "name": "Full show name",
    "dates": "Human-readable date range e.g. March 15-17, 2025",
    "location": "City, State/Province, Country",
    "venue": "Venue name",
    "url": "Source URL — MUST be copied exactly from the input, never modified",
    "description": "Brief description of the show",
    "entryFee": "Entry fee or 'Free' or 'Unknown'",
    "notableDealers": ["Dealer 1", "Dealer 2"]
  }
]
```

Rules:
- Use ONLY data from the verified shows. Do NOT invent any fields.
- url MUST be copied character-for-character from the verified data. NEVER construct,
  guess, or "fix" a URL. If the input URL looks wrong, keep it exactly as-is.
- Normalize dates into a human-readable format
- Normalize location into "City, State, Country" format
- If a field is unknown, use empty string or empty array
- Sort by date (soonest first)

Output ONLY the JSON array wrapped in ```json and ``` markers.""")


class CoinShowSearchState(TypedDict):
    """State flowing through the coin show search pipeline."""

    messages: Annotated[list, lambda a, b: a + b]
    search_results: str
    verification_results: str
    formatted_results: str
    user_message: str
    location_context: str


def create_coin_show_team(
    llm_config: LLMConfig,
    user_context: UserContext | None = None,
    search_prompt: str = "",
):
    """Create the Team 2 coin show search graph.

    Args:
        llm_config: LLM provider configuration
        user_context: Optional user context with zip code for location
        search_prompt: Additional search context from admin settings (prepended to system prompt)
    """
    # The admin prompt provides personality/context; SEARCH_PROMPT provides structure
    if search_prompt:
        base_search_prompt = f"{search_prompt}\n\n{SEARCH_PROMPT}"
    else:
        base_search_prompt = SEARCH_PROMPT
    logger.debug("Coin shows prompt (%d chars): %.80s...", len(base_search_prompt), base_search_prompt)

    model = get_chat_model(llm_config)
    use_react_agent = llm_config.provider == "ollama"
    if use_react_agent:
        search_agent = create_search_agent(llm_config)
    else:
        search_model = get_search_model(llm_config)

    # Build location context from user's zip code
    location_hint = ""
    if user_context and user_context.zip_code:
        location_hint = (
            f"The user is located near ZIP code {user_context.zip_code}. "
            "Prioritize shows near this location, but also include major "
            "national shows."
        )

    async def search_node(state: CoinShowSearchState) -> dict:
        """Search Agent: finds upcoming coin shows via web search."""
        user_msg = state.get("user_message", "")
        loc_ctx = state.get("location_context", "") or location_hint
        logger.debug("[coin_shows] search_node start — query: %.100s, location: %.60s", user_msg, loc_ctx)

        # Inject location context if the prompt has a placeholder, otherwise append
        if "{location_context}" in base_search_prompt:
            prompt = base_search_prompt.format(
                location_context=loc_ctx if loc_ctx else "No specific location preference."
            )
        elif loc_ctx:
            prompt = f"{loc_ctx}\n\n{base_search_prompt}"
        else:
            prompt = base_search_prompt

        if use_react_agent:
            # Ollama: ReAct agent calls SearXNG tool autonomously
            messages = [
                SystemMessage(content=prompt),
                HumanMessage(content=f"Search for: {user_msg}"),
            ]
            result = await search_agent.ainvoke({"messages": messages})
            last_msg = result["messages"][-1]
            response_content = last_msg.content if isinstance(last_msg.content, str) else str(last_msg.content)
            logger.debug(
                "[coin_shows] ReAct agent returned %d messages, content=%d chars",
                len(result["messages"]), len(response_content),
            )
        else:
            # Anthropic: web_search enabled via get_search_model
            messages = [
                SystemMessage(content=prompt),
                HumanMessage(content=f"Search for: {user_msg}"),
            ]
            response = await ainvoke_with_retry(search_model, messages)
            response_content = response.content if isinstance(response.content, str) else str(response.content)
            logger.debug("[coin_shows] Anthropic search response=%d chars", len(response_content))

        return {
            "search_results": response_content,
            "messages": [],
        }

    async def verify_node(state: CoinShowSearchState) -> dict:
        """Verification Agent: confirms dates are future, not cancelled, and location is nearby."""
        search_results = state.get("search_results", "")
        logger.debug("[coin_shows] verify_node start — search_results=%d chars", len(search_results))

        if not search_results or _is_empty_json_result(search_results):
            return {
                "verification_results": "No shows found to verify.",
                "messages": [],
            }

        from datetime import datetime, timezone

        today = datetime.now(tz=timezone.utc).strftime("%B %d, %Y")
        loc_ctx = state.get("location_context", "") or location_hint

        location_note = ""
        if loc_ctx:
            distance_limit = "~50 miles"
            distance_match = re.search(
                r"\bwithin\s+(\d+)\s*(miles?|mi|kilometers?|kms?|km)\b",
                loc_ctx,
                flags=re.IGNORECASE,
            )
            if distance_match:
                unit = distance_match.group(2).lower()
                normalized_unit = "miles" if unit in {"mile", "miles", "mi"} else "km"
                distance_limit = f"{distance_match.group(1)} {normalized_unit}"
            location_note = (
                f"\n\nUser location context: {loc_ctx}\n"
                f"Use a nearby radius of approximately {distance_limit}. "
                "Filter out shows clearly outside this distance."
            )

        messages = [
            SystemMessage(content=DATE_VERIFY_PROMPT),
            HumanMessage(
                content=f"Today's date is: {today}{location_note}\n\n"
                f"Coin show search results:\n{search_results}\n\n"
                "Verify dates are future, show is not cancelled, and location is reasonable."
            ),
        ]
        response = await ainvoke_with_retry(model, messages)

        return {
            "verification_results": response.content if isinstance(response.content, str) else str(response.content),
            "messages": [],
        }

    async def format_node(state: CoinShowSearchState) -> dict:
        """Formatter Agent: structures verified shows into CoinShow schema."""
        verified = state.get("verification_results", "")
        user_msg = state.get("user_message", "")
        logger.debug("[coin_shows] format_node start — verified=%d chars", len(verified))

        has_no_results = (
            "no shows found" in verified.lower()
            or "empty array" in verified.lower()
            or _is_empty_json_result(verified)
        )

        if has_no_results:
            # Call the LLM so the response streams to the user via SSE
            no_results_prompt = (
                "You are an assistant in a coin collecting application. "
                "The user searched for coin shows but no verified results were found. "
                "Generate a brief, helpful response. Suggest specific resources: "
                "coinshows.com, coinshows-usa.com, money.org (ANA), and their "
                "state numismatic association. Keep it concise. Do not use emojis. "
                "Do not invent show details."
            )
            messages = [
                SystemMessage(content=no_results_prompt),
                HumanMessage(
                    content=f"The user asked: {user_msg}\n\n"
                    "No verified coin shows were found. Generate a helpful response."
                ),
            ]
            response = await ainvoke_with_retry(model, messages)
            content = response.content if isinstance(response.content, str) else str(response.content)
            return {"formatted_results": "", "messages": [AIMessage(content=content)]}

        messages = [
            SystemMessage(content=FORMAT_PROMPT),
            HumanMessage(
                content=f"User was searching for: {user_msg}\n\n"
                f"Verified shows:\n{verified}\n\n"
                "Format these into the required JSON schema."
            ),
        ]
        response = await ainvoke_with_retry(model, messages)
        formatted = response.content if isinstance(response.content, str) else str(response.content)

        # Sanitize URLs: replace any fabricated URLs with originals from search results
        search_results = state.get("search_results", "")
        formatted = _sanitize_urls(formatted, search_results, verified)

        count = _count_shows(formatted)
        summary = (
            f"I found {count} upcoming coin show{'s' if count != 1 else ''} "
            "matching your search. All dates have been verified as upcoming events."
        )

        return {
            "formatted_results": formatted,
            "messages": [AIMessage(content=f"{summary}\n\n{formatted}")],
        }

    graph = StateGraph(CoinShowSearchState)
    graph.add_node("search", search_node)
    graph.add_node("verify", verify_node)
    graph.add_node("format", format_node)

    graph.set_entry_point("search")
    graph.add_edge("search", "verify")
    graph.add_edge("verify", "format")
    graph.add_edge("format", END)

    return graph.compile()


def _count_shows(text: str) -> int:
    """Count the number of shows in a JSON block."""
    json_str = _extract_json_block(text)
    if json_str:
        try:
            data = json.loads(json_str)
            if isinstance(data, list):
                return len(data)
        except json.JSONDecodeError:
            pass
    return 0


def _extract_json_block(text: str) -> str | None:
    """Extract the first ```json ... ``` block from text."""
    start = text.find("```json")
    if start == -1:
        return None
    start += len("```json")
    end = text.find("```", start)
    if end == -1:
        return None
    return text[start:end].strip()


def _is_empty_json_result(text: str) -> bool:
    """Check if text contains a JSON block with an empty array."""
    json_block = _extract_json_block(text)
    if json_block:
        try:
            data = json.loads(json_block)
            return isinstance(data, list) and len(data) == 0
        except json.JSONDecodeError:
            pass
    return False


def _extract_urls_from_text(text: str) -> list[str]:
    """Extract all http/https URLs from free text."""
    return re.findall(r'https?://[^\s"\'<>)\],]+', text)


def _sanitize_urls(
    formatted: str, search_results: str, verified: str, url_field: str = "url"
) -> str:
    """Replace fabricated URLs in formatted output with real ones from search results.

    LLMs often modify URLs across pipeline hops (search -> verify -> format).
    This extracts real URLs from search and verify stages, then checks each URL
    in the formatted output. Fabricated URLs are replaced with the best matching
    real URL from the same domain.
    """
    real_urls = set(_extract_urls_from_text(search_results))
    real_urls.update(_extract_urls_from_text(verified))

    if not real_urls:
        return formatted

    json_block = _extract_json_block(formatted)
    if not json_block:
        return formatted

    try:
        items = json.loads(json_block)
        if not isinstance(items, list):
            return formatted
    except json.JSONDecodeError:
        return formatted

    changed = False
    for item in items:
        url = item.get(url_field, "")
        if not url or url in real_urls:
            continue

        try:
            fabricated_domain = urlparse(url).netloc.lower()
        except Exception:
            continue

        domain_matches = [u for u in real_urls if fabricated_domain in urlparse(u).netloc.lower()]
        if domain_matches:
            item[url_field] = domain_matches[0]
            changed = True
            logger.warning("Replaced fabricated URL %s with real URL %s", url, item[url_field])
        else:
            logger.warning("Fabricated URL %s has no domain match in search results", url)

    if changed:
        new_json = json.dumps(items, indent=2)
        start = formatted.find("```json")
        end_marker = formatted.find("```", start + len("```json"))
        if start != -1 and end_marker != -1:
            formatted = formatted[:start] + "```json\n" + new_json + "\n" + formatted[end_marker:]

    return formatted
