import * as React from 'react'
import { useLocation } from 'react-router-dom'

import { cn } from '@/lib/utils'

import { useSidebar } from './SidebarContext'
import { SidebarNavItem } from './SidebarNavItem'
import type { SidebarNavGroupProps } from './sidebar.types'

export function SidebarNavGroup({ title, items }: SidebarNavGroupProps) {
  const { isCollapsed, isMobile } = useSidebar()
  const location = useLocation()

  const showTitle = title && (!isCollapsed || isMobile)

  return (
    <div className="space-y-0.5">
      {showTitle && (
        <h4
          className={cn(
            'mb-1.5 px-3 text-[11px] font-semibold uppercase tracking-wider text-pdt-neutral/40'
          )}
        >
          {title}
        </h4>
      )}
      <nav className="space-y-0.5">
        {items.map((item) => (
          <SidebarNavItem
            key={item.href}
            {...item}
            isActive={location.pathname === item.href}
          />
        ))}
      </nav>
    </div>
  )
}
