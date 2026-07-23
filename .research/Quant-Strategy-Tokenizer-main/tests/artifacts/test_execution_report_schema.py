from __future__ import annotations

import pytest
from jsonschema.exceptions import ValidationError as JsonSchemaValidationError
from pydantic import ValidationError

from qst.artifacts import ExecutionReport
from tests.artifacts.schema_helpers import validate_schema, validator_for

HASH = "sha256:" + "1" * 64


def test_execution_report_schema_validates_model_dump() -> None:
    report = ExecutionReport(
        event_type="trade",
        state="partially_filled",
        qty_intended="2",
        qty_last="1",
        qty_filled="1",
        qty_remaining="1",
        last_fill_price="100",
        raw_payload_hash=HASH,
        raw_payload_ref="raw/fix_001.fix",
        source_protocol="fix",
        source_system="mock_exchange",
        venue="mock",
    )

    validate_schema("execution_report-0.4.schema.json", report.model_dump(mode="json"))


def test_non_trade_event_requires_qty_last_zero() -> None:
    with pytest.raises(ValidationError, match="qty_last"):
        ExecutionReport(
            event_type="new",
            state="pending_new",
            qty_intended="1",
            qty_last="1",
            qty_filled="0",
            qty_remaining="1",
            source_protocol="mock",
            venue="mock",
        )


def test_trade_event_requires_positive_last_qty_and_fill_price() -> None:
    with pytest.raises(ValidationError, match="last_fill_price"):
        ExecutionReport(
            event_type="trade",
            state="partially_filled",
            qty_intended="1",
            qty_last="1",
            qty_filled="1",
            qty_remaining="0",
            source_protocol="mock",
            venue="mock",
        )


def test_raw_payload_ref_requires_hash() -> None:
    with pytest.raises(ValidationError, match="raw_payload_ref"):
        ExecutionReport(
            event_type="new",
            state="pending_new",
            qty_intended="1",
            qty_last="0",
            qty_filled="0",
            qty_remaining="1",
            raw_payload_ref="raw/fix_001.fix",
            source_protocol="mock",
            venue="mock",
        )


def test_execution_report_schema_rejects_negative_zero_decimal_string() -> None:
    payload = {
        "artifact_type": "qst-execution-report/1",
        "artifact_id": None,
        "provenance": {"parent_artifacts": [], "operation": None, "notes": []},
        "metadata": {},
        "raw_payload_ref": None,
        "raw_payload_hash": None,
        "event_type": "new",
        "state": "pending_new",
        "qty_intended": "1",
        "qty_last": "-0",
        "qty_filled": "0",
        "qty_remaining": "1",
        "last_fill_price": None,
        "avg_fill_price": None,
        "source_protocol": "mock",
        "source_system": None,
        "venue": "mock",
        "adapter": None,
    }

    with pytest.raises(JsonSchemaValidationError):
        validator_for("execution_report-0.4.schema.json").validate(payload)
