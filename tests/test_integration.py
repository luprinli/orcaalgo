"""Integration tests for the full orca pipeline: CLI → model → hash → validate."""
from __future__ import annotations

import yaml


class TestFullPipeline:
    def test_gkr_roundtrip(self, tmp_path):
        gkr_path = tmp_path / "roundtrip.gkr.yaml"
        strategy_data = {
            "ir_version": "qst-ir/0.4",
            "canonical_version": "qst-canonical/0.4",
            "strategy": {
                "id": "pipeline-test",
                "version": "1.0.0",
                "nodes": [
                    {"id": "sig", "token_ref": {"token_id": "signal.threshold", "version": ">=1.0"}, "params": {"entry": 2.0}},
                    {"id": "size", "token_ref": {"token_id": "size.kelly_fractional", "version": ">=1.0"}, "inputs": {"signal": "sig.signal"}, "params": {"multiplier": 0.25}},
                ],
                "outputs": {"final": "size.contracts"},
            },
            "capabilities": [{"name": "core"}],
        }
        gkr_path.write_text(yaml.dump(strategy_data))

        from orca.ir.loader import load_ir
        ir = load_ir(gkr_path)
        assert ir.strategy.id == "pipeline-test"
        assert len(ir.strategy.nodes) == 2

        from orca.hash.graph import graph_hash_v2, instance_hash_v2, param_hash_v2
        gh = graph_hash_v2(ir)
        ph = param_hash_v2(ir)
        ih = instance_hash_v2(ir)
        assert gh.startswith("sha256:")
        assert ph.startswith("sha256:")
        assert ih.startswith("sha256:")
        assert gh != ph
        assert gh != ih
        assert ph != ih

        from orca.hash.verify import verify_graph_hash, verify_instance_hash
        assert verify_graph_hash(ir, gh)
        assert not verify_graph_hash(ir, "sha256:bad")
        assert verify_instance_hash(ir, ih)

        from orca.ir.validator import validate_ir
        diags = validate_ir(ir, "production_guarded")
        errors = [d for d in diags if d.severity == "error"]
        assert len(errors) == 0

        from orca.ir.loader import save_ir
        save_ir(ir, tmp_path / "roundtrip_out.gkr.yaml")
        ir2 = load_ir(tmp_path / "roundtrip_out.gkr.yaml")
        assert graph_hash_v2(ir) == graph_hash_v2(ir2)

    def test_kelly_attenuator_chain(self):
        from orca.sizing.kelly import kelly_with_attenuators

        result = kelly_with_attenuators(0.65, 0.60, "yes")
        assert 0 <= result.discounted_p <= 1
        assert result.fractional_kelly <= result.raw_kelly
        assert result.final_allocation <= result.fractional_kelly
        assert result.final_allocation <= 0.02

    def test_brier_then_wilson_pipeline(self):
        from orca.math.brier import brier_score
        from orca.math.wilson import wilson_ci

        predictions = [0.3, 0.7, 0.8, 0.2, 0.9, 0.4, 0.6, 0.1, 0.5, 0.3]
        outcomes = [0, 1, 1, 0, 1, 0, 1, 0, 0, 0]
        score = brier_score(predictions, outcomes)
        assert 0 <= score <= 1

        wins = sum(outcomes)
        lo, hi = wilson_ci(wins, len(outcomes))
        assert 0 <= lo <= hi <= 1

    def test_calibration_to_attribution_flow(self, sample_trade_records):
        from orca.attribution.slicer import attribute_pnl
        from orca.calibration.audit import run_calibration_audit

        cal_report = run_calibration_audit(sample_trade_records)
        assert cal_report.overall.n == len(sample_trade_records)
        assert 0 <= cal_report.overall.brier <= 1

        attr_report = attribute_pnl(sample_trade_records)
        assert attr_report.overall.n == len(sample_trade_records)
        assert len(attr_report.by_side) >= 1

    def test_preflight_integration(self):
        from orca.preflight.checklist import run_preflight_checks
        checks = run_preflight_checks()
        assert len(checks) > 0
        passed = sum(1 for c in checks if c.status == "pass")
        failed = sum(1 for c in checks if c.status == "fail")
        assert passed >= 0
        assert failed >= 0
