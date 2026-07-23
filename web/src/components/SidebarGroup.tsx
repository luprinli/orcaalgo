import { useState, type ReactNode } from 'react'
import { ChevronRight } from 'lucide-react'

interface SidebarGroupItem {
  path: string
  label: string
  icon: ReactNode
  badge?: number
}

interface SidebarGroupProps {
  id: string
  label: string
  items: SidebarGroupItem[]
  isActive: (path: string) => boolean
  onNavigate: (path: string) => void
}

export default function SidebarGroup({ id, label, items, isActive, onNavigate }: SidebarGroupProps) {
  const [collapsed, setCollapsed] = useState(() => {
    return localStorage.getItem(`orca_group_${id}`) === 'collapsed'
  })
  const toggle = () => {
    const next = !collapsed
    setCollapsed(next)
    localStorage.setItem(`orca_group_${id}`, next ? 'collapsed' : 'expanded')
  }
  return (
    <div className="sidebar-group">
      <button className="sidebar-group-label" onClick={toggle} aria-expanded={!collapsed}>
        <ChevronRight size={10} style={{
          transform: collapsed ? 'rotate(0deg)' : 'rotate(90deg)',
          transition: 'transform .15s',
          flexShrink: 0,
        }} />
        {label}
      </button>
      {!collapsed && items.map(i => (
        <button key={i.path} className={`sidebar-link${isActive(i.path) ? ' active' : ''}`}
          onClick={() => onNavigate(i.path)} aria-current={isActive(i.path) ? 'page' : undefined}>
          {i.icon} <span>{i.label}</span>
          {i.badge !== undefined && i.badge > 0 && (
            <span className="sidebar-badge">{i.badge}</span>
          )}
        </button>
      ))}
    </div>
  )
}
