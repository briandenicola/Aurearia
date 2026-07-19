"""Shared helper for extracting a fenced (or raw) JSON payload from LLM output.

Used by any team whose analysis step is asked to respond with a ```json
fenced block (availability check, bid market signal), so the extraction
logic isn't duplicated per team.
"""


def extract_json_payload(raw_response: str) -> str:
    """Extract the JSON section from fenced or raw LLM output."""
    start = raw_response.find("```json")
    if start != -1:
        start += len("```json")
        end = raw_response.find("```", start)
        if end != -1:
            return raw_response[start:end].strip()
        return raw_response[start:].strip()

    return raw_response.strip()
