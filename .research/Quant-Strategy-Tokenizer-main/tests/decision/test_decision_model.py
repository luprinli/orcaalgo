from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.decision import DecisionV2


def test_decision_canonicalizes_reasons_and_score() -> None:
    decision = DecisionV2(kind="accept", reasons=["z", "a", "a"], score="0.25")

    assert decision.model_dump(mode="json", exclude_none=True) == {
        "schema_version": "qst-decision/0.4",
        "kind": "accept",
        "reasons": ["a", "z"],
        "score": "0.25",
    }


def test_decision_kind_rejects_error() -> None:
    with pytest.raises(ValidationError):
        DecisionV2.model_validate({"kind": "error", "reasons": ["boom"]})


@pytest.mark.parametrize("bad_score", ["1.0", "0.10", "-0", "1e-3", "+1"])
def test_score_must_be_canonical_decimal_string(bad_score: str) -> None:
    with pytest.raises(ValidationError):
        DecisionV2(kind="accept", score=bad_score)
