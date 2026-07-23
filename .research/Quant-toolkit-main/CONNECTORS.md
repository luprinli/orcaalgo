# Connectors

## How tool references work

This plugin is venue-agnostic. Skills reference *categories* of tools rather than specific products, using `~~category` placeholders. Replace each placeholder with the concrete tool you connect.

## Connectors for this plugin

| Category         | Placeholder         | Options                                                 |
| ---------------- | ------------------- | ------------------------------------------------------- |
| Prediction-market exchange | `~~exchange`        | Kalshi, Polymarket, PredictIt, Manifold, Betfair Exchange |
| Trade ledger     | `~~ledger`          | SQLite file, Postgres, CSV, the exchange's own positions API |
| Alert channel    | `~~alerts`          | Telegram, Slack, Discord, email, SMS                    |
| Model output     | `~~forecast`        | Your own model API, an upstream service, a CSV / DB column |

## How placeholders show up in skills

The skills do not pin to a specific exchange's API or auth. When a skill says "fetch the open markets from `~~exchange`," you read this as "use whichever exchange you've configured." Examples:

- `kelly-sizer` works with any contract priced as a binary [0, 1] payoff.
- `calibration-audit` works on any `(forecast_p, outcome)` pair regardless of source.
- `market-scanner` and `maker-pricing` assume a standard order book (best bid, best ask, tick size); the data adapter is yours to write.
- `pnl-attribution` reads from `~~ledger` and is agnostic to schema beyond the columns documented in each skill.

## Setup checklist

1. Pick your `~~exchange`. Read the exchange's API docs and write a thin client that exposes `list_markets()`, `get_book(ticker)`, `place_order(ticker, side, price, size)`.
2. Pick your `~~ledger` format. The reference scripts in this plugin support SQLite and CSV out of the box.
3. Pick your `~~alerts` channel. Wire it into the pre-flight checklist and drawdown monitor.
4. Plug your `~~forecast` model into the scanner and Kelly sizer as the `forecast(market)` function.
