import { useState, useEffect, createContext, useContext } from 'react'
import { useTranslation } from 'react-i18next'
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import ErrorBoundary from './components/ErrorBoundary'
import AppHeader from './components/AppHeader'
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
  const { t } = useTranslation()
  const [theme, setTheme] = useState<'dark' | 'light'>(() => (localStorage.getItem('orca_theme') as 'dark' | 'light') || 'dark')
  const [collapsed, setCollapsed] = useState(false)
  useEffect(() => { document.body.className = theme === 'light' ? 'light' : ''; localStorage.setItem('orca_theme', theme) }, [theme])
  const is = (p: string) => loc.pathname === p || (p !== '/' && loc.pathname.startsWith(p))

  const groups: NavGroup[] = [
    {
      id: 'monitoring', label: t('sidebar:group.monitoring', 'Monitoring'), items: [
        { path: '/', label: t('sidebar:nav.dashboard', 'Dashboard'), icon: '⊡' },
        { path: '/live', label: t('sidebar:nav.liveTrading', 'Live Trading'), icon: '⚡' },
        { path: '/live/market', label: t('sidebar:nav.liveMarket', 'Live Market'), icon: '◉' },
        { path: '/execution', label: t('sidebar:nav.execution', 'Execution'), icon: '▶' },
        { path: '/risk', label: t('sidebar:nav.risk', 'Risk'), icon: '⚠' },
        { path: '/status', label: t('sidebar:nav.status', 'Status'), icon: '⬡' },
      ],
    },
    {
      id: 'trading', label: t('sidebar:group.trading', 'Trading'), items: [
        { path: '/backtest', label: t('sidebar:nav.backtest', 'Backtest'), icon: '◈' },
        { path: '/backtest/history', label: t('sidebar:nav.history', 'History'), icon: '☰' },
        { path: '/optimize', label: t('sidebar:nav.optimize', 'Optimize'), icon: '◆' },
        { path: '/strategies', label: t('sidebar:nav.strategies', 'Strategies'), icon: '◇' },
        { path: '/accounts', label: t('sidebar:nav.accounts', 'Accounts'), icon: '⊠' },
        { path: '/propfirm', label: t('sidebar:nav.propFirms', 'Prop Firms'), icon: '⛁' },
      ],
    },
    {
      id: 'data', label: t('sidebar:group.data', 'Data & Sources'), items: [
        { path: '/market-data', label: t('sidebar:nav.marketData', 'Market Data'), icon: '∿' },
        { path: '/indicators', label: t('sidebar:nav.indicators', 'Indicators'), icon: '≋' },
        { path: '/data-sources', label: t('sidebar:nav.dataSources', 'Data Sources'), icon: '◇' },
        { path: '/brokers', label: t('sidebar:nav.brokers', 'Brokers'), icon: '◧' },
        { path: '/symbols', label: t('sidebar:nav.symbols', 'Symbols'), icon: '♆' },
      ],
    },
    {
      id: 'validation', label: t('sidebar:group.validation', 'Validation'), items: [
        { path: '/admin/health', label: t('sidebar:nav.systemHealth', 'System Health'), icon: '⚠' },
        { path: '/admin/logs', label: t('sidebar:nav.errorLogs', 'Error Logs'), icon: '✕' },
        { path: '/audit', label: t('sidebar:nav.auditLog', 'Audit Log'), icon: '⊟' },
        { path: '/calibrate', label: t('sidebar:nav.calibration', 'Calibration'), icon: '◎' },
        { path: '/attribution', label: t('sidebar:nav.attribution', 'Attribution'), icon: '◫' },
        { path: '/simulate', label: t('sidebar:nav.simulate', 'Simulate'), icon: '≋' },
      ],
    },
    {
      id: 'config', label: t('sidebar:group.config', 'Configuration'), items: [
        { path: '/credentials', label: t('sidebar:nav.credentials', 'Credentials'), icon: '⬡' },
        { path: '/webhooks', label: t('sidebar:nav.webhooks', 'Webhooks'), icon: '⌘' },
        { path: '/llm', label: t('sidebar:nav.llm', 'LLM'), icon: '◈' },
        { path: '/2fa', label: t('sidebar:nav.2fa', '2FA'), icon: '◉' },
        { path: '/settings', label: t('sidebar:nav.settings', 'Settings'), icon: '⚙' },
        { path: '/notifications', label: t('sidebar:nav.notifications', 'Notifications'), icon: '◈' },
        { path: '/admin', label: t('sidebar:nav.admin', 'Admin'), icon: '⊛' },
      ],
    },
  ]

  return (
    <aside className={`sidebar app-sidebar${collapsed ? ' sidebar-collapsed' : ''}`} role="navigation" aria-label="Main navigation">
      <div className="sidebar-brand" onClick={() => nav('/')} role="button" tabIndex={0} onKeyDown={e => e.key === 'Enter' && nav('/')}>
        <span className="sidebar-brand-icon">O</span>
        {!collapsed && <span>{t('sidebar:brandName', 'Orca Algo')}</span>}
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
        <button className="sidebar-link" onClick={() => setCollapsed(c => !c)} title={collapsed ? t('sidebar:expand', 'Expand') : t('sidebar:collapse', 'Collapse')}>
          {collapsed ? '▶' : '◀'} {!collapsed && <span>{t('sidebar:collapse', 'Collapse')}</span>}
        </button>
        <button className="sidebar-link" onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}>
          {theme === 'dark' ? '☀' : '☾'} {!collapsed && <span>{theme === 'dark' ? t('sidebar:light', 'Light') : t('sidebar:dark', 'Dark')}</span>}
        </button>
        <button className="sidebar-link sidebar-logout" onClick={() => { localStorage.removeItem('orca_auth'); window.location.href = '/' }} aria-label={t('sidebar:logout', 'Logout')}>
          ⊘ {!collapsed && <span>{t('sidebar:logout', 'Logout')}</span>}
        </button>
      </div>
    </aside>
  )
}

function AuthenticatedApp() {
  return (
    <div className="app-shell">
      <Sidebar />
      <AppHeader />
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

  useEffect(() => { if (token) { localStorage.setItem('orca_auth', token) } else { localStorage.removeItem('orca_auth') } }, [token])

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
