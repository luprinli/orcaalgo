import { useLocation, useNavigate } from 'react-router-dom'

const ROUTE_LABELS: Record<string, string> = {
  '/': 'Dashboard',
  '/live': 'Live Trading',
  '/execution': 'Execution',
  '/risk': 'Risk',
  '/market-data': 'Market Data',
  '/backtest': 'Backtest Runner',
  '/backtest/history': 'Backtest History',
  '/calibrate': 'Calibration',
  '/attribute': 'Attribution',
  '/optimization': 'Optimization',
  '/indicators': 'Indicators',
  '/simulate': 'Simulation',
  '/strategies': 'Strategies',
  '/accounts': 'Accounts',
  '/propfirm': 'Prop Firm',
  '/settings': 'Settings',
  '/settings/2fa': '2FA Setup',
  '/admin': 'Admin Overview',
  '/admin/symbols': 'Symbols & Providers',
  '/admin/universe': 'Universe',
}

export default function Breadcrumbs() {
  const location = useLocation()
  const nav = useNavigate()
  const pathname = location.pathname

  const segments = pathname.split('/').filter(Boolean)
  if (segments.length === 0) return null

  const crumbs: { label: string; path: string }[] = [{ label: 'Home', path: '/' }]

  let accumulated = ''
  for (const seg of segments) {
    accumulated += '/' + seg
    const label = ROUTE_LABELS[accumulated]
    if (label) {
      crumbs.push({ label, path: accumulated })
    }
  }

  const isDetailPage = /^\/backtest\/history\/.+$/.test(pathname) || /^\/strategies\/.+$/.test(pathname)
  if (isDetailPage) {
    crumbs.push({ label: pathname.includes('backtest') ? 'Detail' : 'Edit', path: pathname })
  }

  if (crumbs.length <= 1) return null

  return (
    <nav style={{ padding: '4px 0 12px', fontSize: 12, color: 'var(--text-secondary)' }} aria-label="Breadcrumb">
      {crumbs.map((c, i) => (
        <span key={c.path}>
          {i > 0 && <span style={{ margin: '0 6px', opacity: 0.5 }}>/</span>}
          {i < crumbs.length - 1 ? (
            <span
              onClick={() => nav(c.path)}
              style={{ cursor: 'pointer', color: 'var(--accent)' }}
              onMouseEnter={e => (e.currentTarget.style.textDecoration = 'underline')}
              onMouseLeave={e => (e.currentTarget.style.textDecoration = 'none')}
            >
              {c.label}
            </span>
          ) : (
            <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{c.label}</span>
          )}
        </span>
      ))}
    </nav>
  )
}
