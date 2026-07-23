export type OptimizationConfig = {
  strategy_id: string;
  objective: 'sharpe' | 'sortino' | 'profit_factor' | 'win_rate' | 'min_drawdown' | 'composite';
  max_combinations: number;
  train_years: number;
  test_years: number;
  step_months: number;
  symbols: string[];
  capital: number;
  constraints: Record<string, { min: number; max: number; step: number }>;
};

export type OptimizationRun = {
  run_id: string;
  status: string;
};

export type OptimizationStatus = {
  status: string;
  progress_pct: number;
  current_stage?: string;
  elapsed_s?: number;
  best_metric_so_far?: number;
};

export type OptimizationResult = {
  best_params: Record<string, number>;
  best_metric: number;
  total_trials: number;
  avg_oos_sharpe: number;
  windows_passed: number;
  windows_total: number;
  profit_factor: number;
};

export async function startOptimization(config: OptimizationConfig): Promise<OptimizationRun> {
  const res = await fetch('/api/v1/optimize', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  return res.json();
}

export async function getOptimizationStatus(runId: string): Promise<OptimizationStatus> {
  const res = await fetch(`/api/v1/optimize/${runId}/status`);
  if (!res.ok) return { status: 'not_found', progress_pct: 0 };
  return res.json();
}

export async function getOptimizationResults(runId: string): Promise<OptimizationResult> {
  const res = await fetch(`/api/v1/optimize/${runId}/results`);
  return res.json();
}

export async function submitOptimizationRun(config: OptimizationConfig): Promise<OptimizationRun> {
  const res = await fetch('/api/v1/optimize/run', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  });
  return res.json();
}

export async function listOptimizationRuns(): Promise<{ runs: unknown[] }> {
  const res = await fetch('/api/v1/optimize/runs');
  return res.json();
}
