import * as React from 'react'
import { TrendingUp, TrendingDown } from 'lucide-react'

import { cn } from '@/lib/utils'

import type { TrendIndicatorProps } from './stats-card.types'

export function TrendIndicator({
  value,
  isPositive,
  className
}: TrendIndicatorProps) {
  const Icon = isPositive ? TrendingUp : TrendingDown

  return (
    <div
      className={cn(
        'flex items-center gap-1 text-xs font-medium',
        isPositive ? 'text-green-600' : 'text-red-600',
        className
      )}
    >
      <Icon className="size-3" />
      <span>{Math.abs(value)}%</span>
    </div>
  )
}
