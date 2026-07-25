import { useLocation, Link } from "react-router-dom"
import {
  Breadcrumb,
  BreadcrumbList,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "./ui/breadcrumb"

/**
 * Auto-generates breadcrumbs from the current route pathname.
 * Maps path segments to human-readable labels.
 */
const labelMap: Record<string, string> = {
  "": "Monitor",
  monitor: "Monitor",
  execution: "Execution",
  backtest: "Backtest",
  history: "History",
  strategies: "Strategies",
  optimize: "Optimize",
  accounts: "Accounts",
  propfirm: "Prop Firms",
  "market-data": "Market Data",
  indicators: "Indicators",
  simulate: "Simulate",
  calibrate: "Calibration",
  attribution: "Attribution",
  admin: "Admin",
  universe: "Universe",
  settings: "Settings",
  credentials: "Credentials",
  brokers: "Brokers",
  symbols: "Symbols",
  "data-sources": "Data Sources",
  "2fa": "2FA Setup",
  emergency: "Emergency",
  status: "Status",
  charting: "Charting",
  integrations: "Integrations",
}

export function DynamicBreadcrumb() {
  const { pathname } = useLocation()
  const segments = pathname.split("/").filter(Boolean)

  if (segments.length === 0) {
    return (
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbPage>{labelMap[""]}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
    )
  }

  const crumbs = segments.map((seg, i) => {
    const href = "/" + segments.slice(0, i + 1).join("/")
    const label = labelMap[seg] ?? seg.charAt(0).toUpperCase() + seg.slice(1)
    const isLast = i === segments.length - 1
    return { href, label, isLast }
  })

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink asChild>
            <Link to="/">{labelMap[""]}</Link>
          </BreadcrumbLink>
        </BreadcrumbItem>
        {crumbs.map(c => (
          <BreadcrumbItem key={c.href}>
            <BreadcrumbSeparator />
            {c.isLast ? (
              <BreadcrumbPage>{c.label}</BreadcrumbPage>
            ) : (
              <BreadcrumbLink asChild>
                <Link to={c.href}>{c.label}</Link>
              </BreadcrumbLink>
            )}
          </BreadcrumbItem>
        ))}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
