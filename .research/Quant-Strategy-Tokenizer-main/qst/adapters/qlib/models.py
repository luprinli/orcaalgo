"""Models for the Qlib partial workflow adapter."""

from __future__ import annotations

from pathlib import Path
from typing import Any, Literal

from pydantic import BaseModel, Field

CoverageClass = Literal[
    "supported",
    "partially_supported",
    "custom_token_required",
    "reserved",
    "non_goal",
]
SupportLevel = Literal["known", "generic_record", "custom_required"]


class QlibModelConfig(BaseModel):
    class_name: str | None = None
    module_path: str | None = None
    kwargs: dict[str, Any] = Field(default_factory=dict)
    support: SupportLevel = "generic_record"


class QlibDatasetConfig(BaseModel):
    class_name: str | None = None
    module_path: str | None = None
    handler_class: str | None = None
    handler_module_path: str | None = None
    handler_kwargs: dict[str, Any] = Field(default_factory=dict)
    segments: dict[str, Any] = Field(default_factory=dict)
    support: SupportLevel = "generic_record"


class QlibStrategyConfig(BaseModel):
    class_name: str | None = None
    module_path: str | None = None
    kwargs: dict[str, Any] = Field(default_factory=dict)
    support: SupportLevel = "generic_record"


class QlibBacktestConfig(BaseModel):
    account: float | int | None = None
    benchmark: str | None = None
    deal_price: str | None = None
    open_cost: float | None = None
    close_cost: float | None = None
    min_cost: float | None = None
    limit_threshold: float | None = None
    raw: dict[str, Any] = Field(default_factory=dict)


class QlibRecordConfig(BaseModel):
    class_name: str | None = None
    module_path: str | None = None
    kwargs: dict[str, Any] = Field(default_factory=dict)
    support: SupportLevel = "generic_record"


class UnsupportedQlibComponent(BaseModel):
    name: str
    kind: Literal[
        "custom_model",
        "custom_data_handler",
        "custom_processor",
        "custom_strategy",
        "custom_record",
        "runtime_execution",
        "unknown_config",
        "non_goal",
    ]
    reason: str


class QlibImportCoverage(BaseModel):
    adapter: Literal["qlib"] = "qlib"
    source: str
    classification: CoverageClass
    tokenized_components: list[str] = Field(default_factory=list)
    model: QlibModelConfig | None = None
    dataset: QlibDatasetConfig | None = None
    strategy: QlibStrategyConfig | None = None
    backtest: QlibBacktestConfig | None = None
    records: list[QlibRecordConfig] = Field(default_factory=list)
    unsupported_components: list[UnsupportedQlibComponent] = Field(default_factory=list)
    warnings: list[str] = Field(default_factory=list)


class QlibImportResult(BaseModel):
    ok: bool
    source: str
    strategy_path: Path | None = None
    coverage_path: Path | None = None
    coverage: QlibImportCoverage
    warnings: list[str] = Field(default_factory=list)

