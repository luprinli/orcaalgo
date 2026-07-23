from __future__ import annotations

from qst.artifacts import ExecutionReport, compute_artifact_id

HASH_A = "sha256:" + "a" * 64
HASH_B = "sha256:" + "b" * 64


def _execution_report_dict() -> dict[str, object]:
    report = ExecutionReport(
        event_type="new",
        state="pending_new",
        qty_intended="1",
        qty_last="0",
        qty_filled="0",
        qty_remaining="1",
        raw_payload_hash=HASH_A,
        raw_payload_ref="raw/fix_001.fix",
        source_protocol="fix",
        source_system="mock_exchange",
        venue="mock",
    )
    return report.model_dump(mode="json")


def test_artifact_id_is_deterministic() -> None:
    payload = _execution_report_dict()

    assert compute_artifact_id(payload) == compute_artifact_id(dict(reversed(payload.items())))


def test_artifact_id_excludes_artifact_id_metadata_and_execution_raw_payload_ref() -> None:
    payload = _execution_report_dict()
    original = compute_artifact_id(payload)

    changed = dict(payload)
    changed["artifact_id"] = HASH_B
    changed["metadata"] = {"free": "text"}
    changed["raw_payload_ref"] = "raw/moved.fix"

    assert compute_artifact_id(changed) == original


def test_artifact_id_includes_raw_payload_hash() -> None:
    payload = _execution_report_dict()
    original = compute_artifact_id(payload)

    changed = dict(payload)
    changed["raw_payload_hash"] = HASH_B

    assert compute_artifact_id(changed) != original
