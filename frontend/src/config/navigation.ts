import {
  LayoutDashboard,
  GitBranch,
  GitCommit,
  Kanban,
  FileText,
  Settings,
  Bot,
  Brain,
  MessageSquare,
  Send,
  Calendar,
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
      { title: 'Dashboard', href: '/dashboard/home', icon: LayoutDashboard },
      { title: 'Repositories', href: '/dashboard/repos', icon: GitBranch },
      { title: 'Commits', href: '/dashboard/commits', icon: GitCommit }
    ]
  },
  {
    title: 'Integrations',
    items: [{ title: 'Jira', href: '/dashboard/jira', icon: Kanban }]
  },
  {
    title: 'Reports',
    items: [{ title: 'Reports', href: '/dashboard/reports', icon: FileText }]
  },
  {
    title: 'WhatsApp',
    items: [
      { title: 'Outbox', href: '/dashboard/outbox', icon: Send }
    ]
  },
  {
    title: 'AI',
    items: [
      { title: 'AI Assistant', href: '/assistant', icon: Bot },
      { title: 'AI Usage', href: '/dashboard/ai-usage', icon: Brain },
      { title: 'Schedules', href: '/dashboard/schedules', icon: Calendar }
    ]
  },
  {
    title: 'System',
    items: [{ title: 'Settings', href: '/dashboard/settings', icon: Settings }]
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

  const cardMatch = pathname.match(/^\/dashboard\/jira\/cards\/(.+)$/)
  if (cardMatch) {
    breadcrumbs.push({ title: 'Jira', href: '/dashboard/jira' })
    breadcrumbs.push({ title: cardMatch[1], href: pathname })
    return breadcrumbs
  }

  const item = getNavItemByHref(pathname)
  if (item && item.href !== '/dashboard') {
    breadcrumbs.push({ title: item.title, href: item.href })
  }

  return breadcrumbs
}
