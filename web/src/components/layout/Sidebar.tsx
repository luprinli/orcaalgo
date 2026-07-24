import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export interface NavItem {
  path: string
  label: string
  icon: string
}

export interface NavGroup {
  id: string
  label: string
  items: NavItem[]
}

// Default navigation groups — can be configured per deployment
// Phase 2 will consolidate these after page merges
const DEFAULT_GROUPS: NavGroup[] = [
  {
    id: 'monitoring', label: 'Monitoring', items: [
      { path: '/', label: 'Command Center', icon: '⊡' },
      { path: '/live/market', label: 'Live Market', icon: '◉' },
      { path: '/execution', label: 'Execution', icon: '▶' },
      { path: '/risk', label: 'Risk', icon: '⚠' },
      { path: '/status', label: 'Status', icon: '⬡' },
    ],
  },
  {
    id: 'trading', label: 'Trading', items: [
      { path: '/backtest', label: 'Backtest', icon: '◈' },
      { path: '/backtest/history', label: 'History', icon: '☰' },
      { path: '/optimize', label: 'Optimize', icon: '◆' },
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
      { path: '/admin?tab=health', label: 'System Health', icon: '⚠' },
      { path: '/admin?tab=errors', label: 'Error Logs', icon: '✕' },
      { path: '/admin?tab=audit', label: 'Audit Log', icon: '⊟' },
      { path: '/calibrate', label: 'Calibration', icon: '◎' },
      { path: '/attribution', label: 'Attribution', icon: '◫' },
      { path: '/simulate', label: 'Simulate', icon: '≋' },
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

export function Sidebar() {
  const nav = useNavigate()
  const loc = useLocation()
  const { t } = useTranslation()
  const [theme, setTheme] = useState<'dark' | 'light'>(
    () => (localStorage.getItem('orca_theme') as 'dark' | 'light') || 'dark'
  )
  const [collapsed, setCollapsed] = useState(false)

  useEffect(() => {
    document.body.className = theme === 'light' ? 'light' : ''
    localStorage.setItem('orca_theme', theme)
  }, [theme])

  const is = (p: string) => {
    // Handle query-param paths: /admin?tab=health should highlight /admin
    const pathOnly = p.split('?')[0]
    return loc.pathname === pathOnly || (pathOnly !== '/' && loc.pathname.startsWith(pathOnly))
  }

  const groups: NavGroup[] = DEFAULT_GROUPS

  return (
    <aside
      className={`sidebar app-sidebar${collapsed ? ' sidebar-collapsed' : ''}`}
      role="navigation"
      aria-label="Main navigation"
    >
      <div
        className="sidebar-brand"
        onClick={() => nav('/')}
        role="button"
        tabIndex={0}
        onKeyDown={e => e.key === 'Enter' && nav('/')}
      >
        <span className="sidebar-brand-icon">O</span>
        {!collapsed && <span>{t('sidebar:brandName', 'Orca Algo')}</span>}
      </div>
      <nav className="sidebar-nav">
        {groups.map(g => (
          <div key={g.id}>
            {!collapsed && <div className="sidebar-group-label">{g.label}</div>}
            {g.items.map(i => (
              <button
                key={i.path}
                className={`sidebar-link${is(i.path) ? ' active' : ''}`}
                onClick={() => nav(i.path)}
                title={collapsed ? i.label : undefined}
              >
                {i.icon} {!collapsed && <span>{i.label}</span>}
              </button>
            ))}
          </div>
        ))}
      </nav>
      <div className="sidebar-footer">
        <button
          className="sidebar-link"
          onClick={() => setCollapsed(c => !c)}
          title={collapsed ? t('sidebar:expand', 'Expand') : t('sidebar:collapse', 'Collapse')}
        >
          {collapsed ? '▶' : '◀'}{' '}
          {!collapsed && <span>{t('sidebar:collapse', 'Collapse')}</span>}
        </button>
        <button className="sidebar-link" onClick={() => setTheme(t => (t === 'dark' ? 'light' : 'dark'))}>
          {theme === 'dark' ? '☀' : '☾'}{' '}
          {!collapsed && <span>{theme === 'dark' ? t('sidebar:light', 'Light') : t('sidebar:dark', 'Dark')}</span>}
        </button>
        <button
          className="sidebar-link sidebar-logout"
          onClick={() => {
            localStorage.removeItem('orca_auth')
            window.location.href = '/'
          }}
          aria-label={t('sidebar:logout', 'Logout')}
        >
          ⊘ {!collapsed && <span>{t('sidebar:logout', 'Logout')}</span>}
        </button>
      </div>
    </aside>
  )
}
