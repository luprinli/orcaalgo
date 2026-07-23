from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.ir import (
    CANONICAL_VERSION_V04,
    IR_VERSION_V04,
    NodeV04,
    StrategyBodyV04,
    StrategyIRV04,
    load_ir_v04,
)


def test_v04_minimal_strategy_loads() -> None:
    ir = StrategyIRV04(strategy=StrategyBodyV04(id="empty"))

    assert ir.ir_version == IR_VERSION_V04
    assert ir.canonical_version == CANONICAL_VERSION_V04
    assert ir.strategy.nodes == []
    assert ir.metadata == {}


def test_v04_opaque_node_shell_loads() -> None:
    ir = StrategyIRV04(
        strategy=StrategyBodyV04(
            id="shell",
            nodes=[
                NodeV04(
                    id="n2",
                    token="custom.placeholder",
                    version=1,
                    inputs={"x": "$externals.market"},
                    params={"lookback": 9},
                )
            ],
            outputs={"main": "n2.value"},
        ),
        metadata={"owner": "wp1"},
    )

    assert ir.strategy.nodes[0].token == "custom.placeholder"
    assert ir.strategy.outputs == {"main": "n2.value"}


def test_invalid_ir_version_fails() -> None:
    with pytest.raises(ValidationError):
        StrategyIRV04.model_validate(
            {
                "ir_version": "qst-ir/0.2",
                "canonical_version": CANONICAL_VERSION_V04,
                "strategy": {"id": "bad"},
                "metadata": {},
            }
        )


def test_metadata_must_be_canonical_json_compatible() -> None:
    with pytest.raises(ValidationError):
        StrategyIRV04(
            strategy=StrategyBodyV04(id="bad"),
            metadata={"not_json": object()},
        )


def test_duplicate_node_ids_fail() -> None:
    with pytest.raises(ValidationError):
        StrategyIRV04(
            strategy=StrategyBodyV04(
                id="bad",
                nodes=[NodeV04(id="n"), NodeV04(id="n")],
            )
        )


def test_load_ir_v04_from_yaml_text() -> None:
    ir = load_ir_v04(
        """
        ir_version: qst-ir/0.4
        canonical_version: qst-canonical/0.4
        strategy:
          id: yaml_shell
          version: 1
          nodes: []
          outputs: {}
        metadata: {}
        """
    )

    assert ir.strategy.id == "yaml_shell"
