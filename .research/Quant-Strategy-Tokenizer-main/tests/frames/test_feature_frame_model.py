from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.frames import FeatureFrame, FeatureRow


def test_feature_frame_sorts_feature_keys_rows_and_symbols() -> None:
    frame = FeatureFrame(
        rows=[
            FeatureRow(
                timestamp="2026-05-14T10:00:00Z",
                symbol="ETH/USD",
                features={"zscore": "1.5", "ema": "10"},
            ),
            FeatureRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", features={"ema": "9"}),
        ]
    )

    assert frame.symbols == ["BTC/USD", "ETH/USD"]
    assert list(frame.rows[1].features) == ["ema", "zscore"]


def test_feature_frame_rejects_non_canonical_decimal_feature() -> None:
    with pytest.raises(ValidationError):
        FeatureRow(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", features={"ema": "1.0"})
