from orca.scoring.template_score import (
    compute_template_scores,
)


def _period(template, months=60, age=100, v_cagr=0.2, t_cagr=0.25, dd=10.0, trades=300):
    return {
        "template": template,
        "period_months": months,
        "age_days": age,
        "training_cagr": t_cagr,
        "validation_cagr": v_cagr,
        "validation_max_drawdown_pct": dd,
        "trades": trades,
    }


def test_ranking_between_templates():
    good = [_period("good", v_cagr=0.30, t_cagr=0.32, dd=5.0, trades=500)]
    bad = [_period("bad", v_cagr=-0.05, t_cagr=0.10, dd=30.0, trades=5)]
    out = compute_template_scores(good + bad)
    assert out[0]["template"] == "good"
    assert out[0]["final_score_100"] > out[1]["final_score_100"]


def test_verification_multiplier_band():
    periods = [_period("t")]
    bad_verify = {
        "sharpe_ratio": 0.0,
        "calmar_ratio": 0.0,
        "cagr": -0.5,
        "max_drawdown_ratio": 0.5,
    }
    good_verify = {
        "sharpe_ratio": 3.0,
        "calmar_ratio": 3.0,
        "cagr": 1.0,
        "max_drawdown_ratio": 0.0,
    }
    low = compute_template_scores(periods, {"t": bad_verify})[0]
    high = compute_template_scores(periods, {"t": good_verify})[0]
    assert high["verification_multiplier"] > low["verification_multiplier"]
    assert 0.8 <= high["verification_multiplier"] <= 1.2
    assert 0.8 <= low["verification_multiplier"] <= 1.2


def test_longer_windows_carry_more_weight():
    # Same per-period quality, but the 120-month result should dominate the base
    # score versus a 1-month result.
    short = [_period("t", months=1, age=0, v_cagr=0.5, dd=2.0)]
    long = [_period("t", months=120, age=0, v_cagr=0.1, dd=2.0)]
    short_result = compute_template_scores(short)
    long_result = compute_template_scores(long)
    assert long_result[0]["base_score"] < short_result[0]["base_score"]  # lower CAGR
    # But weight of the long window is larger.
    assert long_result[0]["components"][0]["weight"] > short_result[0]["components"][0]["weight"]


def test_empty_input():
    assert compute_template_scores([]) == []


def test_negative_validation_penalized():
    neg = [_period("neg", v_cagr=-0.2)]
    pos = [_period("pos", v_cagr=0.2)]
    out = compute_template_scores(neg + pos)
    assert out[0]["template"] == "pos"
