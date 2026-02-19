import { useMemo } from 'react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from 'recharts'

import type { JiraSprint } from '@/infrastructure/services/jira.service'

interface SprintVelocityChartProps {
  sprints: JiraSprint[]
}

export function SprintVelocityChart({ sprints }: SprintVelocityChartProps) {
  const data = useMemo(() => {
    return sprints
      .filter((s) => s.cards && s.cards.length > 0)
      .sort((a, b) => (a.start_date || '').localeCompare(b.start_date || ''))
      .slice(-5)
      .map((sprint) => {
        const total = sprint.cards?.length || 0
        const done = sprint.cards?.filter((c) => c.status === 'Done').length || 0
        return {
          name: sprint.name.length > 15 ? sprint.name.slice(0, 15) + '...' : sprint.name,
          total,
          done
        }
      })
  }, [sprints])

  if (data.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
        No sprint data
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" stroke="#2d2d32" />
        <XAxis
          dataKey="name"
          stroke="#FBFFFE50"
          fontSize={11}
          tickLine={false}
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
        <Bar dataKey="total" fill="#A66E00" radius={[4, 4, 0, 0]} name="Total Cards" />
        <Bar dataKey="done" fill="#F8C630" radius={[4, 4, 0, 0]} name="Completed" />
      </BarChart>
    </ResponsiveContainer>
  )
}
