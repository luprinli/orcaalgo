# Quant Toolkit

A Cowork plugin for building, validating, and operating quant-trading strategies on prediction markets. Nine skills covering the full loop from idea to live trading, plus runnable Python that backs each one.

Distilled from operating a live Kalshi weather-market bot for six months — the [GEO demo repo](https://github.com/apeabody007/GEO-The-Kalshi-Weather-Bot-DEMO) is the concrete bot this toolkit was abstracted from. This repo is the venue-agnostic framework: bring your own model and your own exchange.

## What's in it

| Skill                   | Use when you need to…                                                            |
| ----------------------- | -------------------------------------------------------------------------------- |
| `kelly-sizer`           | Compute optimal bet size with fractional Kelly, edge discount, and caps          |
| `calibration-audit`     | Check whether your model's "70% confident" actually comes true 70% of the time   |
| `backtest-runner`       | Run a walk-forward historical simulation with honest execution assumptions       |
| `maker-pricing`         | Quote one tick inside the spread to capture the maker-fee tier                   |
| `pnl-attribution`       | Slice realized P&L by side, price bucket, edge, and cohort to find leaks         |
| `pre-flight-checklist`  | Run the 10-item safety check before flipping to live trading                     |
| `emos-bias-correction`  | Fit Platt scaling / EMOS to correct systematic forecast bias                     |
| `market-scanner`        | Surface candidate trades from open markets with a stacked filter pipeline        |
| `drawdown-monitor`      | Track running drawdown and trigger warn / de-risk / halt at configured levels    |

Each skill ships with:

- A `SKILL.md` that Claude reads when you ask about the topic — it guides reasoning, surfaces failure modes, and tells Claude how to respond.
- A runnable Python script (under `scripts/`) — drop-in helpers you can call from your strategy code or run as one-off CLIs.

## Why this exists

Most retail quant tutorials stop at "here's the Kelly formula" or "here's how to run a backtest." That's the easy half. The hard half — the half this toolkit is built around — is everything that goes wrong between the formula and a live, sized order:

- A model that looks well-calibrated on average is wildly miscalibrated on one side (YES vs NO, favorite vs longshot).
- A backtest that ignores maker-vs-taker fees overstates ROI by 5–15 percentage points.
- Sizing at full Kelly when your edge estimate is even slightly wrong drives the bankroll to zero faster than you can react.
- A pre-flight checklist sounds like overkill until the day two processes acquire the same lock and place duplicate live trades.

Each of the nine skills here exists because of a specific incident or a specific paper that re-shaped how the underlying bot decided things. Where relevant, the SKILL.md cites the source.

## How to use it

Once installed, the skills load automatically when you ask Claude relevant questions:

- "How much should I bet on this trade?" → triggers `kelly-sizer`.
- "Is my model calibrated?" → triggers `calibration-audit`.
- "Run a backtest on this strategy" → triggers `backtest-runner`.
- "Where am I losing money?" → triggers `pnl-attribution`.
- "I'm about to go live — pre-flight" → triggers `pre-flight-checklist`.

You can also reach the scripts directly if you'd rather work in code:

```bash
python skills/kelly-sizer/scripts/kelly.py --p 0.62 --price 0.48 --bankroll 1000
python skills/calibration-audit/scripts/calibration_audit.py --csv predictions.csv
python skills/pnl-attribution/scripts/attribute.py --db trades.sqlite --since 2026-01-01
```

## What this plugin assumes you have

- A model that produces a probability for each contract you might trade. The plugin doesn't include a model — bring your own.
- An exchange to trade on (the plugin is venue-agnostic; see [`CONNECTORS.md`](CONNECTORS.md)).
- A trade ledger (CSV or SQLite) for the attribution and drawdown skills.

## What this plugin is opinionated about

- **Fractional Kelly only.** Full Kelly is the wrong default for retail bankrolls. The sizer recommends ¼ Kelly and pushes back if you ask for more.
- **Honest backtests.** Walk-forward by default; no look-ahead; explicit fill-probability assumption; report n and confidence intervals.
- **Asymmetric calibration.** Audit YES and NO sides separately. Most models are systematically biased on one side.
- **Pre-flight every launch.** The 10-item checklist is the difference between a clean rollout and an incident.
- **Halt before deeper losses.** Drawdown monitor halts at 20% by default; this is conservative, not optional.

## What this plugin is NOT

- Not a trading bot. It's a toolkit. You write the strategy; this plugin gives you the math and the discipline.
- Not a model. Calibration tools can correct bias in your model's output, but the underlying signal has to be yours.
- Not financial advice. Trading prediction markets is risky. Use at your own risk.

## Installation

Drop the `.plugin` file into Cowork's plugin install dialog. Claude will scan the skills and load them on demand.

```bash
# or install from this repo manually:
git clone https://github.com/apeabody007/quant-toolkit.git
# then point Cowork at the cloned directory
```

## See also

- [GEO — The Kalshi Weather Bot (demo)](https://github.com/apeabody007/GEO-The-Kalshi-Weather-Bot-DEMO) — the live trading bot this toolkit was distilled from. Frozen snapshot for portfolio purposes; not actively developed in public.

## License

MIT. See [`LICENSE`](LICENSE).

## References

The skills cite a working bibliography across Kelly criterion theory, calibration scoring rules, market microstructure, and weather forecasting (the original use case). Read any SKILL.md's References section for the relevant papers.
