import { GitCommit, Link2, Trello, RefreshCw, GitBranch, RotateCcw, Cloud } from 'lucide-react'

import { useListCommitsQuery } from '@/infrastructure/services/commit.service'
import {
  useGetActiveSprintQuery,
  useListCardsQuery,
  useListSprintsQuery
} from '@/infrastructure/services/jira.service'
import { useGetSyncStatusQuery, useTriggerSyncMutation } from '@/infrastructure/services/sync.service'
import { useGetProfileQuery } from '@/infrastructure/services/user.service'
import { useListReposQuery } from '@/infrastructure/services/repo.service'
import { Button } from '@/components/ui/button'
import { StatsCard, StatsCardSkeleton } from '@/presentation/components/dashboard'
import { PageHeader, DataCard, StatusBadge } from '@/presentation/components/common'
import {
  CommitActivityChart,
  CardStatusChart,
  LinkageGaugeChart,
  SprintVelocityChart
} from '@/presentation/components/charts'

export function DashboardHomePage() {
  const { data: profile, isLoading: profileLoading } = useGetProfileQuery()
  const { data: commitsData, isLoading: commitsLoading } = useListCommitsQuery()
  const { data: activeSprint, isLoading: sprintLoading } = useGetActiveSprintQuery()
  const { data: syncStatus } = useGetSyncStatusQuery()
  const [triggerSync, { isLoading: isSyncing }] = useTriggerSyncMutation()
  const { data: reposData, isLoading: reposLoading } = useListReposQuery()
  const { data: cardsData, isLoading: cardsLoading } = useListCardsQuery()
  const { data: sprintsData, isLoading: sprintsLoading } = useListSprintsQuery()

  const commits = commitsData || []
  const cards = cardsData || []
  const sprints = sprintsData || []
  const repos = reposData || []
  const totalCommits = commits.length
  const linkedCommits = commits.filter((c) => c.has_link || c.jira_card_key).length
  const linkedPercent = totalCommits > 0 ? Math.round((linkedCommits / totalCommits) * 100) : 0
  const activeSprintCards = activeSprint?.cards?.length || 0

  const isLoading = profileLoading || commitsLoading || sprintLoading || reposLoading

  const stats = [
    {
      title: 'Total Commits (30d)',
      value: totalCommits,
      description: syncStatus?.commits?.last_sync
        ? `Last sync: ${new Date(syncStatus.commits.last_sync).toLocaleString()}`
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
    },
    {
      title: 'Repositories',
      value: repos.length,
      description: `${repos.filter((r) => r.is_valid).length} active`,
      icon: GitBranch
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
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {isLoading
          ? Array.from({ length: 4 }).map((_, i) => <StatsCardSkeleton key={i} />)
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

      {/* Charts Row 1 */}
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <DataCard title="Commit Activity (30 days)">
          {commitsLoading ? (
            <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
              Loading...
            </div>
          ) : (
            <CommitActivityChart commits={commits} />
          )}
        </DataCard>

        <DataCard title="Card Status Breakdown">
          {cardsLoading ? (
            <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
              Loading...
            </div>
          ) : (
            <CardStatusChart cards={cards} />
          )}
        </DataCard>
      </div>

      {/* Charts Row 2 */}
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        <DataCard title="Jira Linkage">
          <LinkageGaugeChart linked={linkedCommits} total={totalCommits} />
        </DataCard>

        <DataCard title="Sprint Velocity">
          {sprintsLoading ? (
            <div className="flex h-[280px] items-center justify-center text-pdt-neutral/40">
              Loading...
            </div>
          ) : (
            <SprintVelocityChart sprints={sprints} />
          )}
        </DataCard>
      </div>

      {/* Sync Status */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <DataCard>
          <div className="flex items-center gap-4">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-pdt-accent/20">
              <RotateCcw className="h-5 w-5 text-pdt-accent" />
            </div>
            <div>
              <p className="text-sm font-medium text-pdt-neutral">Commit Sync</p>
              <p className="text-xs text-pdt-neutral/50">
                {syncStatus?.commits?.last_sync
                  ? `Last synced: ${new Date(syncStatus.commits.last_sync).toLocaleString()}`
                  : 'Never synced'}
              </p>
            </div>
          </div>
        </DataCard>

        <DataCard>
          <div className="flex items-center gap-4">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-pdt-accent/20">
              <Cloud className="h-5 w-5 text-pdt-accent" />
            </div>
            <div>
              <p className="text-sm font-medium text-pdt-neutral">Jira Sync</p>
              <p className="text-xs text-pdt-neutral/50">
                {syncStatus?.jira?.last_sync
                  ? `Last synced: ${new Date(syncStatus.jira.last_sync).toLocaleString()}`
                  : 'Never synced'}
              </p>
            </div>
          </div>
        </DataCard>
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
                {commit.jira_card_key && (
                  <StatusBadge variant="warning">{commit.jira_card_key}</StatusBadge>
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
