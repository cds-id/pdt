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

import type { DailyBucket } from '@/infrastructure/services/executiveReport.service'

interface ExecutiveActivityChartProps {
  buckets: DailyBucket[]
}

export function ExecutiveActivityChart({ buckets }: ExecutiveActivityChartProps) {
  const data = useMemo(() => {
    return buckets.map((b) => ({
      day: b.day,
      label: new Date(b.day).toLocaleDateString('en', {
        month: 'short',
        day: 'numeric'
      }),
      commits: b.commits,
      jira: b.jira_changes,
      wa: b.wa_messages
    }))
  }, [buckets])

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="commitsGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#2563eb" stopOpacity={0.4} />
            <stop offset="95%" stopColor="#2563eb" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="jiraGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.4} />
            <stop offset="95%" stopColor="#f59e0b" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="waGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#10b981" stopOpacity={0.4} />
            <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
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
          stackId="1"
          stroke="#2563eb"
          strokeWidth={2}
          fill="url(#commitsGradient)"
          name="Commits"
        />
        <Area
          type="monotone"
          dataKey="jira"
          stackId="1"
          stroke="#f59e0b"
          strokeWidth={2}
          fill="url(#jiraGradient)"
          name="Jira changes"
        />
        <Area
          type="monotone"
          dataKey="wa"
          stackId="1"
          stroke="#10b981"
          strokeWidth={2}
          fill="url(#waGradient)"
          name="WA messages"
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}
