import { cn } from '@/lib/utils'

type BadgeVariant = 'success' | 'warning' | 'info' | 'neutral' | 'danger'

interface StatusBadgeProps {
  children: React.ReactNode
  variant?: BadgeVariant
  className?: string
}

const variantStyles: Record<BadgeVariant, string> = {
  success: 'bg-green-500/20 text-green-400',
  warning: 'bg-pdt-background/20 text-pdt-background',
  info: 'bg-blue-500/20 text-blue-400',
  neutral: 'bg-gray-500/20 text-gray-400',
  danger: 'bg-red-500/20 text-red-400'
}

export function StatusBadge({ children, variant = 'neutral', className }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded px-2 py-0.5 text-xs font-medium',
        variantStyles[variant],
        className
      )}
    >
      {children}
    </span>
  )
}
