"""Extract record-layer structure from Qlib workflow configs."""

from __future__ import annotations

from typing import Any, cast

from qst.adapters.qlib.models import (
    QlibBacktestConfig,
    QlibDatasetConfig,
    QlibModelConfig,
    QlibRecordConfig,
    QlibStrategyConfig,
    SupportLevel,
    UnsupportedQlibComponent,
)

KNOWN_MODELS = {"LGBModel", "XGBModel", "DNNModel", "LinearModel"}
KNOWN_HANDLERS = {"Alpha158", "Alpha360"}
KNOWN_STRATEGIES = {"TopkDropoutStrategy"}
KNOWN_RECORDS = {"SignalRecord", "PortAnaRecord"}
KNOWN_PROCESSORS = {"DropnaLabel", "CSRankNorm", "RobustZScoreNorm", "Fillna"}


def extract_model(config: dict[str, Any]) -> QlibModelConfig | None:
    model = _section(config, "model")
    if model is None:
        return None
    class_name = _string_or_none(model.get("class"))
    return QlibModelConfig(
        class_name=class_name,
        module_path=_string_or_none(model.get("module_path")),
        kwargs=_mapping(model.get("kwargs")),
        support=_support(class_name, KNOWN_MODELS),
    )


def extract_dataset(config: dict[str, Any]) -> QlibDatasetConfig | None:
    dataset = _section(config, "dataset")
    if dataset is None:
        return None
    dataset_kwargs = _mapping(dataset.get("kwargs"))
    handler = _mapping(dataset_kwargs.get("handler"))
    handler_class = _string_or_none(handler.get("class"))
    return QlibDatasetConfig(
        class_name=_string_or_none(dataset.get("class")),
        module_path=_string_or_none(dataset.get("module_path")),
        handler_class=handler_class,
        handler_module_path=_string_or_none(handler.get("module_path")),
        handler_kwargs=_mapping(handler.get("kwargs")),
        segments=_mapping(dataset_kwargs.get("segments")),
        support=_support(handler_class, KNOWN_HANDLERS),
    )


def extract_records(config: dict[str, Any]) -> list[QlibRecordConfig]:
    task = _mapping(config.get("task"))
    raw_records = config.get("record", task.get("record", []))
    if isinstance(raw_records, dict):
        raw_records = [raw_records]
    if not isinstance(raw_records, list):
        return []

    records: list[QlibRecordConfig] = []
    for record_raw in raw_records:
        if not isinstance(record_raw, dict):
            continue
        record = cast(dict[str, Any], record_raw)
        class_name = _string_or_none(record.get("class"))
        records.append(
            QlibRecordConfig(
                class_name=class_name,
                module_path=_string_or_none(record.get("module_path")),
                kwargs=_mapping(record.get("kwargs")),
                support=_support(class_name, KNOWN_RECORDS),
            )
        )
    return records


def extract_strategy_and_backtest(
    records: list[QlibRecordConfig],
) -> tuple[QlibStrategyConfig | None, QlibBacktestConfig | None]:
    for record in records:
        config = _mapping(record.kwargs.get("config"))
        strategy_raw = _mapping(config.get("strategy"))
        backtest_raw = _mapping(config.get("backtest"))
        strategy = _strategy_from_mapping(strategy_raw) if strategy_raw else None
        backtest = _backtest_from_mapping(backtest_raw) if backtest_raw else None
        if strategy is not None or backtest is not None:
            return strategy, backtest
    return None, None


def collect_unsupported(
    *,
    model: QlibModelConfig | None,
    dataset: QlibDatasetConfig | None,
    strategy: QlibStrategyConfig | None,
    records: list[QlibRecordConfig],
) -> list[UnsupportedQlibComponent]:
    unsupported: list[UnsupportedQlibComponent] = []
    if model is not None and model.support == "custom_required":
        unsupported.append(
            UnsupportedQlibComponent(
                name=model.class_name or "unknown_model",
                kind="custom_model",
                reason="Custom Qlib model is recorded as metadata and requires external runtime to execute.",
            )
        )
    if dataset is not None and dataset.support == "custom_required":
        unsupported.append(
            UnsupportedQlibComponent(
                name=dataset.handler_class or "unknown_handler",
                kind="custom_data_handler",
                reason="Custom Qlib DataHandler is not interpreted by the adapter.",
            )
        )
    if dataset is not None:
        unsupported.extend(_custom_processors(dataset.handler_kwargs))
    if strategy is not None and strategy.support == "custom_required":
        unsupported.append(
            UnsupportedQlibComponent(
                name=strategy.class_name or "unknown_strategy",
                kind="custom_strategy",
                reason="Custom Qlib strategy requires external runtime route.",
            )
        )
    for record in records:
        if record.support == "custom_required":
            unsupported.append(
                UnsupportedQlibComponent(
                    name=record.class_name or "unknown_record",
                    kind="custom_record",
                    reason="Custom Qlib RecordTemplate is not interpreted by the adapter.",
                )
            )
    return unsupported


def _section(config: dict[str, Any], key: str) -> dict[str, Any] | None:
    task = _mapping(config.get("task"))
    raw = config.get(key, task.get(key))
    return _mapping(raw) if isinstance(raw, dict) else None


def _strategy_from_mapping(raw: dict[str, Any]) -> QlibStrategyConfig:
    class_name = _string_or_none(raw.get("class"))
    return QlibStrategyConfig(
        class_name=class_name,
        module_path=_string_or_none(raw.get("module_path")),
        kwargs=_mapping(raw.get("kwargs")),
        support=_support(class_name, KNOWN_STRATEGIES),
    )


def _backtest_from_mapping(raw: dict[str, Any]) -> QlibBacktestConfig:
    return QlibBacktestConfig(
        account=_number(raw.get("account")),
        benchmark=_string_or_none(raw.get("benchmark")),
        deal_price=_string_or_none(raw.get("deal_price")),
        open_cost=_float_or_none(raw.get("open_cost")),
        close_cost=_float_or_none(raw.get("close_cost")),
        min_cost=_float_or_none(raw.get("min_cost")),
        limit_threshold=_float_or_none(raw.get("limit_threshold")),
        raw=raw,
    )


def _custom_processors(handler_kwargs: dict[str, Any]) -> list[UnsupportedQlibComponent]:
    unsupported: list[UnsupportedQlibComponent] = []
    for key in ("processors", "learn_processors", "infer_processors"):
        raw_processors = handler_kwargs.get(key)
        if not isinstance(raw_processors, list):
            continue
        for processor_raw in raw_processors:
            processor = _mapping(processor_raw)
            class_name = _string_or_none(processor.get("class"))
            if class_name is not None and class_name not in KNOWN_PROCESSORS:
                unsupported.append(
                    UnsupportedQlibComponent(
                        name=class_name,
                        kind="custom_processor",
                        reason=f"Custom Qlib processor in {key} is recorded as opaque metadata.",
                    )
                )
    return unsupported


def _support(class_name: str | None, known: set[str]) -> SupportLevel:
    if class_name is None:
        return "generic_record"
    return "known" if class_name in known else "custom_required"


def _mapping(value: Any) -> dict[str, Any]:
    if isinstance(value, dict):
        return cast(dict[str, Any], value)
    return {}


def _string_or_none(value: Any) -> str | None:
    return value if isinstance(value, str) else None


def _number(value: Any) -> float | int | None:
    return value if isinstance(value, int | float) and not isinstance(value, bool) else None


def _float_or_none(value: Any) -> float | None:
    return float(value) if isinstance(value, int | float) and not isinstance(value, bool) else None

