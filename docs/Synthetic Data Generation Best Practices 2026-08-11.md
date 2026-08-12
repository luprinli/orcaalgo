# OrcaAlgo — Synthetic Data Generation: Best Practices & Resampling Strategy

**Date:** 2026-08-11  
**Context:** Post-matrix-re-run analysis revealed 100% of combos with candle data (1,894/1,894) have bar resolution mismatches — `DeclaredBPD` vs `EffectiveBPD` diverge by 3.6× to 30×. All strategies produce uniformly negative returns due to corrupted indicator calculations from mis-labeled timeframe data.  
**Question:** Should we generate higher-timeframe candles from a single fine-resolution source (e.g., 5-minute data) via resampling? What are the best practices and alternatives?

---

## 1. Executive Recommendation

**YES — start from 5-minute data and resample to higher timeframes.** This is the industry-standard approach for systematic trading backtesting. It ensures deterministic reproducibility, eliminates timeframe misalignment, and provides a single source of truth for data quality validation.

**Implementation:** Python resampling pipeline (`orca build-candles`) using pandas OHLC aggregation, stored in the existing TimescaleDB `candles` hypertable with correct `timeframe` labels.

---

## 2. Current Infrastructure Analysis

### 2.1 What Exists

| Component | Status | Details |
|-----------|--------|---------|
| Candles table | ✅ Present | TimescaleDB hypertable with `(symbol_id, timeframe, time)` unique constraint |
| Data fetchers | ✅ Present | Yahoo, Tiingo, Stooq, Alpaca WS — each passes timeframe to external API |
| Python simulation | ✅ Present | `orca simulate generate-1m` creates synthetic 1-minute candles |
| Empty collector/ | ✅ Available | `internal/ingest/collector/` — 0 files, ready for resampling logic |
| Resampling logic | ❌ **MISSING** | Zero files anywhere in Go or Python implement OHLC aggregation |

### 2.2 What's Broken

The `candles` table has data with incorrect `timeframe` labels. Evidence from the diagnostic columns:

| Declared Timeframe | Declared BPD | Effective BPD | Actual Resolution |
|-------------------|-------------|---------------|-------------------|
| 1h | 6.5 | 23.5 | ~15 minutes |
| 4h | 1.6 | 6.0 | ~1 hour |
| 1d | 1.0 | 30.3 | ~30 minutes |

The engine's `LoadCandlesByTimeframe` correctly filters by `c.timeframe = $4`. It retrieves whatever the DB has. The data loaded into the DB was simply labeled with wrong timeframe values.

---

## 3. Best Practice: 5-Minute Base Resampling

### 3.1 Why This Is Industry Standard

**Consistency across timeframes.** When 1h bars are derived from the same 5-minute source as 1d bars, the 1d bar's close price *equals* the close of its last 5-minute constituent. No reconciliation gaps. This property is essential for strategies that use multi-timeframe analysis (e.g., weekly trend filter on daily signals).

**Deterministic reproduction.** Given the same 5-minute source data, the same higher-timeframe bars are always produced. This makes backtest results auditable — critical for institutional compliance and the `orca preflight` checklist.

**Single data quality surface.** Only one dataset needs cleansing and validation (the 5-minute source). All derived timeframes inherit the source's quality. If a gap exists in the 5-minute data, it propagates to all timeframes in a predictable way. If it's fixed, all timeframes are fixed simultaneously.

**Alignment integrity.** Bar boundaries naturally nest: a 1h bar at 10:00 always ends at the same timestamp as the 09:55–10:00 5-minute bar. No boundary misalignment can occur. This is impossible to guarantee with independently-sourced timeframe data from different providers.

### 3.2 The Resampling Formula

Standard OHLC aggregation from finer to coarser bars:

```
Open   = first(Open) of constituent bars ordered by time
High   = max(High) of constituent bars
Low    = min(Low) of constituent bars
Close  = last(Close) of constituent bars ordered by time
Volume = sum(Volume) of constituent bars
```

For a 5m → 1h resampling with `time_bucket('1 hour', time)`:

```python
df.resample('1h', on='time').agg({
    'open':   'first',
    'high':   'max',
    'low':    'min',
    'close':  'last',
    'volume': 'sum',
})
```

**Validation invariant:** For any day, the 1d bar's High must equal `max(high)` of its 78 constituent 5-minute bars during regular hours (09:30–16:00), and the 1d bar's Volume must equal `sum(volume)` of those 78 bars.

### 3.3 Timeframe Hierarchy

The canonical hierarchy (using regular trading hours for equities):

| Timeframe | Bars/Day (RTH) | Constituents | 
|-----------|----------------|-------------|
| 5m | 78 | (base, no constituents) |
| 15m | 26 | 3 × 5m bars |
| 30m | 13 | 6 × 5m bars |
| 1h | 6.5 | 12 × 5m bars (last bar = half bar at close) |
| 4h | 1.6 | 48 × 5m bars |
| 1d | 1 | 78 × 5m bars |

For 24-hour markets (FX, crypto), `barsPerDay = 1440 / resolution_minutes`.

---

## 4. Alternatives Considered

### 4.1 Multi-Source with Reconciliation

**Ingest 5m, 1h, and 1d independently from exchange APIs.** Validate: the 1d bar's close from the exchange should equal the close of the last 5m bar ± 0.1%.

| Aspect | Assessment |
|--------|------------|
| Microstructure fidelity | ✅ Best — each timeframe has authentic exchange-provided prices |
| Implementation complexity | ❌ Requires managing 3+ data sources, handling schema differences, reconciling discrepancies |
| Adjustment consistency | ❌ Exchange daily bars may use different split/dividend adjustment methodology than intraday bars |
| Audit trail | ❌ Harder to explain why 1d close ≠ aggregated 5m close |
| Storage | ❌ 3× storage for redundant data |

**Verdict:** Over-engineered for a research backtesting framework. The marginal benefit of "authentic" daily closes vs. aggregated closes is negligible for strategy evaluation purposes.

### 4.2 TimescaleDB time_bucket (On-The-Fly)

**Store only 5m data. Use SQL time_bucket() to aggregate on query.**

```sql
SELECT time_bucket('1 hour', time) AS bucket,
       first(open_raw, time) AS open_raw,
       max(high_raw) AS high_raw,
       min(low_raw) AS low_raw,
       last(close_raw, time) AS close_raw,
       sum(volume) AS volume
FROM candles
WHERE symbol_id = $1 AND timeframe = '5m'
  AND time >= $2 AND time <= $3
GROUP BY bucket
ORDER BY bucket
```

| Aspect | Assessment |
|--------|------------|
| Storage | ✅ Zero duplication |
| Consistency | ✅ Always derived from source — impossible to desync |
| Query performance | ❌ 10–50× slower than pre-computed bars for long backtests |
| Compression | ❌ TimescaleDB compression doesn't work on aggregated subqueries |
| Go integration | ❌ Requires rewriting `LoadCandlesByTimeframe` to use time_bucket queries instead of simple SELECT |

**Verdict:** Feasible but performance-degrading. The 3,990-combo matrix sweep (14 minutes with pre-computed bars) would take 2–10 hours with on-the-fly aggregation.

### 4.3 Go-Based Aggregation (collector/ package)

**Read 5m bars into memory, aggregate in Go, insert results.**

| Aspect | Assessment |
|--------|------------|
| Performance | ✅ Fast, in-process |
| Go ecosystem | ❌ Go has no built-in OHLC resampling — would need to implement from scratch |
| Code size | ❌ Would require 200+ lines of aggregation logic + edge cases (partial bars, gaps, weekends) |
| Python vs Go | ❌ Python pandas has 15+ years of battle-tested resampling; Go would be a reimplementation |

**Verdict:** Reject. The Python ecosystem solved this problem 15 years ago. A Go implementation would introduce unnecessary risk and maintenance burden.

### 4.4 Python Batch Pipeline (Recommended)

**Run `orca build-candles --source-timeframe 5m` which reads 5m bars, resamples to all higher timeframes using pandas, validates invariants, and upserts into TimescaleDB.**

| Aspect | Assessment |
|--------|------------|
| Performance | ✅ Pandas can resample 10 years of 5m equity data (78 bars/day × 252 days × 10 years ≈ 200K rows) to 5 timeframes in <5 seconds |
| Validation | ✅ Easy to assert invariants: 1h.High == max(12×5m.High) for each bucket |
| Deterministic | ✅ Same input always produces same output |
| Existing infra | ✅ Fits into existing `orca simulate` CLI + DB connection pattern |
| Maintenance | ✅ 50 lines of Python vs. 200+ lines of Go |

---

## 5. Implementation Plan

### 5.1 Architecture

```
5m Source Data (Alpaca/Yahoo)
        │
        ▼
┌─────────────────────────────────┐
│  orca build-candles             │
│  --source-timeframe 5m          │
│  --targets 15m,30m,1h,4h,1d    │
│                                 │
│  1. Read 5m bars from DB        │
│  2. Resample per timeframe      │
│  3. Validate invariants         │
│  4. Upsert into candles table   │
└─────────────────────────────────┘
        │
        ▼
TimescaleDB candles hypertable
(symbol_id, timeframe, time)
with correct timeframe labels
```

### 5.2 Resampling Module (`orca/data/resample.py`)

```python
def resample_ohlc(df: pd.DataFrame, timeframe: str) -> pd.DataFrame:
    """Resample 5-minute OHLCV bars to a higher timeframe.
    
    Args:
        df: DataFrame with columns [time, open, high, low, close, volume]
        timeframe: Pandas offset string ('15min', '1h', '4h', '1d')
    
    Returns:
        DataFrame with aggregated OHLCV bars at requested timeframe
    """
    return df.resample(timeframe, on='time').agg({
        'open':   'first',
        'high':   'max',
        'low':    'min', 
        'close':  'last',
        'volume': 'sum',
    }).dropna()
```

### 5.3 Validation Module (`orca/data/validate_resample.py`)

For each higher timeframe bar, verify against its 5m constituents:

```python
def validate_resampling(
    source_5m: pd.DataFrame,
    derived: pd.DataFrame,
    timeframe: str
) -> list[str]:
    """Return list of validation errors found."""
    errors = []
    
    for _, bar in derived.iterrows():
        # Get constituent 5m bars within this bar's window
        window_end = bar.name
        window_start = window_end - pd.Timedelta(timeframe)
        constituents = source_5m[
            (source_5m['time'] >= window_start) &
            (source_5m['time'] <= window_end)
        ]
        
        if len(constituents) == 0:
            continue
            
        # Validate OHLCV
        if abs(bar['open'] - constituents.iloc[0]['open']) > 1e-8:
            errors.append(f"{bar.name}: open mismatch")
        if abs(bar['high'] - constituents['high'].max()) > 1e-8:
            errors.append(f"{bar.name}: high mismatch")
        if abs(bar['low'] - constituents['low'].min()) > 1e-8:
            errors.append(f"{bar.name}: low mismatch")
        if abs(bar['close'] - constituents.iloc[-1]['close']) > 1e-8:
            errors.append(f"{bar.name}: close mismatch")
        if abs(bar['volume'] - constituents['volume'].sum()) > 1e-8:
            errors.append(f"{bar.name}: volume mismatch")
    
    return errors
```

### 5.4 CLI Entry Point (`orca/cli.py`)

```python
@app.command("build-candles")
def build_candles(
    symbols: list[str] = typer.Option(None, "--symbols"),
    source_timeframe: str = typer.Option("5m", "--source-timeframe"),
    targets: list[str] = typer.Option(["15m", "30m", "1h", "4h", "1d"], "--targets"),
    start: str = typer.Option(None),
    end: str = typer.Option(None),
    validate: bool = typer.Option(True, "--validate/--no-validate"),
):
    """Build higher-timeframe candles from a fine-resolution source.
    
    Resamples 5-minute OHLCV data to 15m, 30m, 1h, 4h, and 1d timeframes
    using standard OHLC aggregation, validates invariants, and upserts
    results into the candles hypertable.
    """
    from orca.data.resample import resample_ohlc
    from orca.data.validate_resample import validate_resampling
    
    for symbol in symbols:
        # 1. Load source 5m data
        df = load_candles(symbol, source_timeframe, start, end)
        
        # 2. Resample to each target timeframe
        for tf in targets:
            derived = resample_ohlc(df, tf)
            
            if validate:
                errors = validate_resampling(df, derived, tf)
                if errors:
                    typer.echo(f"WARNING: {len(errors)} validation errors for {symbol} {tf}")
                    for e in errors[:5]:
                        typer.echo(f"  {e}")
            
            # 3. Upsert into TimescaleDB
            upsert_candles(symbol, tf, derived)
    
    typer.echo(f"Done: {len(symbols)} symbols × {len(targets)} timeframes")
```

### 5.5 DB Upsert Strategy

Use `ON CONFLICT (symbol_id, timeframe, time) DO UPDATE` to support incremental re-builds:

```sql
INSERT INTO candles (symbol_id, timeframe, time, open_raw, high_raw, low_raw, close_raw, volume, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'resampled')
ON CONFLICT (symbol_id, timeframe, time)
DO UPDATE SET
    open_raw = EXCLUDED.open_raw,
    high_raw = EXCLUDED.high_raw,
    low_raw = EXCLUDED.low_raw,
    close_raw = EXCLUDED.close_raw,
    volume = EXCLUDED.volume,
    source = EXCLUDED.source
```

### 5.6 Implementation Effort

| Step | Est. Effort | Dependencies |
|------|------------|-------------|
| 1. `orca/data/resample.py` — resampling core | 2h | pandas |
| 2. `orca/data/validate_resample.py` — validation | 2h | pandas |
| 3. DB integration (upsert logic) | 2h | Existing DB connection in `orca/db/` |
| 4. CLI entry point | 1h | Typer |
| 5. Integration test (1 symbol, all timeframes) | 1h | Test DB |
| 6. Re-run matrix backtest | 30m | Completed pipeline |
| **Total** | **8.5h** | |

---

## 6. Data Source Selection for 5-Minute Base

### 6.1 Candidate Providers

| Provider | 5m Data | Cost | Rate Limits | Best For |
|----------|---------|------|-------------|----------|
| Alpaca Markets | ✅ Free tier | $0 | 200 req/min | US equities |
| Polygon.io | ✅ | $29/mo | Unlimited (paid) | US equities + options |
| Yahoo Finance | ✅ (v8 API) | $0 | ~2000 req/hr (unofficial) | Equities, indices, FX, crypto |
| Tiingo | ❌ Daily only | $0–$10/mo | 1000/day | End-of-day |
| Stooq | ✅ (CSV) | $0 | Unlimited (local files) | Equities, FX, indices |

### 6.2 Recommended: Alpaca + Yahoo Fallback

- **Primary:** Alpaca Markets (US equities — 5m bars for 3,000+ symbols)  
- **Fallback:** Yahoo Finance (indices, FX, crypto, commodities)
- **Rationale:** Both are free, well-documented, and already integrated via `internal/ingest/`

### 6.3 Pre-Existing Data

The `data/5 min/` directory contains 2,395 files (~2.25 GB). These may already be correctly-labeled 5-minute bar CSVs from a previous Stooq import. If so, they can serve as the source for the first resampling run.

---

## 7. Post-Resampling Validation Checklist

Before trusting resampled data for backtesting:

| # | Check | Method | Pass Criteria |
|---|-------|--------|---------------|
| 1 | Bar count consistency | For each symbol, verify `count(1d bars) >= count(4h bars) / 6.5` | Within ±1 bar per year |
| 2 | OHLCV invariants | For each 1d bar, validate against 78 × 5m constituents | Zero violations |
| 3 | No gaps > 1 bar | Check for consecutive bars > 2× expected interval | Zero gaps (or flag holidays) |
| 4 | Volume monotonicity | Sum(volume per day) / bars per day ≈ constant | Within ±50% (volatility varies) |
| 5 | Close consistency | 1d.Close == last(5m.Close) for that day | Zero violations |
| 6 | DeclaredBPD match | Re-run matrix sweep; verify `DeclaredBPD ≈ EffectiveBPD` | Within ±5% |
| 7 | Strategy profitability | Re-run matrix sweep; verify non-negative Sharpe for at least some combos | >0 combos with Sharpe > 0 |
| 8 | Cross-timeframe signal consistency | Run same strategy on 5m-derived 1h vs 5m-derived 1d | Sensible performance differences |

---

## 8. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| 5m source data has gaps (holidays, outages) | High | Medium | Gap detection + interpolation warning; flag affected timeframes |
| Corporate action adjustment mismatch | Medium | Low | Use split-adjusted prices from provider; validate against known split dates |
| Overnight/pre-market bars included in 1d aggregation | High | Medium | Configurable session filter (RTH only vs. extended hours) |
| Last 1h bar of the day has <12 constituents (partial bar) | Medium | Low | Handle partial bars gracefully; label as "partial" in warnings |
| Tick-level microstructure lost at 5m resolution | High | Medium | Acceptable for strategies that don't use tick data; document limitation |
| DB upsert performance with millions of rows | Medium | Medium | Batch INSERT with COPY protocol; TimescaleDB handles this well |

---

## Appendix A: Comparison with Other Frameworks

| Framework | Approach | Source |
|-----------|----------|--------|
| QuantConnect | Multi-source with resampling | Coarse data downloaded first; fine data resampled from coarse via fill-forward |
| Backtrader | Resampling on-the-fly | `cerebro.resampledata()` with timeframe compression |
| Zipline (Quantopian) | Daily + minute bundles | Separate ingestion pipelines; no intra-to-daily resampling |
| VectorBT | Single-resolution by default | `vbt.BinanceData.download()` at one resolution; user must resample manually |
| Blueshift | Multi-source | Cloud platform; handles resampling internally |

**OrcaAlgo's proposed approach** (5m base with batch resampling) aligns most closely with QuantConnect's methodology but with stricter validation invariants.

---

**Report prepared by:** Senior Quantitative Analyst — Data Infrastructure Best Practices  
**Next action:** Implement `orca build-candles` pipeline with 5-minute base resampling  
**Estimated effort:** 8.5 hours implementation + 2 hours validation matrix re-run  
**Expected outcome:** All 6 timeframes producing correctly-labeled candle data; matrix sweep showing non-zero returns for profitable strategies
