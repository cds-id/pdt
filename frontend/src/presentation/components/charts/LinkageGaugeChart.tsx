import { RadialBarChart, RadialBar, ResponsiveContainer } from 'recharts'

interface LinkageGaugeChartProps {
  linked: number
  total: number
}

export function LinkageGaugeChart({ linked, total }: LinkageGaugeChartProps) {
  const percent = total > 0 ? Math.round((linked / total) * 100) : 0

  const data = [
    { name: 'bg', value: 100, fill: '#2d2d32' },
    { name: 'linked', value: percent, fill: '#F8C630' }
  ]

  return (
    <div className="relative">
      <ResponsiveContainer width="100%" height={280}>
        <RadialBarChart
          cx="50%"
          cy="50%"
          innerRadius="60%"
          outerRadius="90%"
          startAngle={90}
          endAngle={-270}
          data={data}
          barSize={20}
        >
          <RadialBar dataKey="value" cornerRadius={10} background={false} />
        </RadialBarChart>
      </ResponsiveContainer>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-3xl font-bold text-pdt-accent">{percent}%</span>
        <span className="text-xs text-pdt-neutral/60">
          {linked}/{total} linked
        </span>
      </div>
    </div>
  )
}
