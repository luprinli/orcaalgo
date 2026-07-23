from __future__ import annotations

import json
from pathlib import Path

from typer.testing import CliRunner

from qst.cli import app
from qst.hash import compute_hashes_v2
from qst.ir import load_ir_v04_file, validate_ir_v04

ROOT = Path(__file__).resolve().parents[2]
DEMO_ROOT = ROOT / "examples" / "strategies"
REFERENCE_ROOT = ROOT / "tests" / "reference" / "strategies"
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


def test_stage3a_public_demos_validate_and_have_hash_sentinels() -> None:
    demo_paths = sorted(DEMO_ROOT.glob("*/strategy.gkr.yaml"))

    assert [path.parent.name for path in demo_paths] == DEMO_CASES

    for path in demo_paths:
        case = path.parent.name
        ir = load_ir_v04_file(path)
        validation = validate_ir_v04(ir)
        expected_diagnostics = json.loads(
            (REFERENCE_ROOT / case / "diagnostics.json").read_text(encoding="utf-8")
        )
        expected_hashes = json.loads(
            (REFERENCE_ROOT / case / "hashes.json").read_text(encoding="utf-8")
        )

        assert [diagnostic.model_dump(mode="json") for diagnostic in validation.diagnostics] == (
            expected_diagnostics["diagnostics"]
        )
        assert validation.ok
        assert compute_hashes_v2(ir).__dict__ == expected_hashes
        readme = path.parent / "README.md"
        assert readme.is_file()
        assert readme.read_text(encoding="utf-8").strip()


def test_stage3a_public_demos_cli_validate_and_hash_match_reference() -> None:
    runner = CliRunner()

    for case in DEMO_CASES:
        strategy = DEMO_ROOT / case / "strategy.gkr.yaml"
        expected_hashes = json.loads(
            (REFERENCE_ROOT / case / "hashes.json").read_text(encoding="utf-8")
        )

        validate_result = runner.invoke(app, ["validate", str(strategy)])
        assert validate_result.exit_code == 0, validate_result.output

        hash_result = runner.invoke(app, ["hash", str(strategy)])
        assert hash_result.exit_code == 0, hash_result.output
        assert json.loads(hash_result.output) == expected_hashes


def test_stage3a_demo_trace_artifact_minimum_set() -> None:
    traced_cases = {
        path.parent.name
        for path in REFERENCE_ROOT.glob("*/trace.json")
    }

    assert traced_cases == TRACE_CASES
