"""CLI for the Qlib partial workflow adapter."""

from __future__ import annotations

from pathlib import Path
from typing import Annotated

import typer

from qst.adapters.qlib.importer import import_qlib_workflow

app = typer.Typer(no_args_is_help=True, help="Qlib partial workflow adapter")


@app.command("import")
def import_cmd(
    source: Path,
    output: Annotated[Path, typer.Option("--output")],
    coverage: Annotated[Path, typer.Option("--coverage")],
) -> None:
    """Import Qlib workflow YAML as candidate QST records and coverage evidence."""

    result = import_qlib_workflow(source, output_path=output, coverage_path=coverage)
    typer.echo(result.coverage.model_dump_json(indent=2))

