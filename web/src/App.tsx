import { useState, useEffect, createContext, useContext, lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { ThemeProvider } from './components/ThemeProvider'
import { ThemeToggle } from './components/ThemeToggle'
import { CommandPalette } from './components/CommandPalette'
import { DynamicBreadcrumb } from './components/DynamicBreadcrumb'
import { TooltipProvider } from './components/ui/tooltip'
import { PageSkeleton } from './components/layout/PageSkeleton'
import ErrorBoundary from './components/ErrorBoundary'
import { Sidebar } from './components/layout/Sidebar'
import { useAlertToast } from './hooks/useAlertToast'
import MonitorPage from './pages/MonitorPage'
import ExecutionPage from './pages/ExecutionPage'
import BacktestHub from './pages/BacktestHub'
import StrategyHub from './pages/StrategyHub'
import SettingsPage from './pages/SettingsPage'
import AccountsPage from './pages/AccountsPage'
const AttributionPage = lazy(() => import('./pages/AttributionPage'))
const CalibratePage = lazy(() => import('./pages/CalibratePage'))
import ChartingHub from './pages/ChartingHub'
const SimulatePage = lazy(() => import('./pages/SimulatePage'))
import TwoFAPage from './pages/TwoFAPage'
const AdminPage = lazy(() => import('./pages/admin/AdminPage'))
import PropFirmPage from './pages/admin/PropFirmPage'
import IntegrationsPage from './pages/IntegrationsPage'
const UniversePage = lazy(() => import('./pages/admin/UniversePage'))
const ParamVersionPage = lazy(() => import('./pages/ParamVersionPage'))
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import ResetPasswordPage from './pages/ResetPasswordPage'
import EmergencyPage from './pages/EmergencyPage'

const AuthCtx = createContext<{ token: string | null; setToken: (t: string | null) => void }>({ token: null, setToken: () => {} })
export const useAuth = () => useContext(AuthCtx)

// Helper: wrap a lazy component with Suspense
const Lazy = ({ Comp, skeleton }: { Comp: React.ComponentType<any>, skeleton?: boolean }) => (
  <Suspense fallback={skeleton ? <PageSkeleton /> : <div className="p-6"><div className="animate-pulse space-y-4"><div className="h-4 bg-muted rounded w-1/3" /><div className="h-20 bg-muted rounded" /></div></div>}>
    <Comp />
  </Suspense>
)

function AuthenticatedApp() {
  useAlertToast()
  return (
    <div className="flex min-h-screen">
      <a href="#main-content" className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:m-2 focus:rounded focus:bg-primary focus:px-4 focus:py-2 focus:text-primary-foreground">
        Skip to main content
      </a>
      <Sidebar />
      <div className="flex-1 min-w-0 flex flex-col">
        <header className="flex h-14 shrink-0 items-center gap-2 border-b border-border bg-background sticky top-0 z-10 px-4">
          <DynamicBreadcrumb />
          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
          </div>
        </header>
        <main className="flex-1 overflow-y-auto p-4" role="main" id="main-content">
          <ErrorBoundary>
            <Routes>
            {/* Core Trading */}
            <Route path="/" element={<MonitorPage />} />
            <Route path="/live" element={<Navigate to="/" replace />} />
            <Route path="/live/market" element={<Navigate to="/charting" replace />} />
            <Route path="/risk" element={<Navigate to="/?tab=risk" replace />} />
            <Route path="/execution" element={<ExecutionPage />} />

            {/* Backtesting & Strategy */}
            <Route path="/backtest" element={<BacktestHub />} />
            <Route path="/backtest/history" element={<BacktestHub />} />
            <Route path="/backtest/history/:id" element={<BacktestHub />} />
            <Route path="/strategies" element={<StrategyHub />} />
            <Route path="/strategies/:id" element={<Navigate to="/strategies?edit=:id" replace />} />
            <Route path="/strategies/:id/edit" element={<Navigate to="/strategies?edit=:id" replace />} />
            <Route path="/strategies/new" element={<Navigate to="/strategies?edit=new" replace />} />
            <Route path="/strategies/edit/:id" element={<Navigate to="/strategies?edit=:id" replace />} />

            {/* Accounts & Prop Firms */}
            <Route path="/accounts" element={<AccountsPage />} />
            <Route path="/propfirm" element={<PropFirmPage />} />

            {/* Charts — consolidated */}
            <Route path="/market-data" element={<Navigate to="/charting" replace />} />
            <Route path="/indicators" element={<Navigate to="/charting?tab=indicators" replace />} />
            <Route path="/charting" element={<ChartingHub />} />

            {/* Integrations — consolidated */}
            <Route path="/integrations" element={<IntegrationsPage />} />
            <Route path="/brokers" element={<Navigate to="/integrations?tab=brokers" replace />} />
            <Route path="/symbols" element={<Navigate to="/integrations?tab=providers-symbols" replace />} />
            <Route path="/data-sources" element={<Navigate to="/settings?tab=trading" replace />} />

            {/* Validation & Analysis */}
            <Route path="/calibrate" element={<Lazy Comp={CalibratePage} skeleton />} />
            <Route path="/attribution" element={<Lazy Comp={AttributionPage} skeleton />} />
            <Route path="/simulate" element={<Lazy Comp={SimulatePage} skeleton />} />
            <Route path="/params" element={<Lazy Comp={ParamVersionPage} skeleton />} />

            {/* Admin */}
            <Route path="/admin" element={<Lazy Comp={AdminPage} skeleton />} />
            <Route path="/admin/universe" element={<Lazy Comp={UniversePage} skeleton />} />
            <Route path="/status" element={<Navigate to="/admin?tab=health" replace />} />

            {/* Configuration */}
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/webhooks" element={<Navigate to="/settings?tab=webhooks" replace />} />
            <Route path="/llm" element={<Navigate to="/settings?tab=llm" replace />} />
            <Route path="/notifications" element={<Navigate to="/settings?tab=notifications" replace />} />
            <Route path="/credentials" element={<Navigate to="/integrations?tab=credentials" replace />} />
            <Route path="/optimize" element={<Navigate to="/backtest?view=runner" replace />} />
            <Route path="/2fa" element={<TwoFAPage />} />
            <Route path="/emergency" element={<EmergencyPage />} />

            {/* Legacy redirects — preserved for bookmarked URLs */}
            <Route path="/admin/health" element={<Navigate to="/admin?tab=health" replace />} />
            <Route path="/admin/logs" element={<Navigate to="/admin?tab=errors" replace />} />
            <Route path="/admin/propfirm" element={<PropFirmPage />} />
            <Route path="/admin/symbols" element={<IntegrationsPage />} />
          </Routes>
        </ErrorBoundary>
        </main>
        <Toaster position="bottom-right" toastOptions={{ duration: 4000, style: { background: 'hsl(var(--popover))', color: 'hsl(var(--popover-foreground))', border: '1px solid hsl(var(--border))', fontSize: 13 } }} />
      </div>
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

  if (token) return <AuthCtx.Provider value={{ token, setToken }}><TooltipProvider delayDuration={300}><AuthenticatedApp /><CommandPalette /></TooltipProvider></AuthCtx.Provider>

  switch (page) {
    case 'register':
      return <RegisterPage onSwitchToLogin={() => switchPage('login')} />
    case 'forgot-password':
      return <ForgotPasswordPage onSwitchToLogin={() => switchPage('login')} />
    case 'reset-password':
      return <ResetPasswordPage onSwitchToLogin={() => switchPage('login')} />
    default:
      return <LoginPage
        onLogin={(t: string) => { setToken(t); window.location.href = '/' }}
        onForgotPassword={() => switchPage('forgot-password')}
        onRegister={() => switchPage('register')}
      />
  }
}

export default function App() {
  return (
    <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <ThemeProvider>
        <AuthGate />
      </ThemeProvider>
    </BrowserRouter>
  )
}
