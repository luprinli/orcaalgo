import { request } from './client'

export interface OptimizeConfig {
  strategy_id: string
  symbols: string[]
  objective?: string
  max_combinations?: number
  train_years?: number
  test_years?: number
  step_months?: number
  capital?: number
  parameters?: Record<string, { min: number; max: number; step: number }>
}

export interface OptimizationStatus {
  run_id: string
  status: 'queued' | 'running' | 'completed' | 'failed'
  progress?: number
  elapsed_seconds?: number
  best_metric?: number
  total_trials?: number
  error?: string
}

export interface OptimizationResult {
  run_id: string
  best_params: Record<string, number>
  best_metric: number
  avg_oos_sharpe?: number
  windows_passed?: number
  total_windows?: number
  trials: Array<{
    params: Record<string, number>
    metric: number
  }>
}

export interface OptimizationRun {
  id: string
  strategy_id: string
  status: string
  created_at: string
  best_metric?: number
}

export function startOptimization(config: OptimizeConfig) {
  return request<{ run_id: string }>('POST', '/api/v1/optimize', config)
}

export function getOptimizationStatus(runId: string) {
  return request<OptimizationStatus>('GET', `/api/v1/optimize/${runId}/status`)
}

export function getOptimizationResults(runId: string) {
  return request<OptimizationResult>('GET', `/api/v1/optimize/${runId}/results`)
}

export function submitOptimizationRun(config: OptimizeConfig) {
  return request<{ run_id: string }>('POST', '/api/v1/optimize/run', config)
}

export function listOptimizationRuns() {
  return request<OptimizationRun[]>('GET', '/api/v1/optimize/runs')
}
