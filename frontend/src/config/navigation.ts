import {
  LayoutDashboard,
  BarChart3,
  Users,
  FileText,
  Settings,
  HelpCircle,
  type LucideIcon
} from 'lucide-react'

export interface NavItem {
  title: string
  href: string
  icon: LucideIcon
  badge?: string
  disabled?: boolean
}

export interface NavGroup {
  title?: string
  items: NavItem[]
}

export const dashboardNavigation: NavGroup[] = [
  {
    items: [
      { title: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
      { title: 'Analytics', href: '/analytics', icon: BarChart3 }
    ]
  },
  {
    title: 'Management',
    items: [
      { title: 'Users', href: '/users', icon: Users },
      { title: 'Reports', href: '/reports', icon: FileText }
    ]
  },
  {
    title: 'Settings',
    items: [
      { title: 'Settings', href: '/settings', icon: Settings },
      { title: 'Help', href: '/help', icon: HelpCircle }
    ]
  }
]

export const getNavItemByHref = (href: string): NavItem | undefined => {
  for (const group of dashboardNavigation) {
    const item = group.items.find((item) => item.href === href)
    if (item) return item
  }
  return undefined
}

export const getBreadcrumbsForPath = (
  pathname: string
): { title: string; href: string }[] => {
  const breadcrumbs: { title: string; href: string }[] = [
    { title: 'Home', href: '/dashboard' }
  ]

  const item = getNavItemByHref(pathname)
  if (item && item.href !== '/dashboard') {
    breadcrumbs.push({ title: item.title, href: item.href })
  }

  return breadcrumbs
}
