import { useState, useEffect, useCallback } from "react";
import { live, orders as ordersApi, positions as positionsApi, risk, monitor, signals } from "../api/client";
import { useWebSocket } from "./useWebSocket";
import type {
  LiveMetrics,
  EquityPoint,
  Position,
  Order,
  TradeSummary,
  RiskStatus,
} from "../types/api";
import type { WSRiskData, WSTickData } from "../types/ws";
import type { SignalEntry } from "../pages/monitor/SignalsTab";

export interface LiveDataState {
  metrics: LiveMetrics | null;
  equity: EquityPoint[];
  positionsList: Position[];
  ordersList: Order[];
  liveTrades: TradeSummary[];
  riskStatus: RiskStatus | null;
  wsRisk: WSRiskData | null;
  regimeHistory: { timestamp: string; regime: number }[];
  signalEntries: SignalEntry[];
  error: string | null;
  lastFetched: number;
}

export interface LiveDataComputed {
  halted: boolean;
  equityVal: number;
  balanceVal: number;
  dailyPnl: number;
  regime: number;
  drawdownUsed: number;
  dailyLossUsed: number;
  dailyLimitPct: number;
  maxDdPct: number;
  winRate: number;
  sharpe: number;
  profitFactor: number;
  totalTrades: number;
}

const INITIAL_STATE: LiveDataState = {
  metrics: null,
  equity: [],
  positionsList: [],
  ordersList: [],
  liveTrades: [],
  riskStatus: null,
  wsRisk: null,
  regimeHistory: [],
  signalEntries: [],
  error: null,
  lastFetched: 0,
};

/**
 * Shared hook for live trading + risk + signal data.
 * Subscribes to WebSocket `risk` + `ticks` channels and polls REST every `intervalMs`.
 * Used by MonitorPage and can be consumed by EmergencyPage.
 */
export function useLiveData(intervalMs = 10000) {
  const [state, setState] = useState<LiveDataState>(INITIAL_STATE);

  const setField = useCallback(
    <K extends keyof LiveDataState>(key: K, value: LiveDataState[K]) => {
      setState((prev) => ({ ...prev, [key]: value }));
    },
    [],
  );

  useWebSocket("risk", {
    onMessage: (data) => setField("wsRisk", data as WSRiskData),
  });

  useWebSocket("ticks", {
    onMessage: () => {
      /* ticks feed — consumed by chart hooks */
    },
  });

  const fetchAll = useCallback(async () => {
    try {
      const [m, e, p, o, t, r, rh, sig] = await Promise.all([
        live.metrics(),
        live.equity("90d"),
        positionsApi.list().catch(() => ({ positions: [] as Position[] })),
        ordersApi.list().catch(() => ({ orders: [] as Order[] })),
        live.trades().catch(() => ({ trades: [] as TradeSummary[] })),
        risk.status().catch(() => null as RiskStatus | null),
        monitor.regimeHistory().catch(() => ({ history: [] })),
        signals.list().catch(() => ({ signals: [] as SignalEntry[] })),
      ]);
      setState((prev) => ({
        ...prev,
        metrics: m,
        equity: Array.isArray(e) ? e : [],
        positionsList: Array.isArray(p?.positions) ? p.positions : [],
        ordersList: Array.isArray(o?.orders) ? o.orders : [],
        liveTrades: Array.isArray(t?.trades) ? t.trades : [],
        riskStatus: r,
        regimeHistory: rh?.history ?? [],
        signalEntries: sig?.signals ?? [],
        error: null,
        lastFetched: Date.now(),
      }));
    } catch (err) {
      setField(
        "error",
        err instanceof Error ? err.message : "Failed to load",
      );
    }
  }, [setField]);

  useEffect(() => {
    fetchAll();
    const interval = setInterval(fetchAll, intervalMs);
    return () => clearInterval(interval);
  }, [fetchAll, intervalMs]);

  const {
    metrics,
    wsRisk,
    riskStatus,
  } = state;

  const computed: LiveDataComputed = {
    halted: wsRisk?.halted ?? riskStatus?.halted ?? false,
    equityVal: wsRisk?.equity ?? riskStatus?.equity ?? metrics?.equity ?? 0,
    balanceVal: wsRisk?.balance ?? riskStatus?.balance ?? metrics?.balance ?? 0,
    dailyPnl: wsRisk?.daily_pnl_pct ?? riskStatus?.daily_pnl_pct ?? metrics?.daily_pnl_pct ?? 0,
    regime: wsRisk?.regime ?? -1,
    drawdownUsed: wsRisk?.drawdown_used ?? riskStatus?.drawdown_used ?? 0,
    dailyLossUsed: wsRisk?.daily_loss_used ?? riskStatus?.daily_loss_used ?? 0,
    dailyLimitPct: wsRisk?.daily_limit_pct ?? riskStatus?.daily_limit_pct ?? 5,
    maxDdPct: wsRisk?.max_dd_pct ?? riskStatus?.max_dd_pct ?? 10,
    winRate: metrics?.win_rate ?? 0,
    sharpe: metrics?.sharpe ?? 0,
    profitFactor: metrics?.profit_factor ?? 0,
    totalTrades: metrics?.num_trades ?? 0,
  };

  return {
    ...state,
    computed,
    refresh: fetchAll,
  };
}
