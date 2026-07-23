from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.artifacts import ProvenanceChain


def test_provenance_chain_parent_artifacts_are_sha256() -> None:
    parent = "sha256:" + "a" * 64

    chain = ProvenanceChain(parent_artifacts=[parent], operation="toy")

    assert chain.parent_artifacts == [parent]


def test_provenance_chain_rejects_invalid_parent_artifact_hash() -> None:
    with pytest.raises(ValidationError):
        ProvenanceChain(parent_artifacts=["not-a-hash"])
