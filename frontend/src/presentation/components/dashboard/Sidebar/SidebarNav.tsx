import * as React from 'react'

import { cn } from '@/lib/utils'
import { dashboardNavigation } from '@/config/navigation'

import { SidebarNavGroup } from './SidebarNavGroup'
import type { SidebarNavProps } from './sidebar.types'

export function SidebarNav({ className }: SidebarNavProps) {
  return (
    <div className={cn('flex flex-col gap-3 py-2', className)}>
      {dashboardNavigation.map((group, index) => (
        <SidebarNavGroup
          key={group.title || `group-${index}`}
          title={group.title}
          items={group.items.map((item) => ({
            ...item,
            isActive: false
          }))}
        />
      ))}
    </div>
  )
}
