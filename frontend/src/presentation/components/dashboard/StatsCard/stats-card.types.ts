import type { LucideIcon } from 'lucide-react'

export interface StatsCardProps {
  title: string
  value: string | number
  description?: string
  icon?: LucideIcon
  trend?: {
    value: number
    isPositive: boolean
  }
  className?: string
}

export interface TrendIndicatorProps {
  value: number
  isPositive: boolean
  className?: string
}

export interface StatsCardSkeletonProps {
  className?: string
}
