import * as React from 'react'

import { cn } from '@/lib/utils'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

import type { StatsCardSkeletonProps } from './stats-card.types'

export function StatsCardSkeleton({ className }: StatsCardSkeletonProps) {
  return (
    <Card
      className={cn('border-pdt-accent/20 bg-pdt-primary-light', className)}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 p-3 pb-1 sm:p-6 sm:pb-2">
        <Skeleton className="h-3 w-16 bg-pdt-neutral/10 sm:h-4 sm:w-24" />
        <Skeleton className="size-8 rounded-lg bg-pdt-neutral/10" />
      </CardHeader>
      <CardContent className="p-3 pt-0 sm:p-6 sm:pt-0">
        <Skeleton className="mb-2 h-5 w-16 bg-pdt-neutral/10 sm:h-8 sm:w-20" />
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          <Skeleton className="h-2 w-20 bg-pdt-neutral/10 sm:h-3 sm:w-32" />
          <Skeleton className="h-2 w-10 bg-pdt-neutral/10 sm:h-3 sm:w-12" />
        </div>
      </CardContent>
    </Card>
  )
}
