# Non-Goals

QST is not:

- a trading bot
- a broker adapter
- an exchange adapter
- a live execution system
- a full backtest engine
- a Qlib runtime replacement
- a qrun launcher
- a model training system
- an inference execution system
- a portfolio optimizer execution engine
- a lossless Qlib converter
- an arbitrary Python strategy parser
- a production risk system
- a profitability claim

The Qlib adapter proof is intentionally narrow. It reads YAML workflow metadata
and writes candidate QST records plus coverage JSON. It does not execute Qlib
or connect to markets.
