import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, GitCommit, ListTree } from 'lucide-react'

import { useGetCardQuery } from '@/infrastructure/services/jira.service'
import { useListCommitsQuery } from '@/infrastructure/services/commit.service'
import { Button } from '@/components/ui/button'
import { PageHeader, DataCard, StatusBadge, EmptyState } from '@/presentation/components/common'

export function JiraCardDetailPage() {
  const { key } = useParams<{ key: string }>()
  const { data: card, isLoading, error } = useGetCardQuery(key!, { skip: !key })
  const { data: allCommits = [] } = useListCommitsQuery({ jira_card_key: key })

  if (isLoading) {
    return <p className="text-pdt-neutral/60">Loading card details...</p>
  }

  if (error || !card) {
    return (
      <EmptyState
        title="Card not found"
        description={`Could not find Jira card ${key}`}
      />
    )
  }

  let subtasks: { key: string; summary: string; status: string }[] = []
  if (card.details_json) {
    try {
      const details = JSON.parse(card.details_json)
      subtasks = details.subtasks || []
    } catch {
      // ignore parse errors
    }
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <div className="flex items-center gap-3">
        <Link to="/dashboard/jira">
          <Button variant="pdtOutline" size="sm">
            <ArrowLeft className="mr-1 h-4 w-4" /> Back
          </Button>
        </Link>
      </div>

      <PageHeader title={card.key} description={card.summary} />

      {/* Card Info */}
      <DataCard>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <p className="text-xs text-pdt-neutral/50">Status</p>
            <StatusBadge
              variant={
                card.status === 'Done' ? 'success'
                  : card.status === 'In Progress' ? 'warning'
                  : 'neutral'
              }
            >
              {card.status}
            </StatusBadge>
          </div>
          <div>
            <p className="text-xs text-pdt-neutral/50">Assignee</p>
            <p className="text-sm text-pdt-neutral">{card.assignee || 'Unassigned'}</p>
          </div>
          <div>
            <p className="text-xs text-pdt-neutral/50">Sprint</p>
            <p className="text-sm text-pdt-neutral">{card.sprint_id ? `Sprint #${card.sprint_id}` : 'No sprint'}</p>
          </div>
        </div>
      </DataCard>

      {/* Linked Commits */}
      <DataCard title="Linked Commits">
        {allCommits.length === 0 ? (
          <EmptyState title="No linked commits" description="No commits reference this card." />
        ) : (
          <div className="space-y-0">
            {allCommits.map((commit) => (
              <div
                key={commit.id}
                className="flex items-center gap-3 border-b border-pdt-neutral/10 py-3 last:border-0"
              >
                <GitCommit className="h-4 w-4 shrink-0 text-pdt-accent" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm text-pdt-neutral">{commit.message}</p>
                  <p className="text-xs text-pdt-neutral/50">
                    <code className="text-pdt-accent">{commit.sha.slice(0, 7)}</code>
                    {' '}&middot; {commit.author} &middot; {new Date(commit.date).toLocaleDateString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </DataCard>

      {/* Subtasks */}
      {subtasks.length > 0 && (
        <DataCard title="Subtasks">
          <div className="space-y-2">
            {subtasks.map((sub) => (
              <div
                key={sub.key}
                className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-3"
              >
                <div className="flex items-center gap-2">
                  <ListTree className="h-4 w-4 text-pdt-accent" />
                  <span className="text-sm font-medium text-pdt-accent">{sub.key}</span>
                  <span className="text-sm text-pdt-neutral">{sub.summary}</span>
                </div>
                <StatusBadge
                  variant={sub.status === 'Done' ? 'success' : sub.status === 'In Progress' ? 'warning' : 'neutral'}
                >
                  {sub.status}
                </StatusBadge>
              </div>
            ))}
          </div>
        </DataCard>
      )}
    </div>
  )
}

export default JiraCardDetailPage
