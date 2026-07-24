"""Tests for orca.optimize.exporter — GKR YAML config exporter."""

import tempfile

import yaml

from orca.optimize.exporter import export_best_params


class TestExportBestParams:
    def test_exports_file(self):
        with tempfile.TemporaryDirectory() as td:
            path = export_best_params(
                "test_strat",
                {"rsi_period": 20, "entry_threshold": 30.0},
                {"sharpe_ratio": 1.5, "max_drawdown": -10.0, "total_return": 25.0, "win_rate": 60.0, "num_trades": 42},
                output_dir=td,
            )
            assert path.exists()
            assert path.suffix == ".yaml"

    def test_content_hash_present(self):
        with tempfile.TemporaryDirectory() as td:
            path = export_best_params(
                "t", {"a": 1},
                {"sharpe_ratio": 1.0, "max_drawdown": -5.0, "total_return": 10.0, "win_rate": 60.0, "num_trades": 42},
                output_dir=td,
            )
            config = yaml.safe_load(open(path))
            assert "content_hash" in config
            assert len(config["content_hash"]) == 64  # SHA-256 hex digest

    def test_optional_validation_block(self):
        with tempfile.TemporaryDirectory() as td:
            path = export_best_params(
                "t", {"a": 1},
                {"sharpe_ratio": 1.0, "max_drawdown": -5.0, "total_return": 10.0, "win_rate": 60.0, "num_trades": 42},
                validation={"avg_oos_sharpe": 0.89, "passed_windows": 6, "total_windows": 12},
                output_dir=td,
            )
            config = yaml.safe_load(open(path))
            assert config["validation"]["avg_oos_sharpe"] == 0.89

    def test_content_hash_reproducible(self):
        with tempfile.TemporaryDirectory() as td:
            params = {"rsi_period": 20}
            metrics = {"sharpe_ratio": 1.5, "max_drawdown": -10.0, "total_return": 25.0, "win_rate": 60.0, "num_trades": 42}
            p1 = export_best_params("test", params, metrics, output_dir=td)
            p2 = export_best_params("test", params, metrics, output_dir=td)
            h1 = yaml.safe_load(open(p1))["content_hash"]
            h2 = yaml.safe_load(open(p2))["content_hash"]
            assert h1 == h2

    def test_strategy_id_in_filename(self):
        with tempfile.TemporaryDirectory() as td:
            path = export_best_params(
                "intraday_mr", {"rsi_period": 20},
                {"sharpe_ratio": 1.0, "max_drawdown": -5.0, "total_return": 10.0, "win_rate": 60.0, "num_trades": 42},
                output_dir=td,
            )
            assert "intraday_mr" in path.name
