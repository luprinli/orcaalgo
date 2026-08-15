"""Extract strategy parameters from GKR YAML configs for Odin execution."""

from __future__ import annotations

import json
from pathlib import Path

from orca.ir.loader import load_ir


def extract_params(gkr_dir: str | Path = "configs/strategies") -> dict:
    gkr_dir = Path(gkr_dir)
    result: dict[str, list] = {"strategies": []}

    for yaml_path in sorted(gkr_dir.glob("*.gkr.yaml")):
        ir = load_ir(yaml_path)
        strat_id = ir.strategy.id

        params: dict[str, float | int | str] = {}
        for node in ir.strategy.nodes:
            if node.token_ref.token_id == "data.bar_aggregator":
                params["period_seconds"] = node.params.get("period_seconds", 60)
            elif node.token_ref.token_id == "math.zscore":
                params["lookback"] = node.params.get("lookback", 20)
            elif node.token_ref.token_id == "signal.threshold":
                params["entry_z"] = node.params.get("entry_threshold", 2.0)
                params["exit_z"] = node.params.get("exit_threshold", 0.5)
            elif node.token_ref.token_id == "size.kelly_fractional":
                params["kelly_multiplier"] = node.params.get("multiplier", 0.25)
                params["per_trade_cap_pct"] = node.params.get("per_trade_cap_pct", 0.02)
                params["total_exposure_cap_pct"] = node.params.get("total_exposure_cap_pct", 0.30)
            elif node.token_ref.token_id == "risk.ftmo_compliance":
                params["max_daily_dd_pct"] = node.params.get("max_daily_dd_pct", 5.0)
                params["max_absolute_dd"] = node.params.get("max_absolute_dd", 1000.0)
                params["max_open_positions"] = node.params.get("max_open_positions", 5)
            elif node.token_ref.token_id == "indicator.ma_crossover":
                params["fast_period"] = node.params.get("fast_period", 20)
                params["slow_period"] = node.params.get("slow_period", 50)
            elif node.token_ref.token_id == "indicator.atr_filter":
                params["atr_period"] = node.params.get("atr_period", 14)
                params["atr_multiplier"] = node.params.get("atr_multiplier", 1.5)
            elif node.token_ref.token_id == "signal.opening_range":
                params["range_minutes"] = node.params.get("range_minutes", 5)
                params["breakout_pct"] = node.params.get("breakout_pct", 0.5)

        # Per-strategy regime participation (regime_gating_deep_dive.md §2). The
        # risk_profile.regime_multipliers tuple is (Calm, Trending, HighVol, Crisis)
        # and maps 1:1 onto the engine's regime_w_* participation weights.
        if ir.risk_profile is not None:
            rp = ir.risk_profile
            params["risk_per_trade_pct"] = rp.risk_per_trade_pct
            params["kelly_multiplier"] = rp.kelly_multiplier
            names = ["regime_w_calm", "regime_w_trending", "regime_w_highvol", "regime_w_crisis"]
            for name, mult in zip(names, rp.regime_multipliers, strict=False):
                params[name] = mult

        result["strategies"].append(
            {
                "id": strat_id,
                "kind": _infer_kind(strat_id),
                "params": params,
            }
        )

    return result


def _infer_kind(strat_id: str) -> str:
    if "intraday-mr" in strat_id.lower() or "mean_reversion" in strat_id.lower():
        return "mean_reversion"
    if "trend" in strat_id.lower():
        return "trend_following"
    if "breakout" in strat_id.lower() or "opening_range" in strat_id.lower():
        return "opening_range_breakout"
    return "unknown"


def export_params_json(
    gkr_dir: str | Path = "configs/strategies",
    output: str | Path = "configs/strategy_params.json",
) -> dict:
    params = extract_params(gkr_dir)
    output = Path(output)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(params, indent=2))
    return params


if __name__ == "__main__":
    result = export_params_json()
    for s in result["strategies"]:
        print(s["id"], s["kind"], s["params"])
