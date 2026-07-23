"""artifact identity calculation."""

from __future__ import annotations

import hashlib
from typing import Any

from qst.canonical_json import stable_json_bytes

ARTIFACT_ID_EXCLUDE_COMMON = frozenset({"artifact_id", "metadata"})
ARTIFACT_ID_EXCLUDE_PER_TYPE = {
    "qst-execution-report/1": frozenset({"raw_payload_ref"}),
    "qst-backtest-evidence/1": frozenset(),
    "qst-portfolio-snapshot/1": frozenset(),
}


def compute_artifact_id(artifact_dict: dict[str, Any]) -> str:
    """Compute a content-derived artifact id."""

    artifact_version = str(artifact_dict.get("artifact_version", ""))
    exclude = ARTIFACT_ID_EXCLUDE_COMMON | ARTIFACT_ID_EXCLUDE_PER_TYPE.get(
        artifact_version, frozenset()
    )
    payload = {key: value for key, value in artifact_dict.items() if key not in exclude}
    return f"sha256:{hashlib.sha256(stable_json_bytes(payload)).hexdigest()}"
