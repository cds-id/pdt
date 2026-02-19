import * as React from 'react'

import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { TrendIndicator } from './TrendIndicator'
import type { StatsCardProps } from './stats-card.types'

export function StatsCard({
  title,
  value,
  description,
  icon: Icon,
  trend,
  className
}: StatsCardProps) {
  return (
    <Card className={cn('border-pdt-background/20 bg-pdt-primary-light', className)}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 p-3 pb-1 sm:p-6 sm:pb-2">
        <CardTitle className="text-xs font-medium text-pdt-neutral/70 sm:text-sm">
          {title}
        </CardTitle>
        {Icon && (
          <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-pdt-background/30">
            <Icon className="size-4 text-pdt-background" />
          </div>
        )}
      </CardHeader>
      <CardContent className="p-3 pt-0 sm:p-6 sm:pt-0">
        <div className="text-lg font-bold text-pdt-neutral sm:text-2xl">{value}</div>
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          {description && (
            <p className="truncate text-[10px] text-pdt-neutral/50 sm:text-xs">
              {description}
            </p>
          )}
          {trend && (
            <TrendIndicator value={trend.value} isPositive={trend.isPositive} />
          )}
        </div>
      </CardContent>
    </Card>
  )
}
