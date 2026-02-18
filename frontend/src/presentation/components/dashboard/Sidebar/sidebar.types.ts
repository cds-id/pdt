import type { LucideIcon } from 'lucide-react'

export interface SidebarContextValue {
  isCollapsed: boolean
  isMobile: boolean
  isOpen: boolean
  toggle: () => void
  setIsOpen: (open: boolean) => void
}

export interface SidebarProps {
  className?: string
}

export interface SidebarNavProps {
  className?: string
}

export interface SidebarNavItemProps {
  title: string
  href: string
  icon: LucideIcon
  badge?: string
  disabled?: boolean
  isActive?: boolean
}

export interface SidebarNavGroupProps {
  title?: string
  items: SidebarNavItemProps[]
}
