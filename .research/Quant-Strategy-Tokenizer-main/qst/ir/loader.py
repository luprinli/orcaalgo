"""Load qst-ir/0.4 shell documents."""

from __future__ import annotations

from collections.abc import Mapping
from pathlib import Path
from typing import Any

import yaml

from qst.ir.schema import StrategyIRV04


def load_ir_v04(value: Mapping[str, Any] | str) -> StrategyIRV04:
    """Load a qst-ir/0.4 strategy from a mapping or YAML/JSON string."""

    if isinstance(value, str):
        raw = yaml.safe_load(value)
        if not isinstance(raw, Mapping):
            raise ValueError("qst-ir/0.4 document must be a mapping")
        return StrategyIRV04.model_validate(raw)

    return StrategyIRV04.model_validate(value)


def load_ir_v04_file(path: str | Path) -> StrategyIRV04:
    """Load a qst-ir/0.4 strategy from disk."""

    return load_ir_v04(Path(path).read_text(encoding="utf-8"))
