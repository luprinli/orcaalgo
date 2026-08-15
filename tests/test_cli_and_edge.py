from __future__ import annotations

import pytest
import yaml

from orca.hash.common import hash_v2, stable_json_bytes
from orca.models.strategy import StrategyIRV04


class TestTemporalValidation:
    def test_clean_strategy_passes(self, sample_strategy_ir):
        from orca.ports.temporal import trace_temporal_validation

        diags = trace_temporal_validation(sample_strategy_ir, "research")
        assert all(d.severity != "error" for d in diags)

    def test_missing_signature_no_errors(self):
        ir = StrategyIRV04.model_validate(
            {
                "ir_version": "qst-ir/0.4",
                "canonical_version": "qst-canonical/0.4",
                "strategy": {
                    "id": "test",
                    "version": "1.0.0",
                    "nodes": [{"id": "n1", "token_ref": {"token_id": "t1", "version": ">=1.0"}}],
                    "outputs": {},
                },
            }
        )
        from orca.ports.temporal import trace_temporal_validation

        diags = trace_temporal_validation(ir, "research")
        assert all(d.severity != "error" for d in diags)


class TestCLIValidate:
    def test_validate_valid_file(self, tmp_path):
        gkr = tmp_path / "test.gkr.yaml"
        gkr.write_text(
            yaml.dump(
                {
                    "ir_version": "qst-ir/0.4",
                    "canonical_version": "qst-canonical/0.4",
                    "strategy": {
                        "id": "test",
                        "version": "1.0.0",
                        "nodes": [
                            {"id": "n1", "token_ref": {"token_id": "t1", "version": ">=1.0"}}
                        ],
                        "outputs": {},
                    },
                    "capabilities": [{"name": "core"}],
                }
            )
        )

        from typer.testing import CliRunner

        from orca.cli import app

        runner = CliRunner()
        result = runner.invoke(app, ["validate", str(gkr)])
        assert result.exit_code == 0
        assert "graph_hash" in result.output
        assert "param_hash" in result.output
        assert "instance_hash" in result.output

    def test_validate_missing_capability_fails(self, tmp_path):
        gkr = tmp_path / "bad.gkr.yaml"
        gkr.write_text(
            yaml.dump(
                {
                    "ir_version": "qst-ir/0.4",
                    "canonical_version": "qst-canonical/0.4",
                    "strategy": {
                        "id": "test",
                        "version": "1.0.0",
                        "nodes": [
                            {"id": "n1", "token_ref": {"token_id": "t1", "version": ">=1.0"}}
                        ],
                        "outputs": {},
                    },
                }
            )
        )

        from typer.testing import CliRunner

        from orca.cli import app

        runner = CliRunner()
        result = runner.invoke(app, ["validate", str(gkr)])
        assert result.exit_code == 1

    def test_validate_file_not_found(self):
        from typer.testing import CliRunner

        from orca.cli import app

        runner = CliRunner()
        result = runner.invoke(app, ["validate", "nonexistent.gkr.yaml"])
        assert result.exit_code != 0


class TestCLIPreflight:
    def test_preflight_runs(self):
        from typer.testing import CliRunner

        from orca.cli import app

        runner = CliRunner()
        result = runner.invoke(app, ["preflight"])
        assert "Pre-Flight Results" in result.output
        assert result.exit_code == 0


class TestHashEdgeCases:
    def test_nested_lists_are_stable(self):
        data = {"a": [3, 1, 2], "b": [{"z": 1, "a": 2}]}
        b1 = stable_json_bytes(data)
        b2 = stable_json_bytes(data)
        assert b1 == b2

    def test_empty_containers(self):
        assert stable_json_bytes({}) == b"{}"
        assert stable_json_bytes([]) == b"[]"
        assert stable_json_bytes({"a": []}) == b'{"a":[]}'

    def test_string_values(self):
        h = hash_v2("test", {"key": "value with spaces & symbols!"})
        assert h.startswith("sha256:")
        assert len(h) == 71

    def test_hash_namespace_isolation(self):
        h1 = hash_v2("graph", {"x": 1})
        assert "sha256:" in h1
        assert len(h1) == 71


class TestKellyEdgeCases:
    def test_p_zero_yes_side(self):
        from orca.sizing.kelly import kelly_fraction_binary

        f = kelly_fraction_binary(0.0, 0.5, "yes")
        assert f == -1.0

    def test_p_one_no_side(self):
        from orca.sizing.kelly import kelly_fraction_binary

        f = kelly_fraction_binary(1.0, 0.5, "no")
        assert f == -1.0  # sure win for YES => certain loss for NO

    def test_price_edge_cases(self):
        from orca.sizing.kelly import kelly_fraction_binary

        with pytest.raises(ValueError):
            kelly_fraction_binary(0.6, 1.0, "yes")
        with pytest.raises(ValueError):
            kelly_fraction_binary(0.6, 0.0, "yes")

    def test_attenuators_never_produce_nan(self):
        import math

        from orca.sizing.kelly import kelly_with_attenuators

        result = kelly_with_attenuators(0.99, 0.01, "yes")
        assert not math.isnan(result.final_allocation)
        assert result.final_allocation >= 0


class TestVolatilityEdgeCases:
    def test_all_zeros_vol_is_zero(self):
        import numpy as np

        from orca.sizing.volatility import ewma_volatility

        returns = np.zeros(100)
        vol = ewma_volatility(returns)
        assert vol == 0.0

    def test_constant_positive_return(self):
        import numpy as np

        from orca.sizing.volatility import ewma_volatility

        returns = np.full(50, 0.01)
        vol = ewma_volatility(returns)
        assert vol < 0.01

    def test_diversification_one_position(self):
        from orca.sizing.volatility import diversification_scaling

        assert diversification_scaling(1, 0.0) == 1.0
        assert diversification_scaling(1, 1.0) == 1.0
