import pytest
from pydantic import ValidationError

from app.models.requests import LLMConfig, WishlistURLExtractionRequest
from app.models.responses import WishlistURLHypothesis, WishlistURLHypothesisField
from app.teams import wishlist_url_extraction
from app.teams.wishlist_url_extraction import (
    _validated_listing_hypothesis,
    extract_wishlist_url,
)


def _request() -> WishlistURLExtractionRequest:
    return WishlistURLExtractionRequest(
        llm=LLMConfig(provider="anthropic", api_key="test", model="test"),
        source_url="https://dealer.example/lot/42",
        page_title="Hadrian Silver Denarius",
        page_text=(
            "Hadrian Silver Denarius\n"
            "HADRIANVS AVG COS III P P\n"
            "Reverse: Roma seated left\n"
            "Weight 3.12 g. Diameter 18 mm. USD 125. Available."
        ),
        page_metadata={"site_name": "Example Coins"},
    )


def _field(value: str, evidence: str) -> WishlistURLHypothesisField:
    return WishlistURLHypothesisField(
        value=value,
        confidence=0.9,
        evidence=[evidence],
    )


def test_listing_hypothesis_reuses_shared_normalization_and_numeric_values():
    result = _validated_listing_hypothesis(
        WishlistURLHypothesis(
            name=_field("Hadrian Silver Denarius", "Hadrian Silver Denarius"),
            category=_field("roman", "Hadrian Silver Denarius"),
            material=_field("silver", "Silver"),
            weightGrams=_field("3.12 g", "Weight 3.12 g"),
            diameterMm=_field("18 mm", "Diameter 18 mm"),
            listedPrice=_field("USD 125", "USD 125"),
            currency=_field("usd", "USD 125"),
            listingStatus=_field("available", "Available"),
        ),
        _request(),
    )

    assert result.category is not None and result.category.value == "Roman"
    assert result.material is not None and result.material.value == "Silver"
    assert result.weightGrams is not None and result.weightGrams.value == "3.12"
    assert result.diameterMm is not None and result.diameterMm.value == "18"
    assert result.listedPrice is not None and result.listedPrice.value == "125"
    assert result.currency is not None and result.currency.value == "USD"


def test_listing_hypothesis_drops_fields_without_verbatim_page_evidence():
    result = _validated_listing_hypothesis(
        WishlistURLHypothesis(
            ruler=_field("Marcus Aurelius", "M AVR ANTONINVS"),
            mint=_field("Rome", "Rome mint"),
            obverseInscription=_field(
                "HADRIANVS AVG COS III P P",
                "HADRIANVS AVG COS III P P",
            ),
        ),
        _request(),
    )

    assert result.ruler is None
    assert result.mint is None
    assert result.obverseInscription is not None


def test_listing_hypothesis_keeps_legends_separate_from_descriptions():
    result = _validated_listing_hypothesis(
        WishlistURLHypothesis(
            obverseInscription=_field(
                "HADRIANVS AVG COS III P P",
                "HADRIANVS AVG COS III P P",
            ),
            reverseDescription=_field("Roma seated left", "Roma seated left"),
        ),
        _request(),
    )

    assert result.obverseInscription is not None
    assert result.reverseDescription is not None
    assert result.obverseDescription is None
    assert result.reverseInscription is None


@pytest.mark.parametrize("status", ["sold", "reserved", "withdrawn"])
def test_listing_hypothesis_preserves_explicit_listing_status(status: str):
    request = _request().model_copy(
        update={"page_text": _request().page_text + f"\nStatus: {status}"}
    )
    result = _validated_listing_hypothesis(
        WishlistURLHypothesis(
            listingStatus=_field(status, f"Status: {status}"),
        ),
        request,
    )
    assert result.listingStatus is not None
    assert result.listingStatus.value == status


def test_wishlist_hypothesis_rejects_malformed_model_output():
    with pytest.raises(ValidationError):
        WishlistURLHypothesis.model_validate(
            {
                "name": {
                    "value": "Hadrian",
                    "confidence": 2,
                    "evidence": ["Hadrian"],
                },
                "unexpected_write_instruction": "create now",
            }
        )


@pytest.mark.asyncio
async def test_extract_wishlist_url_unwraps_include_raw_result(monkeypatch):
    parsed = WishlistURLHypothesis(
        name=_field("Hadrian Silver Denarius", "Hadrian Silver Denarius"),
    )

    monkeypatch.setattr(
        wishlist_url_extraction,
        "get_structured_model",
        lambda *_args: object(),
    )

    async def invoke(*_args, **_kwargs):
        return {"raw": object(), "parsed": parsed, "parsing_error": None}

    monkeypatch.setattr(wishlist_url_extraction, "ainvoke_with_retry", invoke)

    response = await extract_wishlist_url(_request())

    assert response.hypothesis.name is not None
    assert response.hypothesis.name.value == "Hadrian Silver Denarius"


@pytest.mark.asyncio
async def test_extract_wishlist_url_rejects_unparsed_include_raw_result(monkeypatch):
    monkeypatch.setattr(
        wishlist_url_extraction,
        "get_structured_model",
        lambda *_args: object(),
    )

    async def invoke(*_args, **_kwargs):
        return {
            "raw": object(),
            "parsed": None,
            "parsing_error": ValueError("malformed"),
        }

    monkeypatch.setattr(wishlist_url_extraction, "ainvoke_with_retry", invoke)

    with pytest.raises(ValueError, match="did not return a parsed result"):
        await extract_wishlist_url(_request())
