import { useMemo } from 'react'
import { Brain, Eye, MessageSquare, Zap } from 'lucide-react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend
} from 'recharts'

import { useGetAIUsageQuery } from '@/infrastructure/services/ai-usage.service'
import {
  StatsCard,
  StatsCardSkeleton
} from '@/presentation/components/dashboard'
import { PageHeader, DataCard } from '@/presentation/components/common'

const PROVIDER_COLORS: Record<string, string> = {
  minimax: '#F8C630',
  mistral: '#FF6B35',
  gemini: '#4285F4'
}

const PROVIDER_LABELS: Record<string, string> = {
  minimax: 'MiniMax',
  mistral: 'Mistral',
  gemini: 'Gemini'
}

const FEATURE_LABELS: Record<string, string> = {
  chat: 'AI Chat',
  wa_vision: 'WA Vision',
  embedding: 'Embedding'
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toString()
}

export function AIUsagePage() {
  const { data, isLoading } = useGetAIUsageQuery()

  const todayCalls = useMemo(
    () => data?.today?.reduce((sum, p) => sum + p.calls, 0) ?? 0,
    [data]
  )

  const todayTokens = useMemo(
    () =>
      data?.today?.reduce(
        (sum, p) => sum + p.prompt_tokens + p.completion_tokens,
        0
      ) ?? 0,
    [data]
  )

  const monthCalls = useMemo(
    () => data?.month?.reduce((sum, p) => sum + p.calls, 0) ?? 0,
    [data]
  )

  const chartData = useMemo(() => {
    if (!data?.daily) return []

    // Build a map of date -> { date, label, minimax, mistral, gemini }
    const grouped: Record<string, Record<string, number>> = {}

    // Initialize last 30 days
    for (let i = 29; i >= 0; i--) {
      const d = new Date()
      d.setDate(d.getDate() - i)
      const key = d.toISOString().split('T')[0]
      grouped[key] = {}
    }

    for (const entry of data.daily) {
      if (grouped[entry.date]) {
        grouped[entry.date][entry.provider] =
          (grouped[entry.date][entry.provider] || 0) + entry.calls
      }
    }

    return Object.entries(grouped).map(([date, providers]) => ({
      date,
      label: new Date(date + 'T00:00:00').toLocaleDateString('en', {
        month: 'short',
        day: 'numeric'
      }),
      ...providers
    }))
  }, [data])

  const activeProviders = useMemo(() => {
    if (!data?.daily) return []
    const set = new Set(data.daily.map((d) => d.provider))
    return Array.from(set)
  }, [data])

  const visionUsed = data?.rate_limits?.vision?.used ?? 0
  const visionLimit = data?.rate_limits?.vision?.limit ?? 20

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="AI Usage" description="Monitor AI API usage across all providers" />

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => <StatsCardSkeleton key={i} />)
        ) : (
          <>
            <StatsCard
              title="Today's Calls"
              value={todayCalls}
              description={`${formatTokens(todayTokens)} tokens used`}
              icon={Zap}
            />
            <StatsCard
              title="Monthly Calls"
              value={monthCalls}
              description={`Since ${new Date().toLocaleDateString('en', { month: 'long', day: 'numeric' })}`}
              icon={Brain}
            />
            <StatsCard
              title="Total Tokens"
              value={formatTokens(data?.totals?.total_tokens ?? 0)}
              description={`${formatTokens(data?.totals?.prompt_tokens ?? 0)} prompt + ${formatTokens(data?.totals?.completion_tokens ?? 0)} completion`}
              icon={MessageSquare}
            />
            <StatsCard
              title="Vision Today"
              value={`${visionUsed}/${visionLimit}`}
              description={
                visionUsed >= visionLimit
                  ? 'Daily limit reached'
                  : `${visionLimit - visionUsed} remaining`
              }
              icon={Eye}
            />
          </>
        )}
      </div>

      {/* Daily Activity Chart */}
      <DataCard title="Daily AI Calls (30 days)">
        {isLoading ? (
          <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
            Loading...
          </div>
        ) : chartData.length === 0 ? (
          <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
            No usage data yet
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={280}>
            <BarChart data={chartData}>
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
              <Legend />
              {activeProviders.map((provider) => (
                <Bar
                  key={provider}
                  dataKey={provider}
                  name={PROVIDER_LABELS[provider] || provider}
                  fill={PROVIDER_COLORS[provider] || '#888'}
                  stackId="calls"
                  radius={[2, 2, 0, 0]}
                />
              ))}
            </BarChart>
          </ResponsiveContainer>
        )}
      </DataCard>

      {/* Provider Breakdown */}
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <DataCard title="Today's Breakdown">
          {isLoading ? (
            <div className="text-pdt-neutral/40">Loading...</div>
          ) : !data?.today?.length ? (
            <div className="text-pdt-neutral/40">No usage today</div>
          ) : (
            <div className="space-y-3">
              {data.today.map((item, i) => (
                <div
                  key={i}
                  className="flex items-center justify-between border-b border-pdt-neutral/10 py-2 last:border-0"
                >
                  <div className="flex items-center gap-3">
                    <div
                      className="size-3 rounded-full"
                      style={{
                        backgroundColor:
                          PROVIDER_COLORS[item.provider] || '#888'
                      }}
                    />
                    <div>
                      <p className="text-sm font-medium text-pdt-neutral">
                        {PROVIDER_LABELS[item.provider] || item.provider}
                      </p>
                      <p className="text-xs text-pdt-neutral/50">
                        {FEATURE_LABELS[item.feature] || item.feature}
                        {item.model ? ` — ${item.model}` : ''}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-medium text-pdt-neutral">
                      {item.calls} calls
                    </p>
                    <p className="text-xs text-pdt-neutral/50">
                      {formatTokens(item.prompt_tokens + item.completion_tokens)}{' '}
                      tokens
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </DataCard>

        <DataCard title="Monthly Breakdown">
          {isLoading ? (
            <div className="text-pdt-neutral/40">Loading...</div>
          ) : !data?.month?.length ? (
            <div className="text-pdt-neutral/40">No usage this month</div>
          ) : (
            <div className="space-y-3">
              {data.month.map((item, i) => (
                <div
                  key={i}
                  className="flex items-center justify-between border-b border-pdt-neutral/10 py-2 last:border-0"
                >
                  <div className="flex items-center gap-3">
                    <div
                      className="size-3 rounded-full"
                      style={{
                        backgroundColor:
                          PROVIDER_COLORS[item.provider] || '#888'
                      }}
                    />
                    <div>
                      <p className="text-sm font-medium text-pdt-neutral">
                        {PROVIDER_LABELS[item.provider] || item.provider}
                      </p>
                      <p className="text-xs text-pdt-neutral/50">
                        {FEATURE_LABELS[item.feature] || item.feature}
                        {item.model ? ` — ${item.model}` : ''}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-medium text-pdt-neutral">
                      {item.calls} calls
                    </p>
                    <p className="text-xs text-pdt-neutral/50">
                      {formatTokens(item.prompt_tokens + item.completion_tokens)}{' '}
                      tokens
                    </p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </DataCard>
      </div>
    </div>
  )
}

export default AIUsagePage
