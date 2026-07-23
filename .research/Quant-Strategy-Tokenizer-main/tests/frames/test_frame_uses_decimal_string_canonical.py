from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.frames import OHLCVBar


@pytest.mark.parametrize("bad_value", ["1.0", "0.10", "-0", "1e-3", "+1.0", "001.0"])
def test_ohlcv_fields_use_strict_decimal_string(bad_value: str) -> None:
    with pytest.raises(ValidationError):
        OHLCVBar(
            timestamp="2026-05-14T09:00:00Z",
            symbol="BTC/USD",
            open=bad_value,
            high="2",
            low="1",
            close="2",
            volume="3",
        )
