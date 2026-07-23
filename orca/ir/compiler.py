"""GKR Strategy IR → Odin source code compiler.

Parses .gkr.yaml strategy definitions and generates Odin strategy struct
initialization + registry registration code.
"""

from __future__ import annotations

from pathlib import Path

from orca.ir.loader import load_ir
from orca.ir.validator import validate_ir
from orca.models.strategy import Node, StrategyIRV04


def _extract_params(node: Node) -> dict:
    return dict(node.params) if node.params else {}


def _generate_odin_struct(node: Node, strategy_id: int) -> str:
    token = node.token_ref.token_id
    params = _extract_params(node)

    if token == "signal.threshold":
        entry_z = params.get("entry_threshold", 2.0)
        exit_z = params.get("exit_threshold", 0.5)
        return (
            "    {\n"
            "        mr_ctx_ptr := new(strategy.Context_IntradayMR)\n"
            "        mr_ctx := strategy.Strategy_Context{\n"
            f"            strategy_id = {strategy_id},\n"
            "            context_ptr = mr_ctx_ptr,\n"
            "        }\n"
            "        strategy.init_intraday_mr(&mr_ctx)\n"
            f"        mr_ctx_ptr.lookback = 20\n"
            f"        mr_ctx_ptr.entry_z = {entry_z}\n"
            f"        mr_ctx_ptr.exit_z = {exit_z}\n"
            "        _ = strategy.register(strategy.Strategy{\n"
            '            name    = "intraday_mr",\n'
            "            context = mr_ctx,\n"
            "            init    = strategy.init_intraday_mr,\n"
            "            eval    = strategy.eval_intraday_mr,\n"
            "            destroy = strategy.destroy_intraday_mr,\n"
            "        })\n"
            "    }"
        )

    if token in ("signal.trend", "indicator.ma_crossover", "indicator.atr_filter"):
        fast = params.get("fast_period", 20)
        slow = params.get("slow_period", 50)
        atr_period = params.get("atr_period", 14)
        atr_mult = params.get("atr_multiplier", 2.0)
        return (
            "    {\n"
            "        trend_ctx_ptr := new(strategy.Context_TrendFollowing)\n"
            "        trend_ctx := strategy.Strategy_Context{\n"
            f"            strategy_id = {strategy_id},\n"
            "            context_ptr = trend_ctx_ptr,\n"
            "        }\n"
            "        strategy.init_trend(&trend_ctx)\n"
            f"        trend_ctx_ptr.fast_period = {fast}\n"
            f"        trend_ctx_ptr.slow_period = {slow}\n"
            f"        trend_ctx_ptr.atr_period = {atr_period}\n"
            f"        trend_ctx_ptr.atr_multiplier = {atr_mult}\n"
            "        _ = strategy.register(strategy.Strategy{\n"
            '            name    = "trend_following",\n'
            "            context = trend_ctx,\n"
            "            init    = strategy.init_trend,\n"
            "            eval    = strategy.eval_trend,\n"
            "            destroy = strategy.destroy_trend,\n"
            "        })\n"
            "    }"
        )

    if token in ("signal.opening_range", "signal.range_detect", "signal.breakout"):
        minutes = params.get("range_minutes", 5)
        buffer_pct = params.get("entry_buffer_pct", 0.1)
        return (
            "    {\n"
            "        orb_ctx_ptr := new(strategy.Context_ORB)\n"
            "        orb_ctx := strategy.Strategy_Context{\n"
            f"            strategy_id = {strategy_id},\n"
            "            context_ptr = orb_ctx_ptr,\n"
            "        }\n"
            "        strategy.init_orb(&orb_ctx)\n"
            f"        orb_ctx_ptr.range_minutes = {minutes}\n"
            f"        orb_ctx_ptr.entry_buffer_pct = {buffer_pct}\n"
            "        _ = strategy.register(strategy.Strategy{\n"
            '            name    = "opening_range_breakout",\n'
            "            context = orb_ctx,\n"
            "            init    = strategy.init_orb,\n"
            "            eval    = strategy.eval_orb,\n"
            "            destroy = strategy.destroy_orb,\n"
            "        })\n"
            "    }"
        )

    return f"    _ = {strategy_id}"


def compile_strategy(ir: StrategyIRV04, strategy_id: int) -> str:
    signal_nodes = [
        n for n in ir.strategy.nodes
        if n.token_ref.token_id.startswith("signal.")
    ]
    if not signal_nodes:
        signal_nodes = ir.strategy.nodes

    blocks = []
    for node in signal_nodes:
        blocks.append(_generate_odin_struct(node, strategy_id))

    return "\n\n".join(blocks)


def compile_all(
    gkr_dir: str | Path = "configs/strategies",
    output: str | Path = "odin/strategy/generated_strategies.odin",
) -> str:
    gkr_dir = Path(gkr_dir)
    output = Path(output)

    header = (
        "package strategy\n"
        "\n"
        'import "../orderbook"\n'
        'import "../types"\n'
        'import "../risk"\n'
        "\n"
        "@(private)\n"
        "register_generated_strategies :: proc() {\n"
    )
    footer = "}\n"

    blocks = []
    strategy_id = 2
    for yaml_path in sorted(gkr_dir.glob("*.gkr.yaml")):
        ir = load_ir(yaml_path)
        diags = validate_ir(ir, "research")
        errors = [d for d in diags if d.severity == "error"]
        if errors:
            block = f"    // SKIPPED {ir.strategy.id}: validation errors\n"
            for e in errors:
                block += f"    //   {e.code}: {e.message}\n"
            blocks.append(block)
            continue

        compiled = compile_strategy(ir, strategy_id)
        blocks.append(f"    // {ir.strategy.id}\n{compiled}")
        strategy_id += 1

    result = header + "\n".join(blocks) + "\n" + footer
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(result)
    return result


if __name__ == "__main__":
    code = compile_all()
    print(code)
