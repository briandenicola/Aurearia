from app.models.requests import UserContext
from app.supervisor import _build_coin_show_location_context


def test_uses_default_zip_when_no_override_present():
    location_ctx, has_location = _build_coin_show_location_context(
        "Find upcoming coin shows near me this weekend",
        UserContext(user_id=1, zip_code="10001"),
    )

    assert has_location is True
    assert "ZIP code 10001" in location_ctx
    assert "default ZIP code" not in location_ctx


def test_explicit_location_with_distance_overrides_default_zip():
    location_ctx, has_location = _build_coin_show_location_context(
        "Find coin shows within 120 miles of Chicago, IL instead of my home zip",
        UserContext(user_id=1, zip_code="10001"),
    )

    assert has_location is True
    assert "near Chicago, IL" in location_ctx
    assert "within 120 miles" in location_ctx
    assert "instead of their default ZIP code 10001" in location_ctx


def test_distance_without_new_location_uses_default_zip_radius():
    location_ctx, has_location = _build_coin_show_location_context(
        "Show me coin events within 75 miles",
        UserContext(user_id=1, zip_code="10001"),
    )

    assert has_location is True
    assert "ZIP code 10001 within 75 miles" in location_ctx


def test_requires_location_when_no_default_zip_and_no_location_in_message():
    location_ctx, has_location = _build_coin_show_location_context(
        "Find upcoming coin shows for me",
        UserContext(user_id=1, zip_code=""),
    )

    assert has_location is False
    assert location_ctx == ""
