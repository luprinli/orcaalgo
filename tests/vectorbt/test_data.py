"""Tests for orca.vectorbt.data — dual-backend data connector."""

import pandas as pd

from orca.vectorbt.data import (
    _TIMEFRAME_DIR,
    REQUIRED_COLUMNS,
    _find_date_column,
    _normalize_columns,
    _resolve_stooq_path,
    load_asset_classes,
    load_candles,
)


class TestLoadCandlesFile:
    """File-based loading — the primary path since TimescaleDB may not be available."""

    def test_load_from_csv_file(self, monkeypatch, tmp_path):
        csv_path = tmp_path / "spy_1d.csv"
        df = pd.DataFrame(
            {
                "Date": ["2023-01-02", "2023-01-03", "2023-01-04"],
                "Open": [380.0, 382.0, 379.0],
                "High": [385.0, 384.0, 383.0],
                "Low": [378.0, 380.0, 377.0],
                "Close": [383.0, 381.0, 382.0],
                "Volume": [1000, 1500, 1200],
            }
        )
        df.to_csv(csv_path, index=False)
        monkeypatch.setenv("ORCA_DATA_DIR", str(tmp_path))

        result = load_candles("spy", "2023-01-01", "2023-12-31", "1d", backend="file")
        assert len(result) == 3
        for col in REQUIRED_COLUMNS:
            assert col in result.columns, f"Missing column: {col}"

    def test_load_csv_bare_path(self, monkeypatch, tmp_path):
        csv_path = tmp_path / "eurusd_1d.csv"
        df = pd.DataFrame(
            {
                "Date": pd.date_range("2023-01-02", periods=5),
                "open": [1.1, 1.2, 1.15, 1.18, 1.17],
                "high": [1.12, 1.22, 1.17, 1.20, 1.19],
                "low": [1.08, 1.18, 1.14, 1.16, 1.15],
                "close": [1.11, 1.19, 1.16, 1.17, 1.18],
                "volume": [0, 0, 0, 0, 0],
            }
        )
        df.to_csv(csv_path, index=False)
        monkeypatch.setenv("ORCA_DATA_DIR", str(tmp_path))

        result = load_candles("eurusd", "2023-01-01", "2023-12-31", "1d", backend="file")
        assert len(result) > 0

    def test_start_end_filtering(self, monkeypatch, tmp_path):
        csv_path = tmp_path / "test_1h.csv"
        dates = pd.date_range("2023-01-02", periods=100, freq="h")
        df = pd.DataFrame(
            {
                "Date": dates,
                "open": 100.0,
                "high": 101.0,
                "low": 99.0,
                "close": 100.5,
                "volume": 100,
            }
        )
        df.to_csv(csv_path, index=False)
        monkeypatch.setenv("ORCA_DATA_DIR", str(tmp_path))

        result = load_candles("test", "2023-01-03", "2023-01-04", "1h", backend="file")
        assert len(result) > 0
        assert result.index.min() >= pd.Timestamp("2023-01-03")
        assert result.index.max() <= pd.Timestamp("2023-01-04")


class TestNormalizeColumns:
    def test_column_lowercase(self):
        df = pd.DataFrame(columns=["Open", "High", "Low", "Close", "Volume"])
        _normalize_columns(df)
        assert list(df.columns) == ["open", "high", "low", "close", "volume"]

    def test_volume_alias(self):
        df = pd.DataFrame(columns=["Open", "High", "Low", "Close", "vol"])
        _normalize_columns(df)
        assert "volume" in df.columns


class TestFindDateColumn:
    def test_date_column(self):
        df = pd.DataFrame({"Date": [1], "Value": [2]})
        assert _find_date_column(df) == "Date"

    def test_timestamp_column(self):
        df = pd.DataFrame({"timestamp": [1]})
        assert _find_date_column(df) == "timestamp"

    def test_first_column_fallback(self):
        df = pd.DataFrame({"X": [1], "Y": [2]})
        assert _find_date_column(df) == "X"


class TestResolveStooqPath:
    def test_no_path_found(self, tmp_path):
        result = _resolve_stooq_path(tmp_path, "daily", "nonexistent")
        assert result is None

    def test_currencies_path(self, tmp_path):
        currencies = tmp_path / "daily" / "world" / "currencies" / "major"
        currencies.mkdir(parents=True)
        (currencies / "eurusd.txt").write_text("")
        result = _resolve_stooq_path(tmp_path, "daily", "eurusd")
        assert result is not None
        assert "eurusd" in str(result)


class TestTimeframeDir:
    def test_all_timeframes_mapped(self):
        for tf in ["1d", "4h", "1h", "15m", "5m", "1m"]:
            assert tf in _TIMEFRAME_DIR, f"Missing timeframe: {tf}"


class TestLoadAssetClasses:
    def test_returns_dict(self):
        result = load_asset_classes()
        assert isinstance(result, dict)
        assert "EURUSD" in result
        assert result["EURUSD"] == "forex"
        assert result["SPY"] == "equity"
        assert result["BTCUSD"] == "crypto"
