from __future__ import annotations

from qst.artifacts import PortfolioSnapshot, Position
from tests.artifacts.schema_helpers import validate_schema


def test_portfolio_snapshot_schema_validates_model_dump() -> None:
    snapshot = PortfolioSnapshot(
        timestamp="2026-05-14T18:00:00Z",
        base_currency="USD",
        cash="1000",
        equity="1250.25",
        positions=[
            Position(
                symbol="BTC/USD",
                quantity="0.25",
                market_value="250.25",
                average_price="1001",
            )
        ],
    )

    validate_schema("portfolio_snapshot-0.4.schema.json", snapshot.model_dump(mode="json"))
