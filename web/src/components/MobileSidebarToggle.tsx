import { Menu } from 'lucide-react'

interface MobileSidebarToggleProps {
  onClick: () => void
}

export default function MobileSidebarToggle({ onClick }: MobileSidebarToggleProps) {
  return (
    <button className="sidebar-mobile-toggle" onClick={onClick} aria-label="Toggle sidebar">
      <Menu size={20} />
    </button>
  )
}
