import * as React from 'react'
import { Link } from 'react-router-dom'

import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger
} from '@/components/ui/tooltip'

import { useSidebar } from './SidebarContext'
import type { SidebarNavItemProps } from './sidebar.types'

export function SidebarNavItem({
  title,
  href,
  icon: Icon,
  badge,
  disabled,
  isActive
}: SidebarNavItemProps) {
  const { isCollapsed, isMobile, setIsOpen } = useSidebar()

  const handleClick = () => {
    if (isMobile) {
      setIsOpen(false)
    }
  }

  const content = (
    <Link
      to={disabled ? '#' : href}
      onClick={handleClick}
      className={cn(
        'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
        'border-l-2 border-l-transparent text-pdt-neutral/70 hover:bg-pdt-primary-light hover:text-pdt-neutral active:bg-pdt-primary-light/80',
        isActive && 'border-l-pdt-accent bg-pdt-accent/10 text-pdt-accent',
        disabled && 'pointer-events-none opacity-50',
        isCollapsed && !isMobile && 'justify-center px-2'
      )}
    >
      <Icon className="size-5 shrink-0" />
      {(!isCollapsed || isMobile) && (
        <>
          <span className="flex-1 truncate">{title}</span>
          {badge && (
            <Badge variant="secondary" className="ml-auto shrink-0">
              {badge}
            </Badge>
          )}
        </>
      )}
    </Link>
  )

  if (isCollapsed && !isMobile) {
    return (
      <Tooltip delayDuration={0}>
        <TooltipTrigger asChild>{content}</TooltipTrigger>
        <TooltipContent side="right" className="flex items-center gap-2">
          {title}
          {badge && <Badge variant="secondary">{badge}</Badge>}
        </TooltipContent>
      </Tooltip>
    )
  }

  return content
}
