"""Deep Agentic Coin Identification pipeline (344-deep-agentic-coin-identification).

Contract anchor: specs/344-deep-agentic-coin-identification/contracts/
agent-internal-contract.md §6 (graph topology):

    prepare_evidence -> router -> provider_fanout (bounded) -> evaluator -> synthesizer

This package is stateless and DB-free (Principle II). Every automated
provider call goes through app/tools/provider_tools.py, which is the sole
HTTP boundary to the Go internal tool endpoints — no provider is ever
contacted directly from here.
"""
