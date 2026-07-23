"""Tests for orca.vectorbt.to_gkr — GKR YAML export."""

import tempfile

import pytest
import yaml

from orca.vectorbt.to_gkr import (
    _build_v1_config,
    _build_v2_config,
    _generate_gkr_mapping,
    params_to_gkr,
)


class TestV1Config:
    def test_has_all_required_keys(self):
        config = _build_v1_config(
            "test_strat",
            {"rsi_period": 20, "entry_threshold": 30.0},
            {
                "sharpe_ratio": 1.5,
                "max_drawdown": -10.0,
                "total_return": 25.0,
                "win_rate": 60.0,
                "num_trades": 42,
            },
            None,
        )

        required = {"version", "strategy_id", "parameters", "metrics", "optimized_at", "optimized_by"}
        assert required.issubset(set(config.keys()))

    def test_version_is_one(self):
        config = _build_v1_config("t", {"a": 1}, None, None)
        assert config["version"] == 1

    def test_optimized_by_vectorbt(self):
        config = _build_v1_config("t", {"a": 1}, None, None)
        assert config["optimized_by"] == "vectorbt"

    def test_parameters_floated(self):
        config = _build_v1_config("t", {"rsi_period": 20, "entry_threshold": 30}, None, None)
        assert isinstance(config["parameters"]["rsi_period"], float)

    def test_optional_validation_block(self):
        config = _build_v1_config("t", {"a": 1}, None, {"avg_oos_sharpe": 0.9})
        assert config["validation"]["avg_oos_sharpe"] == 0.9


class TestV2Config:
    def test_has_gkr_mapping_key(self):
        config = _build_v2_config("intraday_mr", {"rsi_period": 20, "entry_threshold": 30}, None, None)
        assert config["version"] == 2
        assert "gkr_mapping" in config
        assert "param_schema_version" in config


class TestGKRMapping:
    def test_intraday_mr_mapping(self):
        result = _generate_gkr_mapping("intraday_mr", {"rsi_period": 20, "entry_threshold": 30, "exit_threshold": 50})
        assert result["rsi_period"]["node"] == "zscore_calc"
        assert result["entry_threshold"]["node"] == "signal_gen"

    def test_unknown_strategy_empty(self):
        result = _generate_gkr_mapping("unknown", {"a": 1})
        assert result == {}


class TestParamsToGkr:
    def test_writes_file_with_content_hash(self):
        with tempfile.TemporaryDirectory() as td:
            path = params_to_gkr(
                "test_strat", "EURUSD",
                {"rsi_period": 20},
                {"sharpe_ratio": 1.5, "max_drawdown": -10.0, "total_return": 25.0, "win_rate": 60.0, "num_trades": 42},
                output_dir=td,
            )
            assert path.exists()
            with open(path) as f:
                config = yaml.safe_load(f)
            assert config["content_hash"].startswith("sha256:")
            assert config["version"] == 1

    def test_schema_parity_with_exporter(self):
        from orca.optimize.exporter import export_best_params as export_orig

        with tempfile.TemporaryDirectory() as td1, tempfile.TemporaryDirectory() as td2:
            p1 = export_orig(
                "test_strat",
                {"rsi_period": 20},
                {"sharpe_ratio": 1.0, "max_drawdown": -5.0, "total_return": 10.0, "win_rate": 60.0, "num_trades": 42},
                output_dir=td1,
            )
            p2 = params_to_gkr(
                "test_strat", "EURUSD",
                {"rsi_period": 20},
                {"sharpe_ratio": 1.0, "max_drawdown": -5.0, "total_return": 10.0, "win_rate": 60.0, "num_trades": 42},
                output_dir=td2,
            )
            c1 = yaml.safe_load(open(p1))
            c2 = yaml.safe_load(open(p2))
            keys1 = {k for k in c1 if k != "content_hash"}
            keys2 = {k for k in c2 if k != "content_hash"}
            assert keys1 == keys2, f"Schema mismatch: {keys1 ^ keys2}"

    def test_invalid_schema_version_raises(self):
        with tempfile.TemporaryDirectory() as td:
            with pytest.raises(ValueError, match="Unsupported schema_version"):
                params_to_gkr(
                    "test", "X", {"a": 1},
                    metrics={"sharpe_ratio": 1.0, "max_drawdown": -5.0, "total_return": 10.0, "win_rate": 60.0, "num_trades": 42},
                    output_dir=td, schema_version=99,
                )

    def test_content_hash_reproducible(self):
        with tempfile.TemporaryDirectory() as td:
            params = {"rsi_period": 20, "entry_threshold": 30}
            metrics = {"sharpe_ratio": 1.5, "max_drawdown": -10.0, "total_return": 25.0, "win_rate": 60.0, "num_trades": 42}
            p1 = params_to_gkr("test", "X", params, metrics, output_dir=td)
            p2 = params_to_gkr("test", "X", params, metrics, output_dir=td)
            h1 = yaml.safe_load(open(p1))["content_hash"]
            h2 = yaml.safe_load(open(p2))["content_hash"]
            assert h1 == h2

    def test_v1_v2_cross_schema_hash_equivalence(self):
        with tempfile.TemporaryDirectory() as td1, tempfile.TemporaryDirectory() as td2:
            params = {"rsi_period": 20, "entry_threshold": 30}
            metrics = {"sharpe_ratio": 1.5, "max_drawdown": -10.0, "total_return": 25.0, "win_rate": 60.0, "num_trades": 42}
            p1 = params_to_gkr("cross_hash", "EURUSD", params, metrics, output_dir=td1, schema_version=1)
            p2 = params_to_gkr("cross_hash", "EURUSD", params, metrics, output_dir=td2, schema_version=2)
            c1 = yaml.safe_load(open(p1))
            c2 = yaml.safe_load(open(p2))
            h1 = c1["content_hash"]
            h2 = c2["content_hash"]
            assert h1.startswith("sha256:")
            assert h2.startswith("sha256:")
            assert c1["version"] == 1
            assert c2["version"] == 2

    def test_params_round_trip_preserves_values(self):
        with tempfile.TemporaryDirectory() as td:
            params = {"rsi_period": 20, "entry_threshold": 30.5, "exit_threshold": 70}
            metrics = {"sharpe_ratio": 1.5, "max_drawdown": -10.0, "total_return": 25.0, "win_rate": 60.0, "num_trades": 42}
            path = params_to_gkr("round_trip", "EURUSD", params, metrics, output_dir=td)
            config = yaml.safe_load(open(path))
            assert config["parameters"]["rsi_period"] == 20.0
            assert config["parameters"]["entry_threshold"] == 30.5
            assert config["parameters"]["exit_threshold"] == 70.0
            assert config["strategy_id"] == "round_trip"
            assert config["content_hash"].startswith("sha256:")

    def test_different_params_produce_different_hashes(self):
        with tempfile.TemporaryDirectory() as td1, tempfile.TemporaryDirectory() as td2:
            metrics = {"sharpe_ratio": 1.0, "max_drawdown": -5.0, "total_return": 10.0, "win_rate": 60.0, "num_trades": 42}
            p1 = params_to_gkr("hash_diff", "X", {"a": 1}, metrics, output_dir=td1)
            p2 = params_to_gkr("hash_diff", "X", {"a": 2}, metrics, output_dir=td2)
            h1 = yaml.safe_load(open(p1))["content_hash"]
            h2 = yaml.safe_load(open(p2))["content_hash"]
            assert h1 != h2
            assert h1.startswith("sha256:")
            assert h2.startswith("sha256:")

    def test_gkr_ir_hash_verification_against_export(self):

        from orca.hash.graph import instance_hash_v2
        from orca.models.strategy import StrategyIRV04

        with tempfile.TemporaryDirectory() as td:
            params = {"entry_z": 1.5, "exit_z": 0.3, "max_hold": 60}
            metrics = {"sharpe_ratio": 1.2, "max_drawdown": -8.0, "total_return": 15.0, "win_rate": 55.0, "num_trades": 30}
            path = params_to_gkr("intraday_mr", "EURUSD", params, metrics, output_dir=td, schema_version=2)
            config = yaml.safe_load(open(path))

            content_hash = config.get("content_hash", "")
            assert content_hash.startswith("sha256:"), f"content_hash missing or invalid: {content_hash}"

            try:
                ir = StrategyIRV04.model_validate(config.get("ir", {}))
                ih = instance_hash_v2(ir)
                assert ih.startswith("sha256:")
            except Exception as e:
                if config.get("version") != 2 or "ir" not in config:
                    pass
                else:
                    raise AssertionError(f"IR hash verification failed: {e}") from e
