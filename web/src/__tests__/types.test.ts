import { describe, it, expect } from 'vitest'
import type {
  Strategy,
  BacktestMetrics,
  RiskStatus,
  CalibrationReportResponse,
  AttributionReportResponse,
  DataValidateResponse,
  PreflightResponse,
  DeployStrategyResponse,
} from '../types/api'

describe('type compatibility — Strategy', () => {
  it('validates minimal strategy shape', () => {
    const s: Strategy = {
      id: 'strat-1',
      name: 'Test Strategy',
      type: 'ma_crossover',
      enabled: false,
    }
    expect(s.id).toBe('strat-1')
    expect(s.enabled).toBe(false)
  })

  it('accepts optional parameters', () => {
    const s: Strategy = {
      id: 'strat-2',
      name: 'Param Strategy',
      type: 'grid_trading',
      enabled: true,
      parameters: { lookback: 20, threshold: 2.5 },
    }
    expect(s.parameters?.lookback).toBe(20)
  })
})

describe('type compatibility — BacktestMetrics', () => {
  it('validates full metrics shape', () => {
    const m: BacktestMetrics = {
      sharpe_ratio: 2.1,
      sortino_ratio: 3.5,
      max_drawdown_pct: 12.5,
      win_rate_pct: 65.0,
      profit_factor: 1.8,
      total_return_pct: 45.2,
      num_trades: 150,
      trading_volume: 500000,
      strategy_name: 'ma_crossover',
      pass_probability: 78.0,
      calmar: 1.2,
      var_95: 0.05,
      cvar_95: 0.08,
      ulcer_index: 0.03,
      cagr: 0.22,
      balance: 100000,
      equity: 145200,
    }
    expect(m.sharpe_ratio).toBeGreaterThan(0)
  })
})

describe('type compatibility — Preflight & Deploy', () => {
  it('validates preflight response', () => {
    const r: PreflightResponse = {
      passed: true,
      passed_count: 5,
      warned_count: 1,
      failed_count: 0,
      checks: [
        { name: 'config_exists', status: 'pass', message: 'Found' },
        { name: 'env_ORCA_DB_URL', status: 'warn', message: 'Not set' },
      ],
    }
    expect(r.passed).toBe(true)
    expect(r.checks).toHaveLength(2)
  })

  it('validates deploy response', () => {
    const r: DeployStrategyResponse = {
      deployed: true,
      strategy_name: 'ma_crossover',
      backtest_id: 'bt-123',
      account_id: 'paper',
      capital_allocation_pct: 25,
    }
    expect(r.deployed).toBe(true)
    expect(r.account_id).toBe('paper')
  })
})

describe('type compatibility — Calibration & Attribution', () => {
  it('validates calibration report', () => {
    const r: CalibrationReportResponse = {
      overall: {
        name: 'overall',
        n: 200,
        brier: 0.12,
        reliability: 0.003,
        resolution: 0.08,
        uncertainty: 0.097,
        bin_stats: [
          { bin_start: 0, bin_end: 0.1, count: 20, mean_prediction: 0.05, hit_rate: 0.06 },
          { bin_start: 0.1, bin_end: 0.2, count: 25, mean_prediction: 0.15, hit_rate: 0.14 },
        ],
        needs_calibration: false,
      },
      segments: {},
      generated_at: '2026-06-30T00:00:00Z',
    }
    expect(r.overall.brier).toBeLessThan(0.25)
    expect(r.overall.bin_stats).toHaveLength(2)
  })

  it('validates attribution report', () => {
    const r: AttributionReportResponse = {
      overall: {
        n: 150,
        wins: 90,
        hit_rate: 0.6,
        hit_rate_ci_low: 0.52,
        hit_rate_ci_high: 0.68,
        total_pnl: 25000,
        total_cost: 1200,
        roi: 20.83,
      },
      by_side: {
        BUY: { n: 80, wins: 50, hit_rate: 0.625, hit_rate_ci_low: 0.51, hit_rate_ci_high: 0.73, total_pnl: 15000, total_cost: 600, roi: 25 },
        SELL: { n: 70, wins: 40, hit_rate: 0.571, hit_rate_ci_low: 0.45, hit_rate_ci_high: 0.69, total_pnl: 10000, total_cost: 600, roi: 16.67 },
      },
      by_price_bucket: {},
      by_edge_bucket: {},
      generated_at: '2026-06-30T00:00:00Z',
    }
    expect(r.overall.n).toBe(150)
    expect(r.by_side.BUY).toBeDefined()
  })
})

describe('type compatibility — Risk Status', () => {
  it('validates risk status shape', () => {
    const r: RiskStatus = {
      halted: false,
      reason: '',
      last_trigger: '2026-06-30T00:00:00Z',
      balance: 100000,
      equity: 101500,
      daily_pnl_pct: 1.5,
      daily_loss_used: 30,
      drawdown_used: 15,
      daily_limit_pct: 5,
      max_dd_pct: 10,
      consistency_multiplier: 1.0,
    }
    expect(r.halted).toBe(false)
    expect(r.daily_pnl_pct).toBeGreaterThan(0)
  })
})

describe('type compatibility — Data Validate', () => {
  it('validates data quality response', () => {
    const r: DataValidateResponse = {
      passed: true,
      passed_count: 5,
      warned_count: 2,
      failed_count: 0,
      checks: [
        { name: 'data_exists', status: 'pass', message: 'Found data', symbol: 'AAPL' },
        { name: 'gap_detected', status: 'warn', message: 'Up to 3 zero closes', symbol: 'MSFT' },
      ],
    }
    expect(r.passed).toBe(true)
    expect(r.checks[0].symbol).toBe('AAPL')
  })
})
