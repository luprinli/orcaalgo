import { getRequestHeaders, markRequestComplete } from './middleware'
import type {
  LoginRequest, LoginResponse, Strategy, CreateStrategyRequest, UpdateStrategyRequest,
  DeployStrategyRequest, DeployStrategyResponse, PreflightResponse,
  StrategyValidationRequest, StrategyValidationResponse,
  BacktestRequest, BacktestResponse, BacktestMetrics,
  EquityPoint, TradeSummary, DailyReturn, MonthlyReturn, RollingMetric,
  RegimeStat, OptimizationFootprint, WalkForwardResponse, LiveComparisonResponse,
  RiskStatus, PlaceOrderRequest, Order, Position, Account, CreateAccountRequest,
  CandleResponse, LiveMetrics, BacktestHistoryEntry, AppSettings,
  PropFirmProfile, PropFirmState, AuditLogEntry, ErrorLogEntry,
  CalibrationReportResponse, AttributionReportResponse, DataValidateResponse,
  MatrixResultsResponse, IndicatorSpec, IndicatorComputeRequest, IndicatorComputeResponse,
  SimulateGenerateRequest, SimulateGenerateResponse,
  SimulateCalibrateRequest, SimulateCalibrateResponse,
  SimulateValidateResponse,
  SystemHealth,
} from '../types/api'

export interface Symbol {
  id: string; ticker: string; name?: string; provider_id?: string; enabled?: boolean
}
export interface Provider {
  id: string; name: string; type: string; status?: string
}
export interface Credential {
  id: string; name: string; provider_type: string; api_key?: string; created_at?: string
}

interface AuthData {
  token: string
  refresh: string
  roles: string[]
}

function getAuth(): AuthData | null {
  const raw = localStorage.getItem('orca_auth')
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw)
    if (typeof parsed === 'string') return { token: parsed, refresh: '', roles: [] }
    return parsed
  } catch {
    if (raw.startsWith('{') || raw.startsWith('[')) return null
    return { token: raw, refresh: '', roles: [] }
  }
}

function getToken(): string | null {
  return getAuth()?.token ?? null
}

function setAuth(auth: AuthData) {
  localStorage.setItem('orca_auth', JSON.stringify(auth))
}

function clearAuth() {
  localStorage.removeItem('orca_auth')
}

let refreshPromise: Promise<boolean> | null = null

async function tryRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    const auth = getAuth()
    if (!auth?.token) return false
    try {
      const res = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ access_token: auth.token }),
      })
      if (!res.ok) return false
      const data = await res.json()
      setAuth({ token: data.access_token, refresh: data.refresh_token, roles: auth.roles })
      return true
    } catch {
      return false
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  extraHeaders?: Record<string, string>,
): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...getRequestHeaders(token),
  }
  if (extraHeaders) {
    Object.assign(headers, extraHeaders)
  }

  const res = await fetch(path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  const handleResponse = async () => {
    if (res.status === 401 && token) {
      const refreshed = await tryRefresh()
      if (refreshed) {
        const newToken = getToken()
        if (newToken) headers['Authorization'] = `Bearer ${newToken}`
        const retryRes = await fetch(path, {
          method,
          headers,
          body: body ? JSON.stringify(body) : undefined,
        })
        if (retryRes.ok) {
          return retryRes.json() as Promise<T>
        }
      }
      clearAuth()
      window.location.href = '/login'
      throw new Error('Unauthorized')
    }

    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(err.error || `HTTP ${res.status}`)
    }

    return res.json() as Promise<T>
  }

  return handleResponse().finally(() => markRequestComplete(res.status))
}

export { getToken, getAuth, setAuth, clearAuth, request }

function get<T>(path: string): Promise<T> {
  return request<T>('GET', path)
}

function post<T>(path: string, body?: unknown, extraHeaders?: Record<string, string>): Promise<T> {
  return request<T>('POST', path, body, extraHeaders)
}

function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('PUT', path, body)
}

function del<T = void>(path: string): Promise<T> {
  return request<T>('DELETE', path)
}

export const auth = {
  login: (data: LoginRequest) => post<LoginResponse>('/api/v1/auth/login', data),
  register: (data: LoginRequest) => post<LoginResponse>('/api/v1/auth/register', data),
  forgotPassword: (email: string) =>
    request<{ message: string }>('POST', '/api/v1/auth/forgot-password', { email }),
  resetPassword: (token: string, password: string) =>
    request<{ message: string }>('POST', '/api/v1/auth/reset-password', { token, password }),
}

export const strategies = {
  list: () => get<{ strategies: Strategy[] }>('/api/v1/strategies'),
  get: (id: string) => get<Strategy>(`/api/v1/strategies/${id}`),
  create: (data: CreateStrategyRequest) => post<Strategy>('/api/v1/strategies', data),
  update: (id: string, data: UpdateStrategyRequest) => put<{ updated: boolean; strategy: Strategy }>(`/api/v1/strategies/${id}`, data),
  delete: (id: string) => del<{ deleted: boolean }>(`/api/v1/strategies/${id}`),
  validate: (data: StrategyValidationRequest) => post<StrategyValidationResponse>('/api/v1/strategies/validate', data),
  reload: (id: string) => post<{ reloaded: boolean }>(`/api/v1/strategies/${id}/reload`),
  clone: (id: string) => post<Strategy>(`/api/v1/strategies/${id}/clone`),
  deploy: (data: DeployStrategyRequest) => post<DeployStrategyResponse>('/api/v1/strategies/deploy', data),
  preflight: () => post<PreflightResponse>('/api/v1/strategies/preflight'),
  paramDefs: () => get<{ defs: Record<string, import('../types/api').ParamDef[]> }>('/api/v1/strategies/param-defs'),
  fromGkr: (data: { yaml: string }) => post<{ id?: string; name?: string; type?: string; parameters?: Record<string, unknown>; strategy_type?: string; size_model?: string; risk_profile?: string }>('/api/v1/strategies/from-gkr', data),
}

export const backtests = {
  run: (data: BacktestRequest) => post<BacktestResponse>('/api/v1/backtests', data),
  health: () => get<{ status: string }>('/api/v1/backtests/health'),
  pipeline: (data: BacktestRequest) => post<{ batch_run_id: string }>('/api/v1/backtests/pipeline', data),
  list: (params?: { run_type?: string; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.run_type) q.set('run_type', params.run_type)
    if (params?.limit) q.set('limit', String(params.limit))
    return get<{ runs: BacktestHistoryEntry[] }>(`/api/v1/backtests?${q}`)
  },
  get: (id: string) => get<BacktestHistoryEntry>(`/api/v1/backtests/${id}`),
  delete: (id: string) => del(`/api/v1/backtests/${id}`),
  rerun: (id: string) => post<{ run_id: string }>(`/api/v1/backtests/${id}/rerun`),

  metrics: (id: string) => get<BacktestMetrics>(`/api/v1/backtests/${id}/metrics`),
  equity: (id: string) => get<EquityPoint[]>(`/api/v1/backtests/${id}/equity`),
  trades: (id: string, page = 1, limit = 100) =>
    get<{ trades: TradeSummary[] }>(`/api/v1/backtests/${id}/trades?page=${page}&limit=${limit}`),
  dailyReturns: (id: string) => get<DailyReturn[]>(`/api/v1/backtests/${id}/daily-returns`),
  monthlyReturns: (id: string) => get<MonthlyReturn[]>(`/api/v1/backtests/${id}/monthly-returns`),
  optimization: (id: string) => get<OptimizationFootprint>(`/api/v1/backtests/${id}/optimization`),
  walkForward: (id: string) =>
    request<WalkForwardResponse>('GET', `/api/v1/backtests/${id}/walk-forward`),
  regimeStats: (id: string) => get<RegimeStat[]>(`/api/v1/backtests/${id}/regime-stats`),
  liveComparison: (id: string) => get<LiveComparisonResponse>(`/api/v1/backtests/${id}/live-comparison`),
  progress: (id: string) => get<{ progress: number; completed: number; total: number; status: string }>(`/api/v1/backtests/${id}/progress`),
  matrixResults: (batchId: string) => get<MatrixResultsResponse>(`/api/v1/backtests/matrix/${batchId}/results`),
  matrixResultsSince: (batchId: string, seq: number) =>
    get<MatrixResultsResponse>(`/api/v1/backtests/matrix/${batchId}/results?since=${seq}`),
  cancelMatrix: (batchId: string) =>
    post<{ status: string; batch_run_id: string }>(`/api/v1/backtests/matrix/${batchId}/cancel`, {}),
}

export const live = {
  metrics: (window?: string) => {
    const q = window ? `?window=${window}` : ''
    return get<LiveMetrics>(`/api/v1/live/metrics${q}`)
  },
  equity: (window?: string) => {
    const q = window ? `?window=${window}` : ''
    return get<EquityPoint[]>(`/api/v1/live/equity${q}`)
  },
  trades: (page = 1, limit = 100) =>
    get<{ trades: TradeSummary[] }>(`/api/v1/live/trades?page=${page}&limit=${limit}`),
  dailyReturns: () => get<DailyReturn[]>('/api/v1/live/daily-returns'),
  rollingSharpe: (window = 30) =>
    get<RollingMetric[]>(`/api/v1/live/rolling-sharpe?window=${window}`),
}

export const orders = {
  place: (data: PlaceOrderRequest) => post<Order>('/api/v1/orders', data),
  list: () => get<{ orders: Order[] }>('/api/v1/orders'),
  cancel: (id: string, accountId?: string) => {
    const q = accountId ? `?account_id=${accountId}` : ''
    return del(`/api/v1/orders/${id}${q}`)
  },
  cancelAll: (accountId?: string) => {
    const q = accountId ? `?account_id=${accountId}` : ''
    return del(`/api/v1/orders${q}`)
  },
}

export const positions = {
  list: (accountId?: string) => {
    const q = accountId ? `?account_id=${accountId}` : ''
    return get<{ positions: Position[] }>(`/api/v1/positions${q}`)
  },
}

export const risk = {
  status: () => get<RiskStatus>('/api/v1/risk/status'),
  emergencyStop: (twoFAToken: string) =>
    post<{ halted: boolean }>('/api/v1/emergency/stop', null, { 'X-2FA-Token': twoFAToken }),
  emergencyResume: (twoFAToken: string) =>
    post<{ halted: boolean }>('/api/v1/emergency/resume', null, { 'X-2FA-Token': twoFAToken }),
}

export const accounts = {
  list: () => get<Account[]>('/api/v1/accounts'),
  create: (data: CreateAccountRequest) => post<Account>('/api/v1/accounts', data),
  delete: (id: string) => del(`/api/v1/accounts/${id}`),
  setDefault: (id: string) => post(`/api/v1/accounts/${id}/default`),
}

export const brokers = {
  list: () => get<{ brokers: { id: string; label: string }[] }>('/api/v1/brokers'),
}

export const candles = {
  get: (symbol = 'SPY', range = '1D') =>
    get<CandleResponse>(`/api/v1/candles?symbol=${symbol}&range=${range}`),
}

export const symbols = {
  list: () => request<Symbol[]>('GET', '/api/v1/symbols'),
  create: (data: { ticker: string; name?: string; provider_id?: string }) =>
    request<Symbol>('POST', '/api/v1/symbols', data),
  delete: (id: string) => request<void>('DELETE', `/api/v1/symbols/${id}`),
}

export const providers = {
  list: () => request<Provider[]>('GET', '/api/v1/providers'),
  create: (data: { name: string; type: string }) =>
    request<Provider>('POST', '/api/v1/providers', data),
  delete: (id: string) => request<void>('DELETE', `/api/v1/providers/${id}`),
  test: (id: string) => request<{ success: boolean; latency_ms?: number }>('POST', `/api/v1/providers/${id}/test`),
}

export const credentials = {
  list: () => request<Credential[]>('GET', '/api/v1/credentials'),
  create: (data: { name: string; provider_type: string; api_key: string; secret_key?: string }) =>
    request<Credential>('POST', '/api/v1/credentials', data),
  rotate: (id: string) => request<Credential>('PUT', `/api/v1/credentials/${id}/rotate`),
}

export const propfirm = {
  profiles: {
    list: () => get<PropFirmProfile[]>('/api/v1/propfirm/profiles'),
    create: (data: PropFirmProfile) => post<PropFirmProfile>('/api/v1/propfirm/profiles', data),
    update: (id: string, data: Partial<PropFirmProfile>) =>
      put<PropFirmProfile>(`/api/v1/propfirm/profiles/${id}`, data),
    delete: (id: string) => del(`/api/v1/propfirm/profiles/${id}`),
  },
  active: {
    get: () => get<{ id: string }>('/api/v1/propfirm/active'),
    set: (id: string) => put('/api/v1/propfirm/active', { id }),
  },
  status: () => get<PropFirmState>('/api/v1/propfirm/status'),
}

export const settings = {
  get: () => get<AppSettings>('/api/v1/settings'),
  update: (data: AppSettings) => put('/api/v1/settings', data),
  testNotification: () => post<{ success: boolean; message: string }>('/api/v1/settings/notifications/test'),
  testLLM: (provider: string, apiKey: string, baseUrl: string, model: string) =>
    post<{ reachable: boolean; response: string }>('/api/v1/llm/test', { provider, api_key: apiKey, base_url: baseUrl, model }),
}

export const calibrate = {
  run: () => post<CalibrationReportResponse>('/api/v1/calibrate'),
}

export const attribution = {
  run: () => post<AttributionReportResponse>('/api/v1/attribute'),
}

export const admin = {
  health: () => get<{ status: string }>('/api/v1/admin/health'),
  systemHealth: () => get<{ status: string }>('/api/v1/admin/system/health'),
  users: () => get<{ users: unknown[] }>('/api/v1/admin/users'),
  auditLogs: (params?: { component?: string; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.component) q.set('component', params.component)
    if (params?.limit) q.set('limit', String(params.limit))
    return get<AuditLogEntry[]>(`/api/v1/admin/audit?${q}`)
  },
  errorLogs: (params?: { severity?: string; component?: string; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.severity) q.set('severity', params.severity)
    if (params?.component) q.set('component', params.component)
    if (params?.limit) q.set('limit', String(params.limit))
    return get<ErrorLogEntry[]>(`/api/v1/admin/logs/errors?${q}`)
  },
  seed: (force = false) => post('/api/v1/admin/seed', { force }),
  killHistory: () => get<{ events: { reason: string; source: string; triggered_at: string; resolved_at?: string }[] }>('/api/v1/admin/kill-history'),
}

export const indicators = {
  list: () => get<{ indicators: IndicatorSpec[] }>('/api/v1/indicators'),
  compute: (indicator: string, data: IndicatorComputeRequest) =>
    post<IndicatorComputeResponse>(`/api/v1/indicators/compute?indicator=${indicator}`, data),
  startStream: (symbol: string, indicator: string, timeframe = 'M1', parameters?: Record<string, number | string>) =>
    post<{ streaming: boolean }>('/api/v1/indicators/stream/start', { symbol, indicator, timeframe, parameters }),
  stopStream: (symbol: string, indicator: string, timeframe = 'M1') =>
    post<{ streaming: boolean }>('/api/v1/indicators/stream/stop', { symbol, indicator, timeframe }),
  streamStatus: () => get<{ active: number }>('/api/v1/indicators/stream/status'),
}

export const simulate = {
  generate: (params: SimulateGenerateRequest) =>
    post<SimulateGenerateResponse>('/api/v1/simulate/generate', params),
  calibrate: (params: SimulateCalibrateRequest) =>
    post<SimulateCalibrateResponse>('/api/v1/simulate/calibrate', params),
  validate: (params?: { symbol?: string }) =>
    post<SimulateValidateResponse>('/api/v1/simulate/validate', params || {}),
  calibrateRegime: (params: { symbols: string[]; timeframe?: string; start?: string; end?: string }) =>
    post<{ calibrated: boolean; regimes: Record<string, unknown> }>('/api/v1/simulate/calibrate-regime', params),
  generateTicks: (params: { generation_id: string; symbol: string; ticks_per_minute?: number; spread_bps?: number; seed?: number }) =>
    post<{ ticks_generated: number; output_path: string }>('/api/v1/simulate/ticks', params),
  injectSignal: (params: { generation_id: string; strategy: string; strength?: number }) =>
    post<{ injected: boolean; signal_count: number }>('/api/v1/simulate/inject-signal', params),
  validateRegime: (params: { generation_id: string; symbol: string; coverage?: number; min_sharpe?: number }) =>
    post<{ passed: boolean; regime_persistence: { passed: boolean }; coverage: { passed: boolean }; signal_quality: { passed: boolean } }>('/api/v1/simulate/validate-regime', params),
}

export const monitor = {
  regimeHistory: () => get<{ history: { timestamp: string; regime: number }[] }>('/api/v1/monitor/regime-history'),
}

export const dataValidate = {
  run: () => post<DataValidateResponse>('/api/v1/data/validate'),
}

export const signals = {
  list: () => get<{ signals: { symbol: string; side: string; confidence: number; reason: string; timestamp: string }[] }>('/api/v1/signals'),
}

export const models = {
  register: (data: { model_hash: string; model_type: string; model_name: string; brier_score: number; roc_auc: number; metadata?: Record<string, unknown> }) =>
    post<{ registered: boolean; id: string }>('/api/v1/models/register', data),
  compare: (modelHash: string) =>
    get<{ exists: boolean; registered_at?: string }>(`/api/v1/models/compare?model_hash=${encodeURIComponent(modelHash)}`),
  latest: (modelType: string) =>
    get<{ model_hash: string; model_type: string; model_name: string; brier_score: number; roc_auc: number; registered_at: string }>(`/api/v1/models/latest/${encodeURIComponent(modelType)}`),
}

export const reconciliation = {
  daily: (date: string) =>
    get<{ date: string; matched: number; missing: number; extra: number; price_discrepancies: number; details?: { trade_id: string; internal_price: number; broker_price: number; diff_pct: number }[] }>(`/api/v1/reconciliation/daily?date=${encodeURIComponent(date)}`),
}

export const universe = {
  current: () => get<{ symbols: { ticker: string; exchange: string; asset_type: string; id: number; is_active: boolean; last_price?: number }[] }>('/api/v1/universe/current'),
  configs: () => get<{ configs: { ID: string; Name: string; ProfileID: string; IsActive: boolean }[] }>('/api/v1/universe/configs'),
  override: (ticker: string, action: 'add' | 'remove') => post<{ total: number }>('/api/v1/universe/override', { ticker, action }),
  refresh: () => post<{ total: number }>('/api/v1/universe/refresh'),
  createConfig: (data: Record<string, unknown>) => post('/api/v1/universe/configs', data),
  activateConfig: (id: string) => post(`/api/v1/universe/configs/${id}/activate`),
}

export const system = {
  health: () => get<SystemHealth>('/api/v1/system/health'),
}
