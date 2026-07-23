"""OrcaAlgo optimization CLI — VectorBT-powered strategy parameter discovery."""

import json
import sys

from orca.optimize.exporter import export_best_params
from orca.optimize.indicator_factory import get_config
from orca.optimize.monte_carlo import monte_carlo_pass_probability
from orca.optimize.sweeper import sweep_strategy
from orca.optimize.walk_forward import walk_forward_validate


def main():
    args = _parse_args(sys.argv[1:])
    if not args.get("csv") or not args.get("strategy"):
        print("Usage: python -m orca.optimize.cli --csv data/eurusd_1d.csv --strategy intraday_mr")
        print("  --csv       Path to OHLCV CSV")
        print("  --strategy  Strategy ID (intraday_mr, trend_following, ...)")
        print("  --method    grid | random | bayesian (default: grid)")
        print("  --n-trials  Number of random trials (default: 1000)")
        print("  --wfo       Run walk-forward validation")
        print("  --mc        Run Monte Carlo on best result")
        return 1

    csv_path = args["csv"]
    strategy_id = args["strategy"]
    method = args.get("method", "grid")
    n_trials = int(args.get("n_trials", 1000))
    do_wfo = args.get("wfo", False)
    do_mc = args.get("mc", False)

    config = get_config(strategy_id)
    print(f"Strategy: {strategy_id} — {config['description']}")
    print(f"Data: {csv_path}")
    print(f"Method: {method}")

    result = sweep_strategy(strategy_id, csv_path, config.get("param_grid"), method, n_trials)
    print(json.dumps(result, indent=2, default=str))

    if do_wfo:
        print("\n--- Walk-Forward Validation ---")
        wfo = walk_forward_validate(csv_path, strategy_id, config.get("param_grid", {}))
        print(json.dumps({k: v for k, v in wfo.items() if k != "windows"}, indent=2, default=str))
        result["walk_forward"] = wfo

    if do_mc:
        print("\n--- Monte Carlo ---")
        trades = []
        best = result["best_metrics"]
        _ = best
        mc = monte_carlo_pass_probability(trades, n_simulations=min(n_trials, 10000))
        print(json.dumps(mc, indent=2, default=str))

    if result.get("best_params"):
        path = export_best_params(
            strategy_id,
            result["best_params"],
            result["best_metrics"],
            result.get("walk_forward"),
        )
        print(f"\nConfig written to: {path}")

    return 0


def _parse_args(argv: list[str]) -> dict:
    args = {}
    i = 0
    while i < len(argv):
        a = argv[i]
        if a.startswith("--"):
            key = a[2:].replace("-", "_")
            if i + 1 < len(argv) and not argv[i + 1].startswith("--"):
                args[key] = argv[i + 1]
                i += 2
            else:
                args[key] = True
                i += 1
        else:
            i += 1
    return args


if __name__ == "__main__":
    sys.exit(main())
