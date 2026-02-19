import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface DataCardProps {
  title?: string
  action?: ReactNode
  children: ReactNode
  className?: string
}

export function DataCard({ title, action, children, className }: DataCardProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-pdt-accent/20 bg-pdt-primary-light p-4',
        className
      )}
    >
      {(title || action) && (
        <div className="mb-4 flex items-center justify-between">
          {title && (
            <h2 className="text-lg font-semibold text-pdt-neutral">{title}</h2>
          )}
          {action && <div>{action}</div>}
        </div>
      )}
      {children}
    </div>
  )
}
