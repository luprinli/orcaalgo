import { create } from 'zustand'
import type { ComboResult, MatrixResultsResponse } from '../types/api'

// Telemetry mirrors the server's matrix summary (execution-framework §5.2).
export interface MatrixTelemetry {
  total: number
  completed: number
  running: number
  failed: number
  percent: number
  throughputPerMin: number
  etaSeconds: number
  bestSharpe: number
  bestStrategy: string
  bestSymbol: string
  current: { strategy: string; symbol: string; timeframe: string } | null
  chunkIndex: number
  chunkTotal: number
  skipped: number
  phase: string
}

export type MatrixStatus = 'idle' | 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'

const comboKey = (r: ComboResult) => `${r.strategy_id}|${r.symbol}|${r.timeframe}`

const emptyTelemetry = (): MatrixTelemetry => ({
  total: 0, completed: 0, running: 0, failed: 0, percent: 0,
  throughputPerMin: 0, etaSeconds: 0, bestSharpe: 0, bestStrategy: '', bestSymbol: '',
  current: null, chunkIndex: 0, chunkTotal: 0, skipped: 0, phase: '',
})

interface MatrixState {
  batchId: string | null
  status: MatrixStatus
  seq: number
  byKey: Record<string, ComboResult>
  order: string[]
  telemetry: MatrixTelemetry

  begin: (batchId: string, total: number) => void
  applyDelta: (resp: MatrixResultsResponse) => void
  setStatus: (status: MatrixStatus) => void
  reset: () => void
  results: () => ComboResult[]
}

/**
 * matrixStore holds streaming matrix state, keyed by combo, so results are
 * upserted (never full-array-replaced) and each UI slice subscribes only to what
 * it needs (execution-framework plan §11.2/§11.4).
 */
export const useMatrixStore = create<MatrixState>((set, get) => ({
  batchId: null,
  status: 'idle',
  seq: 0,
  byKey: {},
  order: [],
  telemetry: emptyTelemetry(),

  begin: (batchId, total) =>
    set({
      batchId,
      status: 'running',
      seq: 0,
      byKey: {},
      order: [],
      telemetry: { ...emptyTelemetry(), total },
    }),

  applyDelta: (resp) =>
    set((state) => {
      const byKey = { ...state.byKey }
      const order = state.order.slice()
      for (const r of resp.results ?? []) {
        const k = comboKey(r)
        if (!(k in byKey)) order.push(k)
        byKey[k] = r
      }
      const s = resp.summary ?? ({} as MatrixResultsResponse['summary'])
      const nextSeq = resp.seq ?? s.seq ?? state.seq
      const rawStatus = (s.status ?? state.status) as MatrixStatus
      const status: MatrixStatus = s.cancelled ? 'cancelled' : rawStatus
      return {
        byKey,
        order,
        seq: nextSeq,
        status,
        telemetry: {
          total: s.total_combos ?? state.telemetry.total,
          completed: s.completed ?? order.length,
          running: s.running ?? 0,
          failed: s.failed ?? state.telemetry.failed,
          percent: s.percent ?? state.telemetry.percent,
          throughputPerMin: s.throughput_per_min ?? 0,
          etaSeconds: s.eta_seconds ?? 0,
          bestSharpe: s.best_sharpe ?? state.telemetry.bestSharpe,
          bestStrategy: s.best_strategy ?? state.telemetry.bestStrategy,
          bestSymbol: s.best_symbol ?? state.telemetry.bestSymbol,
          current: s.current ?? null,
          chunkIndex: s.chunk?.index ?? state.telemetry.chunkIndex,
          chunkTotal: s.chunk?.total ?? state.telemetry.chunkTotal,
          skipped: s.skipped ?? state.telemetry.skipped,
          phase: s.phase ?? state.telemetry.phase,
        },
      }
    }),

  setStatus: (status) => set({ status }),
  reset: () => set({ batchId: null, status: 'idle', seq: 0, byKey: {}, order: [], telemetry: emptyTelemetry() }),
  results: () => {
    const { byKey, order } = get()
    return order.map((k) => byKey[k])
  },
}))
