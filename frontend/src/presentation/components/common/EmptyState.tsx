import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-pdt-background/20 bg-pdt-primary-light p-8 text-center',
        className
      )}
    >
      {Icon && (
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-pdt-background/10">
          <Icon className="h-6 w-6 text-pdt-background" />
        </div>
      )}
      <p className="text-pdt-neutral/60">{title}</p>
      {description && (
        <p className="mt-2 text-sm text-pdt-neutral/40">{description}</p>
      )}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}
