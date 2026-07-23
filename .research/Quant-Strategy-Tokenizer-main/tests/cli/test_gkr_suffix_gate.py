from __future__ import annotations

from pathlib import Path

from typer.testing import CliRunner

from qst.cli import app


def _write_minimal_strategy(path: Path) -> None:
    path.write_text(
        "\n".join(
            [
                "ir_version: qst-ir/0.4",
                "canonical_version: qst-canonical/0.4",
                "strategy:",
                "  id: suffix_gate",
                "  version: 1",
                "  nodes: []",
                "  outputs: {}",
                "metadata: {}",
                "",
            ]
        ),
        encoding="utf-8",
    )


def test_cli_accepts_gkr_yaml_strategy(tmp_path: Path) -> None:
    strategy = tmp_path / "strategy.gkr.yaml"
    _write_minimal_strategy(strategy)

    result = CliRunner().invoke(app, ["validate", str(strategy)])

    assert result.exit_code == 0


def test_cli_rejects_pre_gkr_strategy_suffix(tmp_path: Path) -> None:
    strategy = tmp_path / ("strategy.qst" + ".yaml")
    _write_minimal_strategy(strategy)

    result = CliRunner().invoke(app, ["validate", str(strategy)])

    assert result.exit_code != 0
    assert ".gkr.yaml" in result.output
