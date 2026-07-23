# Adapter Boundary

QST includes one final adapter proof: a Qlib partial workflow YAML importer.

The adapter reads Qlib workflow configuration as record-layer metadata and
writes candidate QST `.gkr.yaml` plus deterministic coverage JSON. It is
partial and not lossless. It does not import Qlib, run qrun, train models, run
inference, execute backtests, connect to a broker, connect to an exchange, or
provide live execution.

## Documents

- [Qlib Adapter Boundary](QLIB_ADAPTER_BOUNDARY.md)
- [Qlib Adapter Guide](QLIB_ADAPTER_GUIDE.md)
