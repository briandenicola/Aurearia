from app.llm.content import extract_search_result_urls, extract_search_text, extract_text_content


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


ANTHROPIC_WEB_SEARCH_CONTENT = [
    {"type": "text", "text": "I'll search VCoins for Aurelian coins."},
    {
        "type": "server_tool_use",
        "id": "srvtoolu_1",
        "name": "web_search",
        "input": {"query": "Aurelian antoninianus site:vcoins.com"},
    },
    {
        "type": "web_search_tool_result",
        "tool_use_id": "srvtoolu_1",
        "content": [
            {
                "type": "web_search_result",
                "title": "Aurelian search",
                "url": "https://www.vcoins.com/en/Search.aspx?searchstring=aurelian",
                "encrypted_content": "opaque",
            },
            {
                "type": "web_search_result",
                "title": "Aurelian antoninianus",
                "url": "https://www.ma-shops.com/dealer/item.php?id=1",
                "encrypted_content": "opaque",
            },
        ],
    },
    {
        "type": "text",
        "text": "VCoins lists an Aurelian As.",
        "citations": [
            {
                "type": "web_search_result_location",
                "url": "https://www.vcoins.com/en/stores/x/1/product/aurelian_as/13896/Default.aspx",
                "title": "Aurelian As",
                "cited_text": "Aurelian As",
                "encrypted_index": "opaque",
            }
        ],
    },
]


def test_extract_search_result_urls_reads_result_blocks_and_citations() -> None:
    assert extract_search_result_urls(ANTHROPIC_WEB_SEARCH_CONTENT) == [
        "https://www.vcoins.com/en/Search.aspx?searchstring=aurelian",
        "https://www.ma-shops.com/dealer/item.php?id=1",
        "https://www.vcoins.com/en/stores/x/1/product/aurelian_as/13896/Default.aspx",
    ]


def test_extract_search_text_keeps_prose_and_result_urls_without_transport_fields() -> None:
    result = extract_search_text(ANTHROPIC_WEB_SEARCH_CONTENT)

    assert result.startswith("I'll search VCoins for Aurelian coins.\n\nVCoins lists an Aurelian As.")
    assert "https://www.ma-shops.com/dealer/item.php?id=1" in result
    assert "opaque" not in result
    assert "srvtoolu_1" not in result


def test_extract_search_text_without_results_is_plain_text() -> None:
    assert extract_search_text("Plain answer.") == "Plain answer."
    assert extract_search_result_urls([{"type": "web_search_tool_result", "content": {"error_code": "x"}}]) == []
