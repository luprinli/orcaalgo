import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ThemeToggle } from '../ThemeToggle'
import { Avatar, AvatarFallback } from '../ui/avatar'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem,
  DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger,
} from '../ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'
import {
  LayoutDashboard, Zap, BarChart3, Blocks,
  CandlestickChart, Target, Grid3x3, Play,
  Settings, Globe, Users, FileText, AlertTriangle,
  PanelLeftClose, PanelLeftOpen, LogOut, ChevronsUpDown,
  UserRound, SunMoon, ShieldAlert,
} from 'lucide-react'
import OrcaIcon from '../OrcaIcon'

// ── Icon lookup map ────────────────────────────────────────────────────────
const navIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  '/':               LayoutDashboard,
  '/execution':      Zap,
  '/backtest':       BarChart3,
  '/strategies':     Blocks,
  '/charting':       CandlestickChart,
  '/simulate':       Play,
  '/calibrate':      Target,
  '/attribution':    Grid3x3,
  '/settings':       Settings,
  '/integrations':   Globe,
  '/accounts':       Users,
  '/admin':          FileText,
  '/emergency':      AlertTriangle,
}

// ── NavGroup / NavItem types ────────────────────────────────────────────────
export interface NavItem { path: string; label: string; icon: string }
export interface NavGroup { id: string; label: string; items: NavItem[] }

const DEFAULT_GROUPS: NavGroup[] = [
  { id: 'trading', label: 'Trading Desk', items: [
    { path: '/',              label: 'Dashboard',     icon: '/' },
    { path: '/execution',     label: 'Execution',     icon: '/execution' },
    { path: '/backtest',      label: 'Backtesting',   icon: '/backtest' },
    { path: '/strategies',    label: 'Strategies',    icon: '/strategies' },
  ]},
  { id: 'analysis', label: 'Analysis', items: [
    { path: '/charting',      label: 'Charts',         icon: '/charting' },
    { path: '/calibrate',     label: 'Calibration',    icon: '/calibrate' },
    { path: '/attribution',   label: 'Attribution',    icon: '/attribution' },
    { path: '/simulate',      label: 'Simulation',     icon: '/simulate' },
  ]},
  { id: 'settings', label: 'Settings', items: [
    { path: '/settings',       label: 'System',         icon: '/settings' },
    { path: '/integrations',   label: 'Integrations',   icon: '/integrations' },
    { path: '/accounts',       label: 'Accounts',       icon: '/accounts' },
    { path: '/admin',          label: 'Admin',          icon: '/admin' },
    { path: '/emergency',      label: 'Emergency',      icon: '/emergency' },
  ]},
]

function getInitials(name: string): string {
  return name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0, 2)
}

// ── NavUser — shadcn-fintech-style profile dropdown ────────────────────────
function NavUser({ collapsed }: { collapsed: boolean }) {
  const { t } = useTranslation()
  const nav = useNavigate()
  const name = 'Trader'
  const role = 'Quant Desk'
  const initials = getInitials(name)

  return (
    <DropdownMenu>
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>
          <DropdownMenuTrigger asChild>
            <button
              className={`flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors
                hover:bg-sidebar-accent aria-expanded:bg-sidebar-accent aria-expanded:text-sidebar-accent-foreground
                ${collapsed ? 'justify-center p-1.5' : ''}`}
            >
              <Avatar size="sm" className="shrink-0">
                <AvatarFallback className="text-[10px]">{initials}</AvatarFallback>
              </Avatar>
              {!collapsed && (
                <>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium text-sidebar-foreground">{name}</span>
                    <span className="truncate text-[11px] text-muted-foreground">{role}</span>
                  </div>
                  <ChevronsUpDown className="ml-auto size-4 text-muted-foreground" />
                </>
              )}
            </button>
          </DropdownMenuTrigger>
        </TooltipTrigger>
        <TooltipContent side="right" hidden={!collapsed}>
          {name} — {role}
        </TooltipContent>
      </Tooltip>
      <DropdownMenuContent className="min-w-56 rounded-lg" side="right" align="end" sideOffset={4}>
        <DropdownMenuLabel className="p-1 font-normal">
          <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
            <Avatar size="sm">
              <AvatarFallback>{initials}</AvatarFallback>
            </Avatar>
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-medium">{name}</span>
              <span className="truncate text-xs text-muted-foreground">{role}</span>
            </div>
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem onClick={() => nav('/settings')}>
            <Settings className="size-4" /> {t('sidebar:account', 'Account')}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => nav('/2fa')}>
            <ShieldAlert className="size-4" /> {t('sidebar:security', 'Security')}
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="destructive"
          onClick={() => { localStorage.removeItem('orca_auth'); window.location.href = '/' }}
        >
          <LogOut className="size-4" /> {t('sidebar:logout', 'Log out')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

// ── Main Sidebar ───────────────────────────────────────────────────────────
export function Sidebar() {
  const nav = useNavigate()
  const loc = useLocation()
  const { t } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)

  const isActive = (p: string) => {
    const pathOnly = p.split('?')[0]
    return loc.pathname === pathOnly || (pathOnly !== '/' && loc.pathname.startsWith(pathOnly))
  }

  const activeStyle = "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
  const idleStyle   = "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground"

  const NavItemButton = ({ item }: { item: NavItem }) => {
    const Icon = navIcons[item.icon] ?? LayoutDashboard
    const active = isActive(item.path)

    return (
      <Tooltip delayDuration={500}>
        <TooltipTrigger asChild>
          <button
            onClick={() => nav(item.path)}
            className={`flex w-full items-center gap-2 rounded-md text-[13px] transition-colors
              ${collapsed ? 'justify-center p-1.5' : 'px-2 py-1.5'}
              ${active ? activeStyle : idleStyle}`}
          >
            <Icon className="size-4 shrink-0" />
            {!collapsed && <span className="truncate">{item.label}</span>}
          </button>
        </TooltipTrigger>
        {collapsed && (
          <TooltipContent side="right" align="center">
            {item.label}
          </TooltipContent>
        )}
      </Tooltip>
    )
  }

  return (
    <aside
      className={`flex flex-col sticky top-0 h-screen bg-sidebar border-r border-sidebar-border shrink-0 transition-all duration-200 ${collapsed ? 'w-14' : 'w-[220px]'}`}
      role="navigation"
      aria-label="Main navigation"
    >
      {/* ── Brand Header ── */}
      <div
        className={`flex items-center gap-2.5 py-3.5 cursor-pointer select-none ${collapsed ? 'justify-center px-2' : 'px-3'}`}
        onClick={() => nav('/')}
        role="button"
        tabIndex={0}
        onKeyDown={e => e.key === 'Enter' && nav('/')}
      >
        <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
          <OrcaIcon className="size-5" />
        </div>
        {!collapsed && (
          <div className="grid flex-1 text-left text-sm leading-tight">
            <span className="truncate font-semibold text-sidebar-foreground">
              {t('sidebar:brandName', 'Orca Algo')}
            </span>
            <span className="truncate text-[11px] text-muted-foreground">
              {t('sidebar:brandSubtitle', 'Quant Trading')}
            </span>
          </div>
        )}
      </div>

      {/* ── Nav Groups ── */}
      <nav className="flex-1 flex flex-col gap-1 overflow-y-auto px-1.5 py-2">
        {DEFAULT_GROUPS.map(g => (
          <div key={g.id} className="flex flex-col gap-0.5">
            <div
              className={`${collapsed ? 'sr-only' : ''} text-[10px] font-medium text-sidebar-foreground/40 uppercase tracking-wider px-2 py-1`}
            >
              {g.label}
            </div>
            {g.items.map(i => (
              <NavItemButton key={i.path} item={i} />
            ))}
          </div>
        ))}
      </nav>

      {/* ── Footer ── */}
      <div className="border-t border-sidebar-border flex flex-col px-1.5 py-2 gap-0.5">
        {/* Theme toggle */}
        <div className={`flex items-center rounded-md transition-colors hover:bg-sidebar-accent/50 ${collapsed ? 'justify-center p-1.5' : 'px-2 py-1.5 gap-2'}`}>
          <ThemeToggle />
          {!collapsed && <span className="text-[13px] text-sidebar-foreground/70">{t('sidebar:theme', 'Theme')}</span>}
        </div>
        {/* Collapse */}
        <button
          className={`flex items-center rounded-md transition-colors hover:bg-sidebar-accent/50 ${collapsed ? 'justify-center p-1.5' : 'px-2 py-1.5 gap-2'}`}
          onClick={() => setCollapsed(c => !c)}
          title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
          {!collapsed && <span className="text-[13px] text-sidebar-foreground/70">{t('sidebar:collapse', 'Collapse')}</span>}
        </button>
        {/* User profile */}
        <NavUser collapsed={collapsed} />
      </div>
    </aside>
  )
}
