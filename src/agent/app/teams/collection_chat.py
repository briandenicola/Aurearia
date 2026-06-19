"""Team: Collection Chat — ReAct agent for owned-coin queries and updates.

Handles user queries about their OWN collection: ownership lookups, valuations,
searches, updates. The LLM decides which tools to call and can make MULTIPLE
tool calls in one turn to satisfy compound/multi-intent questions.

Examples:
- "Do I have any moose coins?" → search_my_collection
- "How much are they worth?" → search_my_collection + top_coins_by_value
- "Do I have moose coins AND how much are they worth?" → both tools in one turn
- "Update my Constantine coin's notes to..." → propose_update → commit_update

The agent uses the internal tool layer (Go /api/internal/tools endpoints) via
build_collection_tools. Identity flows through the short-lived internal token.
"""

import logging

from langgraph.prebuilt import create_react_agent

from app.llm.provider import get_chat_model
from app.models.requests import AppContext, LLMConfig
from app.safety import with_safety
from app.tools.collection_tools import build_collection_tools

logger = logging.getLogger(__name__)

COLLECTION_AGENT_PROMPT = with_safety("""You are a numismatic assistant helping a collector
manage and understand their personal coin collection.

You have access to tools that let you:
1. Search the user's collection by various filters (search_my_collection)
2. Get detailed information about a specific coin (get_coin)
3. Get aggregate collection statistics (collection_summary)
4. Find the most valuable coins in the collection (top_coins_by_value)
5. Propose updates to coin fields (propose_update)
6. Commit approved updates (commit_update)

IMPORTANT RULES:
- Answer questions about the coins the user ALREADY OWNS — never recommend purchases
- Call MULTIPLE tools in a single turn when needed to fully answer compound questions
  (e.g., "do I have X coins AND how much are they worth?" → search + value lookup)
- NEVER invent coin data — only report what the tools return
- Use search_my_collection for data-quality questions such as coins missing size,
  diameter, weight, grade, value, references, notes, or other metadata. Treat
  "size" as the coin's diameterMm field.
- Use collection_summary when the user asks for counts of missing properties
  across the whole collection.
- For updates, ALWAYS use propose_update first to show the user what will change,
  then require explicit user confirmation before calling commit_update
- Surface the proposal_id and token to the user in your response so they can confirm
- Do not use emojis
- Be concise but thorough
- If a search returns no results, suggest alternative search terms

When the user asks a compound question (multiple intents), think step-by-step
about which tools you need, then call them ALL in one turn.

Example flow for "Do I have moose coins and how much are they worth?":
1. search_my_collection(query="moose") → find moose coins
2. If found, report details and values from the search results

Example flow for updates:
1. propose_update(coin_id=X, changes={...}) → show user the proposal
2. Tell user: "Please confirm if you want to commit this update."
3. On confirmation: commit_update(proposal_id=..., token=..., confirm=True)
""")


def create_collection_chat_team(
    llm_config: LLMConfig,
    tools_base_url: str,
    internal_token: str,
    app_context: AppContext | None = None,
):
    """Build a ReAct agent for collection queries with internal tool layer.

    Args:
        llm_config: LLM configuration (provider, model, API keys)
        tools_base_url: Base URL for Go internal endpoints
        internal_token: Short-lived JWT token for authenticated tool calls
        app_context: Optional frontend route context for resolving "this coin"

    Returns:
        A LangGraph compiled graph (ReAct agent).
    """
    logger.info(
        "Building collection chat team (provider=%s, model=%s)",
        llm_config.provider,
        llm_config.model,
    )

    # Get chat model (NO web search)
    model = get_chat_model(llm_config)

    # Build collection tools bound to this request's token
    tools = build_collection_tools(tools_base_url, internal_token)

    prompt = COLLECTION_AGENT_PROMPT
    if app_context and (app_context.active_coin_id or app_context.route):
        context_parts = []
        if app_context.active_coin_id:
            context_parts.append(f"active coin id: {app_context.active_coin_id}")
        if app_context.route:
            context_parts.append(f"current route: {app_context.route}")
        prompt = f"{COLLECTION_AGENT_PROMPT}\n\nCurrent app context: {', '.join(context_parts)}."

    # Create ReAct agent with the system prompt
    agent = create_react_agent(
        model,
        tools,
        prompt=prompt,
    )

    logger.debug("Collection chat team built with %d tools", len(tools))
    return agent
