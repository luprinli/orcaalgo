from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.artifacts import ExecutionReport

HASH = "sha256:" + "3" * 64


def test_raw_payload_hash_without_ref_is_valid() -> None:
    report = ExecutionReport(
        event_type="new",
        state="pending_new",
        qty_intended="1",
        qty_last="0",
        qty_filled="0",
        qty_remaining="1",
        raw_payload_hash=HASH,
        source_protocol="mock",
        venue="mock",
    )

    assert report.raw_payload_ref is None
    assert report.raw_payload_hash == HASH


def test_raw_payload_ref_without_hash_is_invalid() -> None:
    with pytest.raises(ValidationError, match="raw_payload_ref"):
        ExecutionReport(
            event_type="new",
            state="pending_new",
            qty_intended="1",
            qty_last="0",
            qty_filled="0",
            qty_remaining="1",
            raw_payload_ref="raw/report.json",
            source_protocol="mock",
            venue="mock",
        )
