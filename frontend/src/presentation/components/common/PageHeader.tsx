import type { ReactNode } from 'react'

interface PageHeaderProps {
  title: string
  description?: string
  action?: ReactNode
}

export function PageHeader({ title, description, action }: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-pdt-neutral md:text-3xl">
          {title}
        </h1>
        {description && (
          <p className="text-sm text-pdt-neutral/60 md:text-base">{description}</p>
        )}
      </div>
      {action && <div className="flex-shrink-0">{action}</div>}
    </div>
  )
}
