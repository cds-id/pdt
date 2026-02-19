import { useMemo } from 'react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from 'recharts'

import type { Commit } from '@/infrastructure/services/commit.service'

interface CommitActivityChartProps {
  commits: Commit[]
}

export function CommitActivityChart({ commits }: CommitActivityChartProps) {
  const data = useMemo(() => {
    const grouped: Record<string, number> = {}

    for (let i = 29; i >= 0; i--) {
      const d = new Date()
      d.setDate(d.getDate() - i)
      const key = d.toISOString().split('T')[0]
      grouped[key] = 0
    }

    commits.forEach((c) => {
      const key = new Date(c.date).toISOString().split('T')[0]
      if (key in grouped) {
        grouped[key]++
      }
    })

    return Object.entries(grouped).map(([date, count]) => ({
      date,
      label: new Date(date).toLocaleDateString('en', {
        month: 'short',
        day: 'numeric'
      }),
      commits: count
    }))
  }, [commits])

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="commitGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#F8C630" stopOpacity={0.4} />
            <stop offset="95%" stopColor="#F8C630" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#2d2d32" />
        <XAxis
          dataKey="label"
          stroke="#FBFFFE50"
          fontSize={11}
          tickLine={false}
          interval="preserveStartEnd"
        />
        <YAxis
          stroke="#FBFFFE50"
          fontSize={11}
          tickLine={false}
          allowDecimals={false}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: '#1B1B1E',
            border: '1px solid #F8C63040',
            borderRadius: '8px',
            color: '#FBFFFE'
          }}
        />
        <Area
          type="monotone"
          dataKey="commits"
          stroke="#F8C630"
          strokeWidth={2}
          fill="url(#commitGradient)"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
