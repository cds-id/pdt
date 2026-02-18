import * as React from 'react'
import { Inbox } from 'lucide-react'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

import type { EmptyStateProps } from './empty-state.types'

export function EmptyState({
  icon: Icon = Inbox,
  title,
  description,
  action,
  className
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center rounded-lg border border-dashed p-6 text-center sm:p-8',
        className
      )}
    >
      <div className="flex size-10 items-center justify-center rounded-full bg-muted sm:size-12">
        <Icon className="size-5 text-muted-foreground sm:size-6" />
      </div>

      <h3 className="mt-3 text-base font-semibold sm:mt-4 sm:text-lg">
        {title}
      </h3>

      {description && (
        <p className="mt-1.5 max-w-sm text-xs text-muted-foreground sm:mt-2 sm:text-sm">
          {description}
        </p>
      )}

      {action && (
        <Button onClick={action.onClick} size="sm" className="mt-3 sm:mt-4">
          {action.label}
        </Button>
      )}
    </div>
  )
}
