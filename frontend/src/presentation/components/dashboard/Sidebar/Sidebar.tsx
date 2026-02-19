import * as React from 'react'
import { PanelLeftClose, PanelLeft } from 'lucide-react'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

import { useSidebar } from './SidebarContext'
import { SidebarNav } from './SidebarNav'
import type { SidebarProps } from './sidebar.types'

export function Sidebar({ className }: SidebarProps) {
  const { isCollapsed, isMobile, toggle } = useSidebar()

  if (isMobile) {
    return null
  }

  return (
      <aside
        className={cn(
          'flex flex-col border-r border-pdt-background/20 bg-pdt-primary transition-all duration-300',
          isCollapsed ? 'w-16' : 'w-64',
          className
        )}
      >
        <div
          className={cn(
            'flex h-16 items-center border-b border-pdt-background/20 px-3',
            isCollapsed && 'justify-center px-2'
          )}
        >
          <div className="flex items-center gap-2">
            <img
              src="/logo.svg"
              alt="Logo"
              className={cn('size-8', isCollapsed && 'size-7')}
            />
            {!isCollapsed && (
              <span className="text-base font-semibold text-pdt-neutral">Dashboard</span>
            )}
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={toggle}
            className={cn('ml-auto size-8 text-pdt-background hover:bg-pdt-primary-light hover:text-pdt-background', isCollapsed && 'ml-0 hidden')}
          >
            {isCollapsed ? (
              <PanelLeft className="size-4" />
            ) : (
              <PanelLeftClose className="size-4" />
            )}
            <span className="sr-only">Toggle sidebar</span>
          </Button>
        </div>

        <div className="flex-1 overflow-y-auto px-3 scrollbar-none">
          <SidebarNav />
        </div>
      </aside>
  )
}
