from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from qst.ir import NodeV04, StrategyBodyV04, StrategyIRV04
from qst.tokens import TokenRegistryV2, TokenSpecV2, builtin_token_packs

ROOT = Path(__file__).resolve().parents[2]
DEMO_ROOT = ROOT / "examples" / "strategies"
REFERENCE_ROOT = ROOT / "tests" / "reference" / "strategies"
REPORT_ROOT = ROOT / "docs" / "reports"

DEMO_CASES = [
    "01_ema_cross",
    "02_rsi_reversal",
    "03_bollinger_mean_reversion",
    "04_breakout_channel",
    "05_cooldown_trend_following",
    "06_circuit_breaker_mean_reversion",
    "07_topk_momentum_panel",
    "08_market_neutral_rank",
    "09_btc_residual_meanrev",
    "10_volatility_target_weight",
    "11_turnover_constrained_rebalance",
    "12_custom_token_kalman_signal",
]
TRACE_CASES = {
    "01_ema_cross",
    "08_market_neutral_rank",
    "12_custom_token_kalman_signal",
}
EXPECTED_PACK_IDS = [
    "qst-tokenpack-core-surface",
    "qst-tokenpack-decision-algebra",
    "qst-tokenpack-panel-ops",
    "qst-tokenpack-panel-weights",
    "qst-tokenpack-state-basic",
    "qst-tokenpack-state-fsm",
]
EXPECTED_TOKEN_COUNT = 179
EXPECTED_FAMILIES = {
    "align",
    "bool",
    "compare",
    "continuous_score",
    "data",
    "decision",
    "distribution",
    "event",
    "execution",
    "factor",
    "gate",
    "indicator",
    "math",
    "optimizer",
    "panel",
    "risk",
    "signal",
    "state",
    "time",
    "weight",
    "window",
}


def all_specs() -> list[TokenSpecV2]:
    return [spec for pack in builtin_token_packs() for spec in pack.tokens]


def spec_by_name() -> dict[str, TokenSpecV2]:
    return {spec.token_ref.name: spec for spec in all_specs()}


def registry() -> TokenRegistryV2:
    return TokenRegistryV2.from_packs(builtin_token_packs())


def strategy_for(token_name: str) -> StrategyIRV04:
    return StrategyIRV04(
        strategy=StrategyBodyV04(
            id=f"stage3b_{token_name.replace('.', '_')}",
            nodes=[
                NodeV04(
                    id="node",
                    token_ref={
                        "namespace": "core",
                        "name": token_name,
                        "version": 1,
                        "behavior_version": 1,
                    },
                )
            ],
        )
    )


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))
