"""OrcaAlgo CLI — Strategy IR, mathematical models, calibration, and attribution."""

from __future__ import annotations

from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import typer

app = typer.Typer(
    name="orca",
    help="OrcaAlgo — Strategy IR, mathematical models, calibration, and attribution",
)


@app.command()
def validate(
    path: str = typer.Argument(..., help="Path to .gkr.yaml strategy file"),
    profile: str = typer.Option("research", help="Validation profile: research|paper|pretrade|production_guarded"),
) -> None:
    """Validate a GKR strategy file."""
    from orca.ir.loader import load_ir
    from orca.ir.validator import validate_ir

    ir = load_ir(Path(path))
    diags = validate_ir(ir, profile)

    from orca.hash.graph import graph_hash_v2, instance_hash_v2, param_hash_v2

    graph_h = graph_hash_v2(ir)
    param_h = param_hash_v2(ir)
    instance_h = instance_hash_v2(ir)

    typer.echo(f"Strategy: {ir.strategy.id} v{ir.strategy.version}")
    typer.echo(f"  graph_hash:    {graph_h}")
    typer.echo(f"  param_hash:    {param_h}")
    typer.echo(f"  instance_hash: {instance_h}")

    errors = [d for d in diags if d.severity == "error"]
    warnings = [d for d in diags if d.severity == "warning"]

    if errors:
        typer.echo(f"\n{len(errors)} error(s):")
        for d in errors:
            typer.echo(f"  [{d.code}] {d.message}")
        raise typer.Exit(code=1)

    if warnings:
        typer.echo(f"\n{len(warnings)} warning(s):")
        for d in warnings:
            typer.echo(f"  [{d.code}] {d.message}")

    typer.echo(f"\nValidation passed for profile '{profile}'.")


@app.command()
def calibrate(
    since: str = typer.Option("30d", help="Lookback period (e.g., 30d, 90d)"),
    output: str | None = typer.Option(None, help="Output JSON file path"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Run calibration audit against trade ledger."""
    import json as _json

    try:
        from orca.calibration.audit import run_calibration_audit
        from orca.calibration.cli import _load_trades_for_calibration, _resolve_since

        parsed_since = _resolve_since(since)
        trades = _load_trades_for_calibration(parsed_since)
        report = run_calibration_audit(trades)

        if json_output:
            typer.echo(_json.dumps(report, indent=2, default=str))
            return

        out_path = output or f"reports/calibration-{datetime.now(UTC).strftime('%Y%m%d')}.json"
        Path(out_path).parent.mkdir(parents=True, exist_ok=True)
        Path(out_path).write_text(_json.dumps(report, indent=2, default=str))

        typer.echo(f"Calibration report written to {out_path}")
    except ImportError:
        if json_output:
            typer.echo(_json.dumps({"error": "Calibration module not available"}))
        else:
            typer.echo("Calibration module not available.", err=True)
        raise typer.Exit(code=1)


@app.command(name="hash")
def hash_cmd(
    path: str = typer.Argument(..., help="Path to .gkr.yaml strategy file"),
    graph: bool = typer.Option(False, "--graph", help="Output graph hash only"),
    param: bool = typer.Option(False, "--param", help="Output param hash only"),
    instance: bool = typer.Option(True, "--instance/--no-instance", help="Output instance hash (default)"),
    json_output: bool = typer.Option(False, "--json", help="Output all hashes as JSON"),
) -> None:
    """Compute content-addressable hashes for a GKR strategy file.

    By default outputs the instance hash (composite of graph + param hashes).
    Use --graph or --param for individual hashes, or --json for all three.
    """
    import json as _json

    from orca.hash.graph import graph_hash_v2, instance_hash_v2, param_hash_v2
    from orca.ir.loader import load_ir

    ir = load_ir(Path(path))
    gh = graph_hash_v2(ir)
    ph = param_hash_v2(ir)
    ih = instance_hash_v2(ir)

    if json_output:
        typer.echo(_json.dumps({
            "graph_hash": gh,
            "param_hash": ph,
            "instance_hash": ih,
        }))
        return

    if graph:
        typer.echo(gh)
    elif param:
        typer.echo(ph)
    else:
        typer.echo(ih)


@app.command()
def preflight(
    json_output: bool = typer.Option(False, "--json", help="Output results as JSON"),
) -> None:
    """Run pre-flight checklist before live deployment."""
    try:
        from orca.preflight.checklist import run_preflight_checks

        results = run_preflight_checks()
        passed = sum(1 for r in results if r.status == "pass")
        warned = sum(1 for r in results if r.status == "warn")
        failed = sum(1 for r in results if r.status == "fail")

        if json_output:
            import json as _json

            typer.echo(
                _json.dumps(
                    {
                        "passed": failed == 0,
                        "passed_count": passed,
                        "warned_count": warned,
                        "failed_count": failed,
                        "checks": [
                            {"name": r.check_name, "status": r.status, "message": r.message}
                            for r in results
                        ],
                    },
                    indent=2,
                )
            )
            return

        typer.echo(f"Pre-Flight Results: {passed} pass, {warned} warn, {failed} fail")

        for r in results:
            icon = {"pass": "[PASS]", "warn": "[WARN]", "fail": "[FAIL]"}[r.status]
            typer.echo(f"  {icon} {r.check_name}: {r.message}")

        if failed > 0:
            typer.echo("\nPre-flight FAILED. Do not deploy to production.", err=True)
            raise typer.Exit(code=1)
        typer.echo("\nPre-flight PASSED.")
    except ImportError:
        if json_output:
            import json as _json

            typer.echo(
                _json.dumps(
                    {
                        "passed": False,
                        "passed_count": 0,
                        "warned_count": 0,
                        "failed_count": 1,
                        "checks": [
                            {
                                "name": "orca_package",
                                "status": "fail",
                                "message": "Pre-flight module not available",
                            }
                        ],
                    }
                )
            )
        else:
            typer.echo("Pre-flight module not available.", err=True)
        raise typer.Exit(code=1)


@app.command()
def attribute(
    since: str = typer.Option("90d", help="Lookback period (e.g., 30d, 90d)"),
    output: str | None = typer.Option(None, help="Output JSON file path"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Run PnL attribution on trade ledger."""
    import json as _json

    try:
        from orca.attribution.cli import _load_trades_for_attribution, _resolve_since
        from orca.attribution.slicer import attribute_pnl

        parsed_since = _resolve_since(since)
        trades = _load_trades_for_attribution(parsed_since)
        report = attribute_pnl(trades)

        if json_output:
            typer.echo(_json.dumps(report, indent=2, default=str))
            return

        out_path = output or f"reports/attribution-{datetime.now(UTC).strftime('%Y%m%d')}.json"
        Path(out_path).parent.mkdir(parents=True, exist_ok=True)
        Path(out_path).write_text(_json.dumps(report, indent=2, default=str))

        typer.echo(f"Attribution report written to {out_path}")
    except ImportError:
        if json_output:
            typer.echo(_json.dumps({"error": "Attribution module not available"}))
        else:
            typer.echo("Attribution module not available.", err=True)
        raise typer.Exit(code=1)


@app.command()
def data_validate(
    universe: bool = typer.Option(False, "--universe", help="Validate all symbols"),
    symbols: str | None = typer.Option(None, "--symbols", help="Comma-separated symbols to validate"),
    timeframe: str = typer.Option("1d", "--timeframe", help="Timeframe to validate"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Validate data quality before running backtests.

    Checks include: gap detection, outlier detection, volume sanity,
    timestamp consistency, and completeness ratio.
    """
    import json as _json

    try:
        from orca.data_quality.validator import run_data_quality_checks

        report = run_data_quality_checks()

        if json_output:
            typer.echo(
                _json.dumps(
                    {
                        "passed": report.failed == 0,
                        "passed_count": report.passed,
                        "warned_count": report.warned,
                        "failed_count": report.failed,
                        "checks": [
                            {"name": c.check_name, "status": c.status, "message": c.message, "symbol": c.symbol, "timeframe": c.timeframe}
                            for c in report.checks
                        ],
                    },
                    indent=2,
                )
            )
            return

        if symbols:
            typer.echo(f"Validating symbols: {symbols}")
        if timeframe:
            typer.echo(f"Timeframe: {timeframe}")

        typer.echo(f"Data Quality Results: {report.passed} pass, {report.warned} warn, {report.failed} fail")
        typer.echo("")

        for c in report.checks:
            icon = {"pass": "[PASS]", "warn": "[WARN]", "fail": "[FAIL]"}[c.status]
            loc = f" ({c.symbol}:{c.timeframe})" if c.symbol else ""
            typer.echo(f"  {icon} {c.check_name}{loc}: {c.message}")

        if report.failed > 0:
            typer.echo("\nData quality checks FAILED. Fix issues before running backtests.", err=True)
            raise typer.Exit(code=1)
        if report.warned > 0:
            typer.echo(f"\n{report.warned} warning(s). Review before production deployment.")
        else:
            typer.echo("\nAll checks PASSED.")
    except ImportError:
        if json_output:
            typer.echo(_json.dumps({"passed": False, "failed_count": 1, "checks": [{"name": "data_quality", "status": "fail", "message": "Data quality module not available"}]}))
        else:
            typer.echo("Data quality module not available.", err=True)
        raise typer.Exit(code=1)


@app.command()
def ir_compile(
    path: str = typer.Option(None, "--path", help="Path to a single .gkr.yaml file"),
    target: str = typer.Option("go", "--target", help="Target output format: go (Go JSON config)"),
) -> None:
    """Compile a GKR strategy IR file into a Go-engine-compatible JSON configuration."""
    import json as _json

    try:
        from orca.ir.compiler import compile_all_go_configs, compile_to_go_config

        if path:
            config = compile_to_go_config(Path(path))
            typer.echo(_json.dumps(config, indent=2))
        else:
            configs = compile_all_go_configs()
            typer.echo(_json.dumps(configs, indent=2))
    except ImportError as e:
        typer.echo(_json.dumps({"error": f"Compiler module not available: {e}"}), err=True)
        raise typer.Exit(code=1)


simulate_app = typer.Typer(help="Synthetic data simulation pipeline")
app.add_typer(simulate_app, name="simulate")


@simulate_app.command()
def sim_calibrate(
    symbols: str = typer.Option(..., "--symbols", help="Comma-separated symbols to calibrate"),
    timeframe: str = typer.Option("1d", "--timeframe", help="Timeframe for calibration"),
    start: str | None = typer.Option(None, "--start", help="Start date (YYYY-MM-DD)"),
    end: str | None = typer.Option(None, "--end", help="End date (YYYY-MM-DD)"),
    output_dir: str = typer.Option("configs/simulation", "--output-dir", help="Output directory"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Calibrate statistical parameters from real data for synthetic generation."""
    import json as _json

    parsed_symbols = [s.strip() for s in symbols.split(",")]
    start_dt = datetime.fromisoformat(start) if start else None
    end_dt = datetime.fromisoformat(end) if end else None

    try:
        from orca.simulation.calibrate import calibrate_all

        results = calibrate_all(
            symbols=parsed_symbols,
            timeframe=timeframe,
            start=start_dt,
            end=end_dt,
            output_dir=output_dir,
        )

        if json_output:
            typer.echo(_json.dumps(results, indent=2, default=str))
        else:
            for sym, path in results.items():
                typer.echo(f"  {sym}: {path}")
    except ImportError:
        typer.echo("Simulation calibration module not available.", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def generate_1m(
    symbols: str = typer.Option(..., "--symbols", help="Comma-separated symbols"),
    start: str = typer.Option(..., "--start", help="Start date (YYYY-MM-DD)"),
    end: str = typer.Option(..., "--end", help="End date (YYYY-MM-DD)"),
    model: str = typer.Option("heston", "--model", help="Model: gbm, ou, heston"),
    seed: int | None = typer.Option(None, "--seed", help="Random seed for reproducibility"),
    volume_profile: str = typer.Option("u_shaped", "--volume-profile", help="Volume profile: u_shaped, sine, flat"),
    timeframe: str = typer.Option("1d", "--timeframe", help="Calibration timeframe"),
    output_dir: str = typer.Option("data/synthetic/1m", "--output-dir", help="Output directory"),
    calibration_dir: str = typer.Option("configs/simulation", "--calibration-dir", help="Calibration directory"),
    daily_volume: float | None = typer.Option(None, "--daily-volume", help="Average daily volume override"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Generate synthetic 1-minute OHLCV candles."""
    import json as _json

    parsed_symbols = [s.strip() for s in symbols.split(",")]
    start_dt = datetime.fromisoformat(start)
    end_dt = datetime.fromisoformat(end)

    try:
        from orca.simulation.generate_1m import generate_and_save

        results: dict[str, str] = {}
        for symbol in parsed_symbols:
            calib_path = Path(calibration_dir) / f"calibration_{symbol}.json"
            calibration = None
            if calib_path.exists():
                calibration = _json.loads(calib_path.read_text())

            gen_id = generate_and_save(
                symbol=symbol,
                start=start_dt,
                end=end_dt,
                model=model,
                calibration=calibration,
                seed=seed,
                volume_profile=volume_profile,
                timeframe=timeframe,
                output_dir=output_dir,
                daily_volume_avg=daily_volume,
            )
            results[symbol] = gen_id

        if json_output:
            typer.echo(_json.dumps(results, indent=2))
        else:
            for sym, gen_id in results.items():
                typer.echo(f"  {sym}: generation_id={gen_id}")
    except ImportError:
        typer.echo("Simulation generation module not available.", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def ticks(
    generation_id: str = typer.Option(..., "--generation-id", help="Generation ID from generate-1m"),
    symbol: str = typer.Option(..., "--symbol", help="Symbol"),
    ticks_per_minute: int = typer.Option(60, "--ticks-per-minute", help="Ticks to generate per minute"),
    spread_bps: float = typer.Option(0.5, "--spread-bps", help="Bid-ask spread in basis points"),
    volume_profile: str = typer.Option("sine", "--volume-profile", help="Volume distribution shape"),
    seed: int | None = typer.Option(None, "--seed", help="Random seed"),
    output_dir: str = typer.Option("data/synthetic/ticks", "--output-dir", help="Output directory"),
    candle_dir: str = typer.Option("data/synthetic/1m", "--candle-dir", help="Candle Parquet directory"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Disaggregate 1-minute candles into tick-level data."""
    import json as _json

    try:
        from orca.simulation.tick_disaggregator import disaggregate_and_save, load_candle_parquet

        candle_path = Path(candle_dir) / symbol / generation_id
        if not candle_path.exists():
            typer.echo(f"Candle data not found at {candle_path}", err=True)
            raise typer.Exit(code=1)

        df = load_candle_parquet(candle_path)
        ticks_df = disaggregate_and_save(
            candle_df=df,
            generation_id=generation_id,
            symbol=symbol,
            ticks_per_minute=ticks_per_minute,
            spread_bps=spread_bps,
            volume_profile=volume_profile,
            seed=seed,
            output_dir=output_dir,
        )

        result = {
            "generation_id": generation_id,
            "symbol": symbol,
            "n_ticks": len(ticks_df),
        }

        if json_output:
            typer.echo(_json.dumps(result, indent=2))
        else:
            typer.echo(f"  symbol={symbol} generation_id={generation_id} ticks={len(ticks_df)}")
    except ImportError:
        typer.echo("Tick disaggregation module not available.", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def sim_validate(
    generation_id: str = typer.Option(..., "--generation-id", help="Generation ID to validate"),
    symbol: str = typer.Option(..., "--symbol", help="Symbol"),
    timeframe: str = typer.Option("1d", "--timeframe", help="Timeframe for comparison"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Validate synthetic data against real data distributions."""
    import json as _json

    try:
        from orca.simulation.validate import validate_generation

        report = validate_generation(generation_id, symbol, timeframe)
        passed = report.get("passed", False)

        if json_output:
            typer.echo(_json.dumps(report, indent=2, default=str))
        else:
            typer.echo(f"Validation: {'PASSED' if passed else 'FAILED'}")
            for check in report.get("checks", []):
                icon = "[PASS]" if check.get("passed") else "[FAIL]"
                typer.echo(f"  {icon} {check['name']}: {check.get('detail', '')}")

        if not passed:
            raise typer.Exit(code=1)
    except ImportError:
        typer.echo("Simulation validation module not available.", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def generate_regime(
    symbol: str = typer.Option("SPY", "--symbol", help="Symbol to generate"),
    start: str = typer.Option("2020-01-01", "--start", help="Start date (YYYY-MM-DD)"),
    end: str = typer.Option("2024-12-31", "--end", help="End date (YYYY-MM-DD)"),
    model: str = typer.Option("heston", "--model", help="Model: gbm, heston"),
    seed: int | None = typer.Option(None, "--seed", help="Random seed"),
    config_dir: str = typer.Option("configs/simulation", "--config-dir", help="Config directory"),
    output_dir: str = typer.Option("data/synthetic/regime", "--output-dir", help="Output directory"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
    ticks: bool = typer.Option(False, "--ticks", help="Also generate tick data"),
    use_factor_model: bool = typer.Option(False, "--use-factor-model", help="Use multi-factor generation instead of pure Heston"),
) -> None:
    """Generate regime-aware synthetic 1-minute data with embedded regime labels."""
    import json as _json

    try:
        from orca.simulation.regime_generator import (
            generate_regime_aware,
            generate_regime_ticks,
        )

        gen_id, candles_df, labels, state = generate_regime_aware(
            symbol=symbol,
            start_date=start,
            end_date=end,
            model=model,
            config_dir=config_dir,
            output_dir=output_dir,
            seed=seed,
            use_factor_model=use_factor_model,
            progress_callback=lambda p: typer.echo(
                f"  Progress: {p['progress_pct']}% ({p['completed_days']}/{p['total_days']} days, "
                f"regime={p['current_regime']})"
            ),
        )

        result = {
            "generation_id": gen_id,
            "symbol": symbol,
            "n_candles": len(candles_df),
            "n_trading_days": state.total_days,
            "model": "factor" if use_factor_model else model,
            "regime_distribution": {
                ["Calm", "Trending", "HighVol", "Crisis"][k]: int((labels == k).sum()) if k < 4 and len(labels) > 0 else 0
                for k in range(4)
            },
        }

        if ticks and len(candles_df) > 0:
            ticks_df = generate_regime_ticks(
                candles_df=candles_df,
                generation_id=gen_id,
                output_dir=output_dir,
                seed=seed,
            )
            result["n_ticks"] = len(ticks_df)

        if json_output:
            typer.echo(_json.dumps(result, indent=2, default=str))
        else:
            typer.echo(f"  symbol={symbol} generation_id={gen_id}")
            typer.echo(f"  candles={len(candles_df)} days={state.total_days}")
            typer.echo(f"  regime distribution: {result['regime_distribution']}")
            if ticks and "n_ticks" in result:
                typer.echo(f"  ticks={result['n_ticks']}")
    except ImportError as e:
        typer.echo(f"Regime generation module not available: {e}", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def inject_signal(
    generation_id: str = typer.Option(..., "--generation-id", help="Existing generation ID to inject into"),
    strategy: str = typer.Option(..., "--strategy", help="Signal type: trend, mean_reversion, breakout"),
    strength: float = typer.Option(0.3, "--strength", help="Signal strength (0-1)"),
    output: str = typer.Option("data/synthetic/signals", "--output", help="Output directory"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Inject structured signal into an existing synthetic dataset."""
    import json as _json
    import os as _os

    try:
        from orca.simulation.signal_injector import (
            BreakoutInjector,
            MeanReversionInjector,
            TrendInjector,
        )

        input_path = f"data/synthetic/regime/{generation_id}/"
        if not _os.path.exists(input_path):
            input_path = f"data/synthetic/1m/{generation_id}/"
        if not _os.path.exists(input_path):
            typer.echo(f"Error: generation {generation_id} not found in data/synthetic/regime/ or data/synthetic/1m/", err=True)
            raise typer.Exit(code=1)

        results: list[dict[str, Any]] = []
        for file in _os.listdir(input_path):
            if not file.endswith(".parquet"):
                continue

            import numpy as np
            import pandas as pd

            df = pd.read_parquet(_os.path.join(input_path, file))
            symbol_name = file.replace(".parquet", "")
            prices = df["close"].values

            if strategy == "trend":
                injector = TrendInjector(strength=strength)
            elif strategy == "mean_reversion":
                injector = MeanReversionInjector(strength=strength)
            elif strategy == "breakout":
                injector = BreakoutInjector(strength=strength)
            else:
                typer.echo(f"Unknown strategy: {strategy}", err=True)
                raise typer.Exit(code=1)

            new_prices = injector.inject(prices)
            df["close"] = new_prices
            df["open"] = np.roll(new_prices, 1)
            df["open"].iloc[0] = new_prices[0]
            df["high"] = np.maximum(df["open"], new_prices)
            df["low"] = np.minimum(df["open"], new_prices)

            out_path = _os.path.join(output, generation_id, f"{symbol_name}.parquet")
            _os.makedirs(_os.path.dirname(out_path), exist_ok=True)
            df.to_parquet(out_path)
            results.append({"symbol": symbol_name, "output": out_path})
            typer.echo(f"Injected {strategy} signal into {symbol_name} -> {out_path}")

        if json_output:
            typer.echo(_json.dumps({"generation_id": generation_id, "strategy": strategy, "files": results}, indent=2))
    except ImportError as e:
        typer.echo(f"Signal injector module not available: {e}", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def bootstrap(
    symbols: str = typer.Option(..., "--symbols", help="Comma-separated symbols"),
    start: str = typer.Option("2020-01-01", "--start", help="Start date (YYYY-MM-DD)"),
    end: str | None = typer.Option(None, "--end", help="End date (YYYY-MM-DD)"),
    output: str = typer.Option("data/synthetic/bootstrap", "--output", help="Output directory"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Generate synthetic data using residual bootstrap from real data."""
    import json as _json
    import os as _os

    try:
        from orca.simulation.residual_bootstrap import bootstrap_generate

        parsed_symbols = [s.strip() for s in symbols.split(",")]
        results: list[dict[str, Any]] = []

        for sym in parsed_symbols:
            df = bootstrap_generate(sym, start, end, lookback_years=5)
            out_path = _os.path.join(output, f"{sym}.parquet")
            _os.makedirs(_os.path.dirname(out_path), exist_ok=True)
            df.to_parquet(out_path)
            results.append({"symbol": sym, "n_candles": len(df), "output": out_path})
            typer.echo(f"Generated {sym} ({len(df)} candles) -> {out_path}")

        if json_output:
            typer.echo(_json.dumps({"results": results}, indent=2))
    except ImportError as e:
        typer.echo(f"Bootstrap module not available: {e}", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def calibrate_regime(
    symbols: str = typer.Option(..., "--symbols", help="Comma-separated symbols"),
    timeframe: str = typer.Option("1d", "--timeframe", help="Timeframe for calibration"),
    start: str | None = typer.Option(None, "--start", help="Start date (YYYY-MM-DD)"),
    end: str | None = typer.Option(None, "--end", help="End date (YYYY-MM-DD)"),
    output_dir: str = typer.Option("configs/simulation", "--output-dir", help="Output directory"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Calibrate model parameters separately for each market regime."""
    import json as _json

    parsed_symbols = [s.strip() for s in symbols.split(",")]
    start_dt = datetime.fromisoformat(start) if start else None
    end_dt = datetime.fromisoformat(end) if end else None

    try:
        from orca.simulation.calibrate_regime import calibrate_per_regime, save_regime_params

        results: dict[str, str] = {}
        for symbol in parsed_symbols:
            params = calibrate_per_regime(
                symbol=symbol, start=start_dt, end=end_dt, timeframe=timeframe,
            )
            path = save_regime_params(params, symbol, output_dir)
            results[symbol] = str(path)

        if json_output:
            typer.echo(_json.dumps(results, indent=2))
        else:
            for sym, path in results.items():
                typer.echo(f"  {sym}: {path}")
    except ImportError as e:
        typer.echo(f"Regime calibration module not available: {e}", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def validate_regime(
    generation_id: str = typer.Option(..., "--generation-id", help="Generation ID to validate"),
    symbol: str = typer.Option(..., "--symbol", help="Symbol"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
    coverage: bool = typer.Option(False, "--coverage", help="Also run strategy coverage validation"),
    min_sharpe: float = typer.Option(0.3, "--min-sharpe", help="Minimum Sharpe ratio for strategy coverage"),
) -> None:
    """Validate regime-aware synthetic data: regime persistence, strategy coverage."""
    import json as _json

    try:
        from orca.simulation.regime import DEFAULT_TRANSITION_MATRIX, RegimeSequenceGenerator

        label_path = Path(f"data/synthetic/regime/{symbol}_{generation_id}_labels.npy")
        if not label_path.exists():
            typer.echo(f"Regime labels not found at {label_path}", err=True)
            raise typer.Exit(code=1)

        import numpy as np
        labels = np.load(label_path)

        gen = RegimeSequenceGenerator()
        avg_durations = gen.get_avg_durations(labels)

        expected_durations = {
            "Calm": 60, "Trending": 40, "HighVol": 20, "Crisis": 8,
        }

        checks = []
        all_passed = True
        for regime_id, expected in expected_durations.items():
            actual = avg_durations.get(regime_id, 0.0)
            within_tolerance = abs(actual - expected) <= expected * 0.30
            checks.append({
                "name": f"Regime {regime_id} duration",
                "expected": expected,
                "actual": round(actual, 1),
                "passed": within_tolerance,
                "detail": f"{actual:.1f} days (target: {expected}, tolerance: ±30%)",
            })
            if not within_tolerance:
                all_passed = False

        report = {
            "generation_id": generation_id,
            "symbol": symbol,
            "passed": all_passed,
            "avg_durations": {str(k): round(v, 1) for k, v in avg_durations.items()},
            "checks": checks,
        }

        if coverage:
            from orca.simulation.validate import validate_strategy_coverage
            cov_report = validate_strategy_coverage(
                generation_id=generation_id,
                min_sharpe=min_sharpe,
                symbol=symbol,
            )
            report["coverage"] = cov_report
            if not cov_report.get("passed", False):
                all_passed = False
            report["passed"] = all_passed

        if json_output:
            typer.echo(_json.dumps(report, indent=2, default=str))
        else:
            typer.echo(f"Regime Validation: {'PASSED' if all_passed else 'FAILED'}")
            for check in checks:
                icon = "[PASS]" if check["passed"] else "[FAIL]"
                typer.echo(f"  {icon} {check['name']}: {check['detail']}")
            if coverage and "coverage" in report:
                typer.echo(f"  Strategy Coverage: {'PASSED' if report['coverage'].get('passed', False) else 'FAILED'}")
                for strat, sres in report["coverage"].get("strategies", {}).items():
                    if "error" in sres:
                        typer.echo(f"    {strat}: ERROR - {sres['error']}")
                    else:
                        icon = "[PASS]" if sres.get("passed") else "[FAIL]"
                        typer.echo(f"    {icon} {strat}: Sharpe={sres.get('sharpe_ratio', 'N/A')} Trades={sres.get('num_trades', 0)}")

        if not all_passed:
            raise typer.Exit(code=1)
    except ImportError as e:
        typer.echo(f"Regime validation module not available: {e}", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def status(
    batch_id: str | None = typer.Option(None, "--batch-id", help="Specific batch ID"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Show progress of all active or a specific synthetic generation batch."""
    import json as _json

    try:
        from orca.simulation.progress import list_active_batches, read_progress

        if batch_id:
            progress = read_progress(batch_id)
            if progress is None:
                typer.echo(f"No progress found for batch {batch_id}")
                raise typer.Exit(code=1)
            if json_output:
                typer.echo(_json.dumps(progress, indent=2))
            else:
                _print_progress(progress)
        else:
            batches = list_active_batches()
            if not batches:
                typer.echo("No active batches.")
                return
            if json_output:
                typer.echo(_json.dumps(batches, indent=2))
            else:
                typer.echo(f"{len(batches)} active batch(es):")
                for bp in batches:
                    _print_progress(bp, compact=True)
                    typer.echo("")
    except ImportError as e:
        typer.echo(f"Progress module not available: {e}", err=True)
        raise typer.Exit(code=1)


@simulate_app.command()
def halt(
    batch_id: str = typer.Option(..., "--batch-id", help="Batch ID to halt"),
    json_output: bool = typer.Option(False, "--json", help="Output result as JSON"),
) -> None:
    """Gracefully halt a running synthetic generation batch."""
    import json as _json

    try:
        from orca.simulation.progress import halt_batch

        ok = halt_batch(batch_id)
        if json_output:
            typer.echo(_json.dumps({"batch_id": batch_id, "halted": ok}))
        else:
            if ok:
                typer.echo(f"Halt requested for batch {batch_id}. The process will stop at the next checkpoint.")
            else:
                typer.echo(f"Failed to halt batch {batch_id}.")
                raise typer.Exit(code=1)
    except ImportError as e:
        typer.echo(f"Progress module not available: {e}", err=True)
        raise typer.Exit(code=1)


def _print_progress(progress: dict, compact: bool = False) -> None:
    """Format progress dict for terminal display."""
    bid = progress.get("batch_id", "?")
    status = progress.get("status", "?")
    pct = progress.get("progress_pct", 0)
    desc = progress.get("description", "")
    completed = progress.get("completed", 0)
    total = progress.get("total", 0)
    elapsed = progress.get("elapsed_s", 0)
    eta = progress.get("eta_s")

    bar_width = 30
    filled = int(bar_width * pct / 100)
    bar = "\u2588" * filled + "\u2591" * (bar_width - filled)

    if compact:
        typer.echo(f"  [{bid[:8]}] {bar} {pct:.0f}%  {status}")
    else:
        typer.echo(f"  Batch:     {bid}")
        typer.echo(f"  Status:    {status}")
        if desc:
            typer.echo(f"  Description: {desc}")
        typer.echo(f"  Progress:  {bar} {pct:.1f}%")
        typer.echo(f"  Items:     {completed}/{total} (failed: {progress.get('failed', 0)})")
        typer.echo(f"  Elapsed:   {elapsed:.0f}s")
        if eta:
            typer.echo(f"  ETA:       {eta:.0f}s")
        extra = progress.get("extra", {})
        if extra:
            for k, v in extra.items():
                typer.echo(f"  {k}: {v}")


if __name__ == "__main__":
    app()
