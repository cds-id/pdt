import { GitCommit, Link2, Trello, RefreshCw } from 'lucide-react'

import { useListCommitsQuery } from '@/infrastructure/services/commit.service'
import { useGetActiveSprintQuery } from '@/infrastructure/services/jira.service'
import { useGetSyncStatusQuery, useTriggerSyncMutation } from '@/infrastructure/services/sync.service'
import { useGetProfileQuery } from '@/infrastructure/services/user.service'
import { Button } from '@/components/ui/button'
import { StatsCard, StatsCardSkeleton } from '@/presentation/components/dashboard'
import { PageHeader, DataCard, StatusBadge } from '@/presentation/components/common'

export function DashboardHomePage() {
  const { data: profile, isLoading: profileLoading } = useGetProfileQuery()
  const { data: commitsData, isLoading: commitsLoading } = useListCommitsQuery()
  const { data: activeSprint, isLoading: sprintLoading } = useGetActiveSprintQuery()
  const { data: syncStatus } = useGetSyncStatusQuery()
  const [triggerSync, { isLoading: isSyncing }] = useTriggerSyncMutation()

  const totalCommits = commitsData?.total || 0
  const commits = commitsData?.commits || []
  const linkedCommits = commits.filter((c) => (c as any).hasLink || (c as any).jiraCardKey).length
  const linkedPercent = totalCommits > 0 ? Math.round((linkedCommits / totalCommits) * 100) : 0
  const activeSprintCards = activeSprint?.cards?.length || 0

  const isLoading = profileLoading || commitsLoading || sprintLoading

  const stats = [
    {
      title: 'Total Commits (30d)',
      value: totalCommits,
      description: syncStatus?.lastSyncAt
        ? `Last sync: ${new Date(syncStatus.lastSyncAt).toLocaleString()}`
        : 'No sync yet',
      icon: GitCommit
    },
    {
      title: 'Linked to Jira',
      value: linkedCommits,
      description: `${linkedPercent}% linked`,
      icon: Link2
    },
    {
      title: 'Active Sprint',
      value: activeSprintCards,
      description: activeSprint?.name || 'No active sprint',
      icon: Trello
    }
  ]

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      {/* Welcome */}
      <PageHeader
        title="Welcome back"
        description={profile?.email || 'Loading...'}
        action={
          <Button onClick={() => triggerSync()} disabled={isSyncing} variant="pdt">
            <RefreshCw className={`mr-2 h-4 w-4 ${isSyncing ? 'animate-spin' : ''}`} />
            {isSyncing ? 'Syncing...' : 'Sync Now'}
          </Button>
        }
      />

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {isLoading
          ? Array.from({ length: 3 }).map((_, i) => <StatsCardSkeleton key={i} />)
          : stats.map((stat) => (
              <StatsCard
                key={stat.title}
                title={stat.title}
                value={stat.value}
                description={stat.description}
                icon={stat.icon}
              />
            ))}
      </div>

      {/* Recent Commits */}
      <DataCard title="Recent Commits">
        {commitsLoading ? (
          <p className="text-pdt-neutral/60">Loading...</p>
        ) : commits.length === 0 ? (
          <p className="text-pdt-neutral/60">No commits yet. Add a repository to get started.</p>
        ) : (
          <div className="space-y-0">
            {commits.slice(0, 5).map((commit) => (
              <div
                key={commit.id}
                className="flex items-center justify-between border-b border-pdt-neutral/10 py-3 last:border-0"
              >
                <div className="flex-1 min-w-0">
                  <p className="truncate text-pdt-neutral">{commit.message}</p>
                  <p className="text-sm text-pdt-neutral/50">
                    {commit.sha.slice(0, 7)} &middot;{' '}
                    {new Date(commit.date).toLocaleDateString()}
                  </p>
                </div>
                {(commit as any).jiraCardKey && (
                  <StatusBadge variant="warning">{(commit as any).jiraCardKey}</StatusBadge>
                )}
              </div>
            ))}
          </div>
        )}
      </DataCard>
    </div>
  )
}

export default DashboardHomePage
