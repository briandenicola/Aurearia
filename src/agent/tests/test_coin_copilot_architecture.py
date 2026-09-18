"""Feature 359 Python architecture and capability-surface guards."""

import ast
from pathlib import Path

from app.models.requests import COPILOT_ALLOWED_TOOLS
from app.tools.copilot_collection_tools import CALLBACK_TOOLS

ROOT = Path(__file__).parents[1] / "app"
COPILOT_FILES = [
    ROOT / "teams" / "coin_copilot.py",
    ROOT / "tools" / "copilot_collection_tools.py",
    ROOT / "llm" / "capabilities.py",
]


def test_coin_copilot_imports_no_database_or_execution_capability():
    forbidden_roots = {
        "sqlite3",
        "sqlalchemy",
        "subprocess",
        "socket",
        "shutil",
        "pathlib",
    }
    for path in COPILOT_FILES:
        tree = ast.parse(path.read_text(encoding="utf-8"))
        imports = {
            alias.name.split(".")[0]
            for node in ast.walk(tree)
            if isinstance(node, (ast.Import, ast.ImportFrom))
            for alias in node.names
        }
        assert not imports.intersection(forbidden_roots), (path, imports)


def test_coin_copilot_imports_no_write_or_escalation_modules():
    forbidden_modules = {
        "app.teams.coin_intake",
        "app.teams.deep_identification",
        "app.tools.provider_tools",
        "app.tools.update_proposals",
        "app.tools.web_search",
    }
    forbidden_fragments = {
        "approval",
        "arbitrary_http",
        "database",
        "deep_identification",
        "filesystem",
        "mutation",
        "shell",
        "write",
    }
    for path in COPILOT_FILES:
        tree = ast.parse(path.read_text(encoding="utf-8"))
        modules = {
            node.module
            for node in ast.walk(tree)
            if isinstance(node, ast.ImportFrom) and node.module is not None
        }
        modules.update(
            alias.name
            for node in ast.walk(tree)
            if isinstance(node, ast.Import)
            for alias in node.names
        )
        assert not modules.intersection(forbidden_modules), (path, modules)
        exported_names = {
            node.id.lower()
            for node in ast.walk(tree)
            if isinstance(node, ast.Name)
        }
        assert not {
            name
            for name in exported_names
            if any(fragment in name for fragment in forbidden_fragments)
        }, (path, exported_names)


def test_coin_copilot_exposes_only_locked_read_only_capabilities():
    assert COPILOT_ALLOWED_TOOLS == {
        "search_my_collection",
        "get_coin",
        "collection_summary",
        "top_coins_by_value",
        "portfolio_review",
        "gap_analysis",
        "market_search",
        "auction_search",
        "price_trends",
        "similar_lots",
    }
    assert CALLBACK_TOOLS == {
        "search_my_collection",
        "get_coin",
        "collection_summary",
        "top_coins_by_value",
    }
    forbidden = {
        "web_search",
        "propose_update",
        "commit_update",
        "deep_identification",
        "filesystem",
        "shell",
        "database",
        "memory",
    }
    assert not COPILOT_ALLOWED_TOOLS.intersection(forbidden)
