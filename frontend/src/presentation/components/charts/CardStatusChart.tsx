import { useMemo } from 'react'
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts'

import type { JiraCard } from '@/infrastructure/services/jira.service'

interface CardStatusChartProps {
  cards: JiraCard[]
}

const STATUS_COLORS: Record<string, string> = {
  'Done': '#22c55e',
  'In Progress': '#F8C630',
  'To Do': '#6b7280',
  'In Review': '#3b82f6',
  'Blocked': '#96031A'
}

const DEFAULT_COLOR = '#A66E00'

export function CardStatusChart({ cards }: CardStatusChartProps) {
  const data = useMemo(() => {
    const grouped: Record<string, number> = {}
    cards.forEach((card) => {
      grouped[card.status] = (grouped[card.status] || 0) + 1
    })
    return Object.entries(grouped).map(([status, count]) => ({
      name: status,
      value: count,
      color: STATUS_COLORS[status] || DEFAULT_COLOR
    }))
  }, [cards])

  if (data.length === 0) {
    return (
      <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
        No card data
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <PieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          innerRadius={60}
          outerRadius={100}
          paddingAngle={3}
          dataKey="value"
        >
          {data.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip
          contentStyle={{
            backgroundColor: '#1B1B1E',
            border: '1px solid #F8C63040',
            borderRadius: '8px',
            color: '#FBFFFE'
          }}
        />
        <Legend
          wrapperStyle={{ color: '#FBFFFE', fontSize: '12px' }}
        />
      </PieChart>
    </ResponsiveContainer>
  )
}
