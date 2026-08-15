from orca.scoring.ticker_split import (
    is_training_ticker,
    split_tickers,
)


def test_forced_validation_symbols():
    assert is_training_ticker("SPY") is False
    assert is_training_ticker("qqq") is False  # case-insensitive


def test_split_is_deterministic():
    tickers = [
        "AAPL",
        "MSFT",
        "NVDA",
        "TSLA",
        "IWM",
        "GLD",
        "TLT",
        "EURUSD",
        "GBPUSD",
        "BTC-USD",
        "^DAX",
        "SPY",
        "QQQ",
    ]
    a = split_tickers(tickers)
    b = split_tickers(tickers)
    assert a == b
    assert a[0] == b[0] and a[1] == b[1]


def test_split_partitions_all_input():
    tickers = ["AAPL", "MSFT", "NVDA", "TSLA", "IWM", "GLD", "TLT"]
    training, validation = split_tickers(tickers)
    assert sorted(training + validation) == sorted(tickers)
    assert len(training) + len(validation) == len(tickers)


def test_hash_is_stable_across_case():
    # Same symbol in different case must land in the same bucket (after the
    # forced-validation check).
    assert is_training_ticker("aapl") == is_training_ticker("AAPL")


def test_scaled_hash_in_unit_interval():
    from orca.scoring.ticker_split import _scaled_hash

    for sym in ["AAA", "ZZZ", "META", "F", "BRK.B"]:
        v = _scaled_hash(sym)
        assert 0.0 <= v < 1.0
