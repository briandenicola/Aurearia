"""Graph topology test (T067) — verifies node/edge shape without ever
invoking a real LLM or provider call.
"""

from app.teams.deep_identification.graph import build_graph


def test_build_graph_has_expected_node_topology():
    graph = build_graph(model=None, tools=None)
    nodes = set(graph.get_graph().nodes.keys())

    assert nodes == {
        "__start__",
        "prepare_evidence",
        "router",
        "provider_fanout",
        "evaluator",
        "synthesizer",
        "__end__",
    }

    edges = {(edge.source, edge.target) for edge in graph.get_graph().edges}
    assert ("__start__", "prepare_evidence") in edges
    assert ("prepare_evidence", "router") in edges
    assert ("router", "provider_fanout") in edges
    assert ("provider_fanout", "evaluator") in edges
    assert ("evaluator", "synthesizer") in edges
    assert ("synthesizer", "__end__") in edges
