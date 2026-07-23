import { useState, useEffect, createContext, useContext } from 'react'
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import ErrorBoundary from './components/ErrorBoundary'
import Dashboard from './pages/Dashboard'
import LiveTrading from './pages/LiveTrading'
import ExecutionPage from './pages/ExecutionPage'
import BacktestPage from './pages/BacktestPage'
import BacktestDetail from './pages/BacktestDetail'
import BacktestHistory from './pages/BacktestHistory'
import StrategiesPage from './pages/StrategiesPage'
import SettingsPage from './pages/SettingsPage'
import RiskPage from './pages/RiskPage'
import AccountsPage from './pages/AccountsPage'
import AttributionPage from './pages/AttributionPage'
import CalibratePage from './pages/CalibratePage'
import IndicatorsPage from './pages/IndicatorsPage'
import MarketDataPage from './pages/MarketDataPage'
import OptimizationPage from './pages/OptimizationPage'
import SimulatePage from './pages/SimulatePage'
import StrategyEditor from './pages/StrategyEditor'
import TwoFAPage from './pages/TwoFAPage'
import AdminPage from './pages/admin/AdminPage'
import PropFirmPage from './pages/admin/PropFirmPage'
import SymbolAdminPage from './pages/admin/SymbolAdminPage'
import UniversePage from './pages/admin/UniversePage'
import LiveMarket from './pages/LiveMarket'
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

const AuthCtx = createContext<{ token: string | null; setToken: (t: string | null) => void }>({ token: null, setToken: () => {} })
export const useAuth = () => useContext(AuthCtx)

interface NavItem {
  path: string
  label: string
  icon: string
}

interface NavGroup {
  id: string
  label: string
  items: NavItem[]
}

function Sidebar() {
  const nav = useNavigate()
  const loc = useLocation()
  const [theme, setTheme] = useState<'dark' | 'light'>(() => (localStorage.getItem('orca_theme') as 'dark' | 'light') || 'dark')
  const [collapsed, setCollapsed] = useState(false)
  useEffect(() => { document.body.className = theme === 'light' ? 'light' : ''; localStorage.setItem('orca_theme', theme) }, [theme])
  const is = (p: string) => loc.pathname === p || (p !== '/' && loc.pathname.startsWith(p))

  const groups: NavGroup[] = [
    {
      id: 'monitoring', label: 'Monitoring', items: [
        { path: '/', label: 'Dashboard', icon: '⊡' },
        { path: '/live', label: 'Live Trading', icon: '⚡' },
        { path: '/live/market', label: 'Live Market', icon: '◉' },
        { path: '/execution', label: 'Execution', icon: '▶' },
        { path: '/status', label: 'Status', icon: '⬡' },
      ],
    },
    {
      id: 'trading', label: 'Trading', items: [
        { path: '/backtest', label: 'Backtest', icon: '◈' },
        { path: '/backtest/history', label: 'History', icon: '☰' },
        { path: '/strategies', label: 'Strategies', icon: '◇' },
        { path: '/accounts', label: 'Accounts', icon: '⊠' },
        { path: '/propfirm', label: 'Prop Firms', icon: '⛁' },
      ],
    },
    {
      id: 'data', label: 'Data & Sources', items: [
        { path: '/market-data', label: 'Market Data', icon: '∿' },
        { path: '/indicators', label: 'Indicators', icon: '≋' },
        { path: '/data-sources', label: 'Data Sources', icon: '◇' },
        { path: '/brokers', label: 'Brokers', icon: '◧' },
        { path: '/symbols', label: 'Symbols', icon: '♆' },
      ],
    },
    {
      id: 'validation', label: 'Validation', items: [
        { path: '/admin/health', label: 'System Health', icon: '⚠' },
        { path: '/admin/logs', label: 'Error Logs', icon: '✕' },
        { path: '/audit', label: 'Audit Log', icon: '⊟' },
        { path: '/calibrate', label: 'Calibration', icon: '◎' },
        { path: '/attribution', label: 'Attribution', icon: '◫' },
      ],
    },
    {
      id: 'config', label: 'Configuration', items: [
        { path: '/credentials', label: 'Credentials', icon: '⬡' },
        { path: '/webhooks', label: 'Webhooks', icon: '⌘' },
        { path: '/llm', label: 'LLM', icon: '◈' },
        { path: '/2fa', label: '2FA', icon: '◉' },
        { path: '/settings', label: 'Settings', icon: '⚙' },
        { path: '/notifications', label: 'Notifications', icon: '◈' },
        { path: '/admin', label: 'Admin', icon: '⊛' },
      ],
    },
  ]

  return (
    <aside className={`sidebar app-sidebar${collapsed ? ' sidebar-collapsed' : ''}`} role="navigation" aria-label="Main navigation">
      <div className="sidebar-brand" onClick={() => nav('/')} role="button" tabIndex={0} onKeyDown={e => e.key === 'Enter' && nav('/')}>
        <span className="sidebar-brand-icon">O</span>
        {!collapsed && <span>Orca Algo</span>}
      </div>
      <nav className="sidebar-nav">
        {groups.map(g => (
          <div key={g.id}>
            {!collapsed && <div className="sidebar-group-label">{g.label}</div>}
            {g.items.map(i => (
              <button key={i.path} className={`sidebar-link${is(i.path) ? ' active' : ''}`} onClick={() => nav(i.path)} title={collapsed ? i.label : undefined}>
                {i.icon} {!collapsed && <span>{i.label}</span>}
              </button>
            ))}
          </div>
        ))}
      </nav>
      <div className="sidebar-footer">
        <button className="sidebar-link" onClick={() => setCollapsed(c => !c)} title={collapsed ? 'Expand' : 'Collapse'}>
          {collapsed ? '▶' : '◀'} {!collapsed && <span>Collapse</span>}
        </button>
        <button className="sidebar-link" onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}>
          {theme === 'dark' ? '☀' : '☾'} {!collapsed && <span>{theme === 'dark' ? 'Light' : 'Dark'}</span>}
        </button>
        <button className="sidebar-link sidebar-logout" onClick={() => { localStorage.removeItem('orca_auth'); window.location.href = '/' }} aria-label="Log out">
          ⊘ {!collapsed && <span>Logout</span>}
        </button>
      </div>
    </aside>
  )
}

function AuthenticatedApp() {
  return (
    <div className="app-shell">
      <Sidebar />
      <main className="main app-main" role="main" id="main-content">
        <ErrorBoundary>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/live" element={<LiveTrading />} />
            <Route path="/live/market" element={<LiveMarket />} />
            <Route path="/execution" element={<ExecutionPage />} />
            <Route path="/status" element={<StatusPage />} />
            <Route path="/risk" element={<RiskPage />} />
            <Route path="/backtest" element={<BacktestPage />} />
            <Route path="/backtest/history" element={<BacktestHistory />} />
            <Route path="/backtest/history/:id" element={<BacktestDetail />} />
            <Route path="/strategies" element={<StrategiesPage />} />
            <Route path="/strategies/new" element={<Navigate to="/strategies/edit/new" replace />} />
            <Route path="/strategies/:id" element={<StrategyEditor />} />
            <Route path="/strategies/:id/edit" element={<StrategyEditor />} />
            <Route path="/strategies/edit/:id" element={<StrategyEditor />} />
            <Route path="/optimize" element={<OptimizationPage />} />
            <Route path="/accounts" element={<AccountsPage />} />
            <Route path="/propfirm" element={<PropFirmPage />} />
            <Route path="/market-data" element={<MarketDataPage />} />
            <Route path="/indicators" element={<IndicatorsPage />} />
            <Route path="/data-sources" element={<DataSources />} />
            <Route path="/brokers" element={<BrokerManagement />} />
            <Route path="/symbols" element={<SymbolAdminPage />} />
            <Route path="/calibrate" element={<CalibratePage />} />
            <Route path="/attribution" element={<AttributionPage />} />
            <Route path="/simulate" element={<SimulatePage />} />
            <Route path="/admin" element={<AdminPage />} />
            <Route path="/admin/health" element={<Navigate to="/admin?tab=health" replace />} />
            <Route path="/admin/logs" element={<Navigate to="/admin?tab=errors" replace />} />
            <Route path="/admin/propfirm" element={<PropFirmPage />} />
            <Route path="/admin/symbols" element={<SymbolAdminPage />} />
            <Route path="/admin/universe" element={<UniversePage />} />
            <Route path="/audit" element={<Navigate to="/admin?tab=audit" replace />} />
            <Route path="/users" element={<Navigate to="/admin?tab=users" replace />} />
            <Route path="/credentials" element={<CredentialManagement />} />
            <Route path="/webhooks" element={<WebhookConfig />} />
            <Route path="/llm" element={<LLMSettings />} />
            <Route path="/2fa" element={<TwoFAPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/notifications" element={<NotificationSettings />} />
          </Routes>
        </ErrorBoundary>
      </main>
      <Toaster position="bottom-right" toastOptions={{ duration: 4000, style: { background: '#21262d', color: '#c9d1d9', border: '1px solid #30363d', fontSize: 13 }, success: { iconTheme: { primary: '#3fb950', secondary: '#21262d' } }, error: { iconTheme: { primary: '#f85149', secondary: '#21262d' } } }} />
    </div>
  )
}

function AuthGate() {
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('orca_auth'))
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

  useEffect(() => { token ? localStorage.setItem('orca_auth', token) : localStorage.removeItem('orca_auth') }, [token])

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
