from app.llm.content import extract_text_content


def test_extract_text_content_ignores_anthropic_thinking_and_signature() -> None:
    content = [
        {
            "type": "thinking",
            "thinking": "private reasoning",
            "signature": "encoded-signature",
        },
        {
            "type": "text",
            "text": "The evidence supports a Roman denarius attribution.",
        },
    ]

    result = extract_text_content(content)

    assert result == "The evidence supports a Roman denarius attribution."
    assert "thinking" not in result
    assert "signature" not in result


def test_extract_text_content_rejects_non_text_transport_content() -> None:
    assert extract_text_content([{"type": "thinking", "signature": "secret"}]) == ""
    assert extract_text_content({"type": "image", "source": {"data": "encoded"}}) == ""


def test_extract_text_content_preserves_plain_string_blocks() -> None:
    assert extract_text_content(["First finding.", "Second finding."]) == (
        "First finding.\n\nSecond finding."
    )
