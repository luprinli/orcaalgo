"""Map Qlib coverage records to qst-ir/0.4 candidate GKR documents."""

from __future__ import annotations

import re
from typing import Any

from qst.adapters.qlib.models import QlibImportCoverage

_SLUG_RE = re.compile(r"[^a-zA-Z0-9_]+")
_RECORD_TYPE = "State[object]"


def build_candidate_gkr(coverage: QlibImportCoverage) -> dict[str, Any]:
    nodes: list[dict[str, Any]] = []
    if coverage.dataset is not None:
        nodes.append(
            _node(
                "dataset",
                "data.qlib_dataset_record",
                coverage.dataset.model_dump(mode="json"),
            )
        )
    if coverage.model is not None:
        nodes.append(
            _node(
                "model",
                "model.forecast_model_record",
                coverage.model.model_dump(mode="json"),
                {"dataset": "dataset.record"} if coverage.dataset is not None else {},
            )
        )
    for index, record in enumerate(coverage.records, start=1):
        local_name = _record_token_name(record.class_name)
        node_id = _record_node_id(record.class_name, index)
        inputs = {"model": "model.record"} if coverage.model is not None else {}
        nodes.append(_node(node_id, local_name, record.model_dump(mode="json"), inputs))
    if coverage.strategy is not None:
        inputs = {"signal": "model.record"} if coverage.model is not None else {}
        nodes.append(
            _node(
                "strategy",
                _strategy_token_name(coverage.strategy.class_name),
                coverage.strategy.model_dump(mode="json"),
                inputs,
            )
        )
    if coverage.backtest is not None:
        inputs = {"strategy": "strategy.record"} if coverage.strategy is not None else {}
        nodes.append(
            _node(
                "backtest_config",
                "portfolio.backtest_config_record",
                coverage.backtest.model_dump(mode="json"),
                inputs,
            )
        )

    strategy_id = _strategy_id(coverage.source)
    output_ref = f"{nodes[-1]['id']}.record" if nodes else ""
    return {
        "schema_version": "qst-ir-schema/0.4",
        "ir_version": "qst-ir/0.4",
        "canonical_version": "qst-canonical/0.4",
        "capabilities": ["core"],
        "strategy": {
            "id": strategy_id,
            "version": 1,
            "nodes": nodes,
            "outputs": {"record": output_ref} if output_ref else {},
        },
        "metadata": {
            "source_adapter": "qlib",
            "source_file": coverage.source,
            "classification": coverage.classification,
            "lossless": False,
            "runtime_execution": False,
            "adapter_warnings": coverage.warnings,
            "unsupported_components": [
                item.model_dump(mode="json") for item in coverage.unsupported_components
            ],
        },
    }


def _node(
    node_id: str,
    token_name: str,
    params: dict[str, Any],
    inputs: dict[str, Any] | None = None,
) -> dict[str, Any]:
    return {
        "id": node_id,
        "token_ref": {
            "namespace": "adapter",
            "name": token_name,
            "version": 1,
            "behavior_version": 1,
        },
        "inputs": inputs or {},
        "params": params,
        "signature": {"outputs": {"record": {"type": _RECORD_TYPE}}},
        "metadata": {"adapter": "qlib", "adapter_local_token": True},
    }


def _record_token_name(class_name: str | None) -> str:
    if class_name == "SignalRecord":
        return "record.signal_record"
    if class_name == "PortAnaRecord":
        return "record.portfolio_analysis_record"
    return "record.generic_record"


def _strategy_token_name(class_name: str | None) -> str:
    if class_name == "TopkDropoutStrategy":
        return "qlib.topk_dropout_strategy_record"
    return "qlib.strategy_generic_record"


def _record_node_id(class_name: str | None, index: int) -> str:
    if class_name == "SignalRecord":
        return "signal_record"
    if class_name == "PortAnaRecord":
        return "portfolio_analysis_record"
    return f"record_{index}"


def _strategy_id(source: str) -> str:
    stem = source.rsplit("/", 1)[-1].rsplit(".", 1)[0]
    slug = _SLUG_RE.sub("_", stem).strip("_").lower()
    return f"qlib_{slug or 'workflow'}"
