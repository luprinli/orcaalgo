from __future__ import annotations

from pathlib import Path

import yaml

from orca.models.strategy import StrategyIRV04


def load_ir(path: Path) -> StrategyIRV04:
    with open(path) as f:
        data = yaml.safe_load(f)
    return StrategyIRV04(**data)


def save_ir(ir: StrategyIRV04, path: Path) -> None:
    with open(path, "w") as f:
        yaml.dump(ir.model_dump(mode="json"), f, sort_keys=False, allow_unicode=True)
