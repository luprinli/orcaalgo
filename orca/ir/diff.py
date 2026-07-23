"""GKR strategy diff engine.

Compares two .gkr.yaml strategy files and returns structured differences:
parameter deltas, topology changes (added/removed nodes), and hash comparison.
"""

from __future__ import annotations

from pathlib import Path

from orca.hash.graph import graph_hash_v2
from orca.ir.loader import load_ir


def diff_strategies(path_a: str | Path, path_b: str | Path) -> dict:
    """Compare two .gkr.yaml files and return structured diff.

    Returns a dict with keys:
        params: dict of parameter deltas per node
        nodes: {added: [...], removed: [...], changed: [...], unchanged: int}
        outputs: {added: [...], removed: [...], changed: [...], unchanged: int}
        hash: {a: str, b: str, same: bool}
    """
    ir_a = load_ir(Path(path_a))
    ir_b = load_ir(Path(path_b))

    nodes_a = {n.id: n for n in ir_a.strategy.nodes}
    nodes_b = {n.id: n for n in ir_b.strategy.nodes}

    param_deltas: dict[str, dict] = {}
    nodes_added: list[str] = []
    nodes_removed: list[str] = []
    nodes_changed: list[str] = []
    nodes_unchanged: int = 0

    all_ids = set(nodes_a.keys()) | set(nodes_b.keys())

    for node_id in sorted(all_ids):
        na = nodes_a.get(node_id)
        nb = nodes_b.get(node_id)

        if na is None:
            nodes_added.append(node_id)
            continue
        if nb is None:
            nodes_removed.append(node_id)
            continue

        token_changed = na.token_ref.token_id != nb.token_ref.token_id
        params_a = dict(na.params) if na.params else {}
        params_b = dict(nb.params) if nb.params else {}
        param_diff: dict[str, dict] = {}

        all_params = set(params_a.keys()) | set(params_b.keys())
        for p in sorted(all_params):
            va = params_a.get(p)
            vb = params_b.get(p)
            if va != vb:
                param_diff[p] = {"old": str(va), "new": str(vb)}

        if param_diff or token_changed:
            nodes_changed.append(node_id)
            detail: dict = {}
            if token_changed:
                detail["token_id"] = {
                    "old": na.token_ref.token_id,
                    "new": nb.token_ref.token_id,
                }
            if param_diff:
                detail["params"] = param_diff
            param_deltas[node_id] = detail
        else:
            nodes_unchanged += 1

    outs_a = dict(sorted(ir_a.strategy.outputs.items()))
    outs_b = dict(sorted(ir_b.strategy.outputs.items()))
    outs_diff = _diff_dict(outs_a, outs_b)

    hash_a = graph_hash_v2(ir_a)
    hash_b = graph_hash_v2(ir_b)

    return {
        "files": {"a": str(path_a), "b": str(path_b)},
        "graphs": {
            "same": hash_a == hash_b,
            "hash_a": hash_a,
            "hash_b": hash_b,
        },
        "nodes": {
            "added": nodes_added,
            "removed": nodes_removed,
            "changed": nodes_changed,
            "unchanged": nodes_unchanged,
            "deltas": param_deltas,
        },
        "outputs": outs_diff,
    }


def _diff_dict(a: dict, b: dict) -> dict:
    all_keys = set(a) | set(b)
    added: list[str] = []
    removed: list[str] = []
    changed: list[str] = []
    unchanged: int = 0

    for k in sorted(all_keys):
        va = a.get(k)
        vb = b.get(k)
        if k not in a:
            added.append(k)
        elif k not in b:
            removed.append(k)
        elif va != vb:
            changed.append(k)
        else:
            unchanged += 1

    return {"added": added, "removed": removed, "changed": changed, "unchanged": unchanged}
