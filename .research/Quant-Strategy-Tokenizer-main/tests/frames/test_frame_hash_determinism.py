from __future__ import annotations

from qst.frames import (
    FeatureFrame,
    FeatureRow,
    MarketFrame,
    OHLCVBar,
    compute_frame_hash,
)


def test_frame_hash_excludes_frame_hash_field() -> None:
    frame = MarketFrame(
        bars=[
            OHLCVBar(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                open="1",
                high="2",
                low="0.5",
                close="1.5",
                volume="10",
            )
        ]
    )
    baseline = compute_frame_hash(frame)

    same = frame.model_copy(update={"frame_hash": "sha256:" + "f" * 64})
    assert compute_frame_hash(same) == baseline


def test_frame_hash_is_stable_across_input_order() -> None:
    first = MarketFrame(
        symbols=["ETH/USD", "BTC/USD"],
        bars=[
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
        ],
    )
    second = MarketFrame(
        symbols=["BTC/USD", "ETH/USD"],
        bars=[
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="BTC/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T09:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
            OHLCVBar(timestamp="2026-05-14T10:00:00Z", symbol="ETH/USD", open="1", high="2", low="1", close="2", volume="3"),
        ],
    )

    assert compute_frame_hash(first) == compute_frame_hash(second)


def test_p4a1b_frame_hash_evidence_is_preserved() -> None:
    market = MarketFrame(
        bars=[
            OHLCVBar(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                open="1",
                high="2",
                low="0.5",
                close="1.5",
                volume="10",
            )
        ]
    )
    feature = FeatureFrame(
        rows=[
            FeatureRow(
                timestamp="2026-05-14T09:00:00Z",
                symbol="BTC/USD",
                features={"zscore": "1.5", "ema": "10"},
            )
        ]
    )

    assert compute_frame_hash(market) == (
        "sha256:4b2887c7d18e5df8509a07a0491d93228e5d95760e6576f97045ee1dc3bc981f"
    )
    assert compute_frame_hash(feature) == (
        "sha256:82124a257ad3524a113e1eac88866551e32f29c3c654c3f00cb4646da2d2b8d1"
    )
