from orca.scoring.param_score import (
    score_backtest_parameters,
)


def _row(**overrides):
    base = {
        "parameters": {"entry_z": 1.25, "exit_z": 0.5},
        "sharpe_ratio": 1.0,
        "calmar_ratio": 1.0,
        "total_return": 0.1,
        "trades": 100,
        "max_drawdown_ratio": 0.1,
    }
    base.update(overrides)
    return base


def test_eligibility_filters_low_trade_rows():
    rows = [_row(trades=5), _row(trades=100)]
    out = score_backtest_parameters(rows)
    assert len(out) == 1
    assert out[0]["trades"] == 100


def test_empty_input():
    assert score_backtest_parameters([]) == []


def test_better_sharpe_scores_higher():
    rows = [_row(sharpe_ratio=0.5), _row(sharpe_ratio=2.0)]
    out = score_backtest_parameters(rows)
    assert out[0]["sharpe_ratio"] == 2.0


def test_stability_prefers_plateau_over_isolated_spike():
    # A dense 1-D grid. The isolated high-sharpe spike (p=0.0) is surrounded by
    # low-quality neighbours (p=0.1, 0.2), while the plateau (p=1.0..1.25) has a
    # lower Sharpe but sits in a smooth region of near-identical high quality.
    spike = _row(parameters={"p": 0.0}, sharpe_ratio=3.0, calmar_ratio=3.0, total_return=0.5)
    spike_neighbors = [
        _row(parameters={"p": 0.1}, sharpe_ratio=0.3, calmar_ratio=0.3, total_return=0.03),
        _row(parameters={"p": 0.2}, sharpe_ratio=0.4, calmar_ratio=0.4, total_return=0.04),
    ]
    plateau = [
        _row(parameters={"p": 1.0}, sharpe_ratio=1.5, calmar_ratio=1.5, total_return=0.30),
        _row(parameters={"p": 1.05}, sharpe_ratio=1.4, calmar_ratio=1.4, total_return=0.28),
        _row(parameters={"p": 1.10}, sharpe_ratio=1.45, calmar_ratio=1.45, total_return=0.29),
        _row(parameters={"p": 1.15}, sharpe_ratio=1.42, calmar_ratio=1.42, total_return=0.285),
        _row(parameters={"p": 1.20}, sharpe_ratio=1.38, calmar_ratio=1.38, total_return=0.275),
        _row(parameters={"p": 1.25}, sharpe_ratio=1.35, calmar_ratio=1.35, total_return=0.27),
    ]
    out = score_backtest_parameters([spike, *spike_neighbors, *plateau])
    top = out[0]
    # The plateau wins despite the spike's higher raw Sharpe: the winner must be
    # inside the plateau region (p >= 1.0), never the isolated spike (p == 0.0).
    assert top["parameters"]["p"] >= 1.0
    assert top["parameters"]["p"] != 0.0
    # The spike, despite the best raw Sharpe, must not be ranked first.
    assert all(r["parameters"]["p"] != 0.0 for r in out[:2])


def test_balance_penalty_punishes_training_overfit():
    good = _row(balance_training_cagr=0.2, balance_validation_cagr=0.18)
    overfit = _row(balance_training_cagr=0.5, balance_validation_cagr=0.05)
    out = score_backtest_parameters([good, overfit])
    by_final = {r["balance_validation_cagr"]: r["balance_penalty"] for r in out}
    assert by_final[0.18] > by_final[0.05]


def test_verify_metrics_strengthen_score():
    # Small pool (below the stability guard) so only the core decides. Two rows
    # tie on training metrics; the one with excellent verification metrics must
    # outrank the unverified row.
    unverified = _row(sharpe_ratio=2.0, calmar_ratio=2.0, total_return=0.3)
    verified_best = _row(
        sharpe_ratio=2.0,
        calmar_ratio=2.0,
        total_return=0.3,
        verify_sharpe_ratio=3.0,
        verify_calmar_ratio=3.0,
        verify_cagr=0.5,
        verify_max_drawdown_ratio=0.02,
    )
    weaker = _row(
        sharpe_ratio=1.0,
        calmar_ratio=1.0,
        total_return=0.1,
        verify_sharpe_ratio=1.0,
        verify_calmar_ratio=1.0,
        verify_cagr=0.1,
        verify_max_drawdown_ratio=0.05,
    )
    out = score_backtest_parameters([unverified, verified_best, weaker])
    assert out[0].get("verify_sharpe_ratio") == 3.0
    assert out[0]["core_score"] > out[1]["core_score"]


def test_scores_are_finite_and_sorted():
    rows = [
        _row(sharpe_ratio=s, max_drawdown_ratio=d)
        for s, d in [(0.4, 0.3), (1.0, 0.1), (2.0, 0.05), (0.8, 0.2)]
    ]
    out = score_backtest_parameters(rows)
    finals = [r["final_score"] for r in out]
    assert all(f >= 0 for f in finals)
    assert finals == sorted(finals, reverse=True)
