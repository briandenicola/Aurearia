"""Shared helper for turning structured coin data into a search query description.

Used by any team that needs to describe a coin/lot in free text for a web
search (Team 10 similar-lots, the bid market-signal team), so this logic
isn't duplicated per team.
"""

from app.models.requests import CoinData


def build_coin_description(coin: CoinData) -> str:
    parts = []
    if coin.name:
        parts.append(f"Name: {coin.name}")
    if coin.category:
        parts.append(f"Category: {coin.category}")
    if coin.denomination:
        parts.append(f"Denomination: {coin.denomination}")
    if coin.ruler:
        parts.append(f"Ruler: {coin.ruler}")
    if coin.era:
        parts.append(f"Era: {coin.era}")
    if coin.material:
        parts.append(f"Material: {coin.material}")
    if coin.grade:
        parts.append(f"Grade: {coin.grade}")
    return "\n".join(parts) if parts else "Unknown coin"
