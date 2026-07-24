import { useState, useEffect, createContext, useContext, lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { PageSkeleton } from './components/layout/PageSkeleton'
import ErrorBoundary from './components/ErrorBoundary'
import AppHeader from './components/AppHeader'
import { Sidebar } from './components/layout/Sidebar'
import Dashboard from './pages/Dashboard'
import LiveTrading from './pages/LiveTrading'
import ExecutionPage from './pages/ExecutionPage'
import BacktestPage from './pages/BacktestPage'
const BacktestDetail = lazy(() => import('./pages/BacktestDetail'))
const BacktestHistory = lazy(() => import('./pages/BacktestHistory'))
import StrategiesPage from './pages/StrategiesPage'
import SettingsPage from './pages/SettingsPage'
import RiskPage from './pages/RiskPage'
import AccountsPage from './pages/AccountsPage'
const AttributionPage = lazy(() => import('./pages/AttributionPage'))
const CalibratePage = lazy(() => import('./pages/CalibratePage'))
import IndicatorsPage from './pages/IndicatorsPage'
import MarketDataPage from './pages/MarketDataPage'
import OptimizationPanel from './pages/OptimizationPanel'
const SimulatePage = lazy(() => import('./pages/SimulatePage'))
const StrategyEditor = lazy(() => import('./pages/StrategyEditor'))
import TwoFAPage from './pages/TwoFAPage'
const AdminPage = lazy(() => import('./pages/admin/AdminPage'))
import PropFirmPage from './pages/admin/PropFirmPage'
import SymbolAdminPage from './pages/admin/SymbolAdminPage'
const UniversePage = lazy(() => import('./pages/admin/UniversePage'))
import StatusPage from './pages/StatusPage'
import BrokerManagement from './pages/BrokerManagement'
import DataSources from './pages/DataSources'
import CredentialManagement from './pages/CredentialManagement'
import WebhookConfig from './pages/WebhookConfig'
import LLMSettings from './pages/LLMSettings'
import NotificationSettings from './pages/NotificationSettings'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import EmergencyPage from './pages/EmergencyPage'

const AuthCtx = createContext<{ token: string | null; setToken: (t: string | null) => void }>({ token: null, setToken: () => {} })
export const useAuth = () => useContext(AuthCtx)

function AuthenticatedApp() {
  return (
    <div className="app-shell">
      <Sidebar />
      <AppHeader />
      <main className="main app-main" role="main" id="main-content">
        <ErrorBoundary>
          <Routes>
            {/* Core Trading */}
            <Route path="/" element={<Dashboard />} />
            <Route path="/live" element={<LiveTrading />} />
            <Route path="/live/market" element={<Navigate to="/market-data" replace />} />
            <Route path="/execution" element={<ExecutionPage />} />
            <Route path="/risk" element={<RiskPage />} />
            <Route path="/status" element={<StatusPage />} />

            {/* Backtesting & Strategy */}
            <Route path="/backtest" element={<BacktestPage />} />
            <Suspense fallback={<PageSkeleton />}>
              <Route path="/backtest/history" element={<BacktestHistory />} />
              <Route path="/backtest/history/:id" element={<BacktestDetail />} />
            </Suspense>
            <Route path="/strategies" element={<StrategiesPage />} />
            <Suspense fallback={<PageSkeleton />}>
              <Route path="/strategies/:id/edit" element={<StrategyEditor />} />
            </Suspense>
            <Route path="/optimize" element={<div className="main" style={{ maxWidth: 1000, margin: '0 auto' }}><OptimizationPanel /></div>} />

            {/* Accounts & Prop Firms */}
            <Route path="/accounts" element={<AccountsPage />} />
            <Route path="/propfirm" element={<PropFirmPage />} />

            {/* Market Data & Indicators */}
            <Route path="/market-data" element={<MarketDataPage />} />
            <Route path="/indicators" element={<IndicatorsPage />} />
            <Route path="/data-sources" element={<DataSources />} />
            <Route path="/brokers" element={<BrokerManagement />} />
            <Route path="/symbols" element={<SymbolAdminPage />} />

            {/* Validation & Analysis */}
            <Suspense fallback={<PageSkeleton />}>
              <Route path="/calibrate" element={<CalibratePage />} />
              <Route path="/attribution" element={<AttributionPage />} />
              <Route path="/simulate" element={<SimulatePage />} />
            </Suspense>

            {/* Admin */}
            <Suspense fallback={<PageSkeleton />}>
              <Route path="/admin" element={<AdminPage />} />
              <Route path="/admin/universe" element={<UniversePage />} />
            </Suspense>

            {/* Configuration */}
            <Route path="/credentials" element={<CredentialManagement />} />
            <Route path="/webhooks" element={<WebhookConfig />} />
            <Route path="/llm" element={<LLMSettings />} />
            <Route path="/2fa" element={<TwoFAPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/notifications" element={<NotificationSettings />} />

            {/* Emergency mobile access — also available without auth */}
            <Route path="/emergency" element={<EmergencyPage />} />

            {/* Legacy route redirects — maintain for 2 release cycles */}
            <Route path="/strategies/new" element={<Navigate to="/strategies/new/edit" replace />} />
            <Suspense fallback={<PageSkeleton />}>
              <Route path="/strategies/:id" element={<StrategyEditor />} />
            </Suspense>
            <Route path="/strategies/edit/:id" element={<Navigate to="/strategies/:id/edit" replace />} />
            <Route path="/admin/health" element={<Navigate to="/admin?tab=health" replace />} />
            <Route path="/admin/logs" element={<Navigate to="/admin?tab=errors" replace />} />
            <Route path="/admin/propfirm" element={<PropFirmPage />} />
            <Route path="/admin/symbols" element={<SymbolAdminPage />} />
            <Route path="/audit" element={<Navigate to="/admin?tab=audit" replace />} />
            <Route path="/users" element={<Navigate to="/admin?tab=users" replace />} />
          </Routes>
        </ErrorBoundary>
      </main>
      <Toaster position="bottom-right" toastOptions={{ duration: 4000, style: { background: '#21262d', color: '#c9d1d9', border: '1px solid #30363d', fontSize: 13 }, success: { iconTheme: { primary: '#3fb950', secondary: '#21262d' } }, error: { iconTheme: { primary: '#f85149', secondary: '#21262d' } } }} />
    </div>
  )
}

function AuthGate() {
  const [token, setToken] = useState<string | null>(() => {
    const raw = localStorage.getItem('orca_auth')
    if (!raw) return null
    try {
      const parsed = JSON.parse(raw)
      return typeof parsed === 'string' ? parsed : parsed.token || null
    } catch {
      return raw || null
    }
  })
  const [page, setPage] = useState<'login' | 'register' | 'forgot-password' | 'reset-password'>('login')

  useEffect(() => {
    const path = window.location.pathname
    if (path === '/register') setPage('register')
    else if (path === '/forgot-password') setPage('forgot-password')
    else if (path === '/reset-password') setPage('reset-password')
  }, [])

  const switchPage = (p: typeof page) => {
    setPage(p)
    const paths: Record<string, string> = { login: '/', register: '/register', 'forgot-password': '/forgot-password', 'reset-password': '/reset-password' }
    window.history.pushState({}, '', paths[p])
  }

  useEffect(() => { if (token) { localStorage.setItem('orca_auth', token) } else { localStorage.removeItem('orca_auth') } }, [token])

  if (window.location.pathname === '/emergency') return <EmergencyPage />

  if (token) return <AuthCtx.Provider value={{ token, setToken }}><AuthenticatedApp /></AuthCtx.Provider>

  switch (page) {
    case 'register':
      return <RegisterPage onSwitchToLogin={() => switchPage('login')} />
    case 'forgot-password':
      return <ForgotPasswordPage onSwitchToLogin={() => switchPage('login')} />
    case 'reset-password':
      return <ResetPasswordPage onSwitchToLogin={() => switchPage('login')} />
    default:
      return <LoginPage onLogin={(t: string) => { setToken(t); window.location.href = '/' }} />
  }
}

export default function App() { return <BrowserRouter><AuthGate /></BrowserRouter> }
