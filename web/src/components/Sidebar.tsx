import { useState, useEffect, type ReactNode } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  LayoutDashboard, Zap, Play, ShieldAlert, BarChart3,
  List, Scale, Crosshair, Activity, Workflow, Wallet,
  Briefcase, Settings, UserCog, Database, Globe,
  Fingerprint, Sun, Moon, Fish, FlaskConical,
} from 'lucide-react'
import SidebarGroup from './SidebarGroup'
import { useWSStore } from '../stores/wsStore'

const THEME_KEY = 'orca_theme'

interface SidebarProps {
  visible?: boolean
}

export default function Sidebar({ visible = true }: SidebarProps) {
  const nav = useNavigate()
  const loc = useLocation()
  const [theme, setTheme] = useState<'dark' | 'light'>(() => {
    const stored = localStorage.getItem(THEME_KEY)
    return stored === 'light' ? 'light' : 'dark'
  })
  const posCount = useWSStore(s => s.openPositionCount)
  const btCount = useWSStore(s => s.activeBacktestRuns)

  useEffect(() => {
    document.body.classList.toggle('light', theme === 'light')
    localStorage.setItem(THEME_KEY, theme)
  }, [theme])

  const isActive = (p: string) => loc.pathname === p || (p !== '/' && loc.pathname.startsWith(p))
  const iconSize = 18

  const groups: { id: string; label: string; items: { path: string; label: string; icon: ReactNode; badge?: number }[] }[] = [
    {
      id: 'monitor', label: 'Monitor', items: [
        { path: '/', label: 'Dashboard', icon: <LayoutDashboard size={iconSize} /> },
        { path: '/live', label: 'Live Trading', icon: <Zap size={iconSize} />, badge: posCount > 0 ? posCount : undefined },
        { path: '/execution', label: 'Execution', icon: <Play size={iconSize} /> },
        { path: '/risk', label: 'Risk', icon: <ShieldAlert size={iconSize} /> },
      ],
    },
    {
      id: 'research', label: 'Research', items: [
        { path: '/backtest', label: 'Backtest Runner', icon: <Play size={iconSize} />, badge: btCount > 0 ? btCount : undefined },
        { path: '/backtest/history', label: 'History', icon: <List size={iconSize} /> },
        { path: '/optimization', label: 'Optimization', icon: <Workflow size={iconSize} /> },
        { path: '/indicators', label: 'Indicators', icon: <Activity size={iconSize} /> },
        { path: '/simulate', label: 'Simulation', icon: <FlaskConical size={iconSize} /> },
        { path: '/market-data', label: 'Market Data', icon: <BarChart3 size={iconSize} /> },
      ],
    },
    {
      id: 'audit', label: 'Audit', items: [
        { path: '/calibrate', label: 'Calibration', icon: <Scale size={iconSize} /> },
        { path: '/attribute', label: 'Attribution', icon: <Crosshair size={iconSize} /> },
      ],
    },
    {
      id: 'config', label: 'Config', items: [
        { path: '/strategies', label: 'Strategies', icon: <Workflow size={iconSize} /> },
        { path: '/accounts', label: 'Accounts', icon: <Wallet size={iconSize} /> },
        { path: '/propfirm', label: 'Prop Firm', icon: <Briefcase size={iconSize} /> },
        { path: '/settings', label: 'Settings', icon: <Settings size={iconSize} /> },
      ],
    },
    {
      id: 'admin', label: 'Admin', items: [
        { path: '/admin', label: 'Overview', icon: <UserCog size={iconSize} /> },
        { path: '/admin/symbols', label: 'Symbols & Providers', icon: <Database size={iconSize} /> },
        { path: '/admin/universe', label: 'Universe', icon: <Globe size={iconSize} /> },
      ],
    },
  ]

  return (
    <aside className="sidebar" data-visible={visible ? 'true' : 'false'}>
      <div className="sidebar-brand" onClick={() => nav('/')}>
        <span className="sidebar-brand-icon"><Fish size={20} /></span>
        <span>Orca Algo</span>
      </div>
      <nav className="sidebar-nav">
        {groups.map(g => (
          <SidebarGroup
            key={g.id}
            id={g.id}
            label={g.label}
            items={g.items}
            isActive={isActive}
            onNavigate={(path) => nav(path)}
          />
        ))}
      </nav>
      <div className="sidebar-footer">
        <button className="sidebar-link" onClick={() => nav('/settings/2fa')}>
          <Fingerprint size={16} /> <span>2FA Setup</span>
        </button>
        <button
          className="sidebar-link"
          onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')}
          aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
        >
          {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
          <span>{theme === 'dark' ? 'Light' : 'Dark'}</span>
        </button>
      </div>
    </aside>
  )
}
