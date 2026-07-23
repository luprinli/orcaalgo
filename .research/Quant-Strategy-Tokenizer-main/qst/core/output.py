"""Token executor output protocol."""

from __future__ import annotations

from typing import Any, Literal

import numpy as np
import pandas as pd
from pydantic import BaseModel, ConfigDict

TokenStatus = Literal["ok", "unknown", "error"]


class TokenOutput(BaseModel):
    """Uniform return value for all token executors.

    `status="unknown"` is a token execution state. It is intentionally
    separate from `Decision(kind="unknown")`, which is a domain value.
    """

    model_config = ConfigDict(arbitrary_types_allowed=True)

    values: dict[str, Any]
    status: TokenStatus = "ok"
    unknown_reason: str | None = None
    error_kind: str | None = None
    warnings: list[str] = []


def normalize_token_output(value: Any) -> TokenOutput:
    """Coerce executor returns to TokenOutput."""

    if isinstance(value, TokenOutput):
        return value
    if isinstance(value, dict):
        return TokenOutput(values=value)
    return TokenOutput(values={"value": value})


def jsonable_value(value: Any) -> Any:
    """Return a JSON-friendly representation for traces and CLI output."""

    if isinstance(value, pd.Series):
        return [jsonable_value(v) for v in value.tolist()]
    if isinstance(value, pd.DataFrame):
        return value.to_dict(orient="records")
    if isinstance(value, BaseModel):
        return value.model_dump(mode="json", exclude_none=True)
    if isinstance(value, np.generic):
        return value.item()
    if isinstance(value, float) and np.isnan(value):
        return None
    if isinstance(value, dict):
        return {str(k): jsonable_value(v) for k, v in value.items()}
    if isinstance(value, list):
        return [jsonable_value(v) for v in value]
    return value


def summarize_value(value: Any) -> dict[str, Any]:
    """Summarize a token output value without dumping large data into traces."""

    if isinstance(value, pd.Series):
        non_na = value.dropna()
        last = None if non_na.empty else jsonable_value(non_na.iloc[-1])
        return {"kind": "series", "length": len(value), "last": last}
    if isinstance(value, pd.DataFrame):
        return {"kind": "frame", "rows": len(value), "columns": list(value.columns)}
    if isinstance(value, BaseModel):
        return {"kind": value.__class__.__name__, "value": value.model_dump(mode="json", exclude_none=True)}
    return {"kind": type(value).__name__, "value": jsonable_value(value)}
