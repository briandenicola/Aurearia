"""Coin intake draft generation from observation images and optional coin card."""

import json
import logging

from langchain_core.messages import HumanMessage, SystemMessage
from pydantic import ValidationError

from app.llm.provider import get_chat_model
from app.llm.retry import ainvoke_with_retry
from app.models.requests import LLMConfig
from app.models.responses import IntakeDraftResponse
from app.safety import with_safety
from app.teams.coin_analysis import _build_image_contents

logger = logging.getLogger(__name__)

INTAKE_PROMPT = with_safety("""You are a numismatic intake assistant.
Analyze the provided coin observation images and optional coin-card image.
Return a single JSON object only.

Required JSON shape:
{
  "coin": {
    "name": "",
    "category": "",
    "material": "",
    "denomination": "",
    "ruler": "",
    "era": "",
    "mint": "",
    "obverse_inscription": "",
    "reverse_inscription": "",
    "obverse_description": "",
    "reverse_description": "",
    "notes": ""
  },
  "confidenceSummary": {
    "overall": "low|medium|high",
    "uncertainFields": []
  },
  "evidence": [
    {
      "type": "vision|coin_card",
      "source": "",
      "field": "",
      "value": "",
      "confidence": "low|medium|high",
      "notes": ""
    }
  ],
  "unresolvedFields": []
}

Rules:
- Use snake_case keys inside "coin"
- Leave unknown values as empty strings; do not invent data
- Keep notes concise and practical
- Do not use markdown or code fences in the final output
- Do not use emojis""")


def _extract_json_payload(raw: str) -> str:
    start = raw.find("```json")
    if start != -1:
        start += len("```json")
        end = raw.find("```", start)
        if end != -1:
            return raw[start:end].strip()
        return raw[start:].strip()
    return raw.strip()


async def generate_intake_draft(
    llm_config: LLMConfig,
    images: list[str],
    coin_card_image: str = "",
) -> IntakeDraftResponse:
    """Generate a structured intake draft from coin images."""
    if not images:
        return IntakeDraftResponse(unresolved_fields=["observation_images"])

    model = get_chat_model(llm_config)
    human_content = [{"type": "text", "text": INTAKE_PROMPT}]
    human_content.extend(_build_image_contents(images))
    if coin_card_image:
        human_content.extend(_build_image_contents([coin_card_image]))

    messages = [
        SystemMessage(content="You extract structured numismatic draft fields from images."),
        HumanMessage(content=human_content),
    ]

    try:
        response = await ainvoke_with_retry(model, messages)
        content = response.content if isinstance(response.content, str) else str(response.content)
        payload = _extract_json_payload(content)
        parsed = json.loads(payload)
        return IntakeDraftResponse.model_validate(parsed)
    except (json.JSONDecodeError, ValidationError):
        logger.exception("Intake draft parsing failed")
    except Exception:
        logger.exception("Intake draft generation failed")

    return IntakeDraftResponse(
        coin={},
        confidence_summary={"overall": "low", "uncertain_fields": ["all"]},
        evidence=[],
        unresolved_fields=["name", "category", "material", "denomination"],
    )
