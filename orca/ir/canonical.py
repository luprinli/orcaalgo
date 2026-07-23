from __future__ import annotations

from orca.models.strategy import StrategyIRV04


def canonicalize_ir(ir: StrategyIRV04) -> bytes:
    from orca.hash.common import stable_json_bytes

    data = ir.model_dump(mode="json")
    data["strategy"]["nodes"] = sorted(data["strategy"]["nodes"], key=lambda n: n["id"])
    return stable_json_bytes(data)
