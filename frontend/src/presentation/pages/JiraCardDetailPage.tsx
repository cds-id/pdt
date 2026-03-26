import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, GitCommit, ListTree, MessageSquare } from 'lucide-react'

import { useGetCardQuery, useGetCardCommentsQuery } from '@/infrastructure/services/jira.service'
import { useListCommitsQuery } from '@/infrastructure/services/commit.service'
import { Button } from '@/components/ui/button'
import {
  PageHeader,
  DataCard,
  StatusBadge,
  EmptyState
} from '@/presentation/components/common'
import { MessageResponse } from '@/components/ai-elements/message'
import { jiraToMarkdown } from '@/lib/jira-markup'

export function JiraCardDetailPage() {
  const { key } = useParams<{ key: string }>()
  const { data: card, isLoading, error } = useGetCardQuery(key!, { skip: !key })
  const { data: allCommits = [] } = useListCommitsQuery({ jira_card_key: key })
  const { data: comments = [] } = useGetCardCommentsQuery(key!, { skip: !key })

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

  let subtasks: { key: string; summary: string; status: string; type: string }[] = (card as any).subtasks || []
  let description = (card as any).description || ''
  let parent: { key: string; summary: string; status: string; type: string } | null = (card as any).parent || null
  let issueType = (card as any).issue_type || ''

  // Fallback to details_json if direct fields are missing
  if (!description && card.details_json) {
    try {
      const details = JSON.parse(card.details_json)
      subtasks = subtasks.length ? subtasks : (details.subtasks || [])
      description = details.description || ''
      issueType = issueType || details.issue_type || ''
      if (!parent && details.parent) {
        parent = details.parent
      }
    } catch {
      // ignore
    }
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <div className="flex items-center gap-3">
        <Link to="/dashboard/jira">
          <Button variant="pdtOutline" size="sm">
            <ArrowLeft className="mr-1 size-4" /> Back
          </Button>
        </Link>
      </div>

      <PageHeader title={card.key} description={card.summary} />

      {/* Card Info */}
      <DataCard>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
          <div>
            <p className="text-xs text-pdt-neutral/50">Status</p>
            <StatusBadge
              variant={
                card.status === 'Done'
                  ? 'success'
                  : card.status === 'In Progress'
                    ? 'warning'
                    : 'neutral'
              }
            >
              {card.status}
            </StatusBadge>
          </div>
          <div>
            <p className="text-xs text-pdt-neutral/50">Assignee</p>
            <p className="text-sm text-pdt-neutral">
              {card.assignee || 'Unassigned'}
            </p>
          </div>
          <div>
            <p className="text-xs text-pdt-neutral/50">Sprint</p>
            <p className="text-sm text-pdt-neutral">
              {card.sprint_id ? `Sprint #${card.sprint_id}` : 'No sprint'}
            </p>
          </div>
          <div>
            <p className="text-xs text-pdt-neutral/50">Type</p>
            <p className="text-sm text-pdt-neutral">{issueType || 'Unknown'}</p>
          </div>
        </div>
      </DataCard>

      {/* Description */}
      {description && (
        <DataCard title="Description">
          <div className="text-sm overflow-hidden">
            <MessageResponse>{jiraToMarkdown(description)}</MessageResponse>
          </div>
        </DataCard>
      )}

      {/* Linked Commits */}
      <DataCard title="Linked Commits">
        {allCommits.length === 0 ? (
          <EmptyState
            title="No linked commits"
            description="No commits reference this card."
          />
        ) : (
          <div className="space-y-0">
            {allCommits.map((commit) => (
              <div
                key={commit.id}
                className="flex items-center gap-3 border-b border-pdt-neutral/10 py-3 last:border-0"
              >
                <GitCommit className="size-4 shrink-0 text-pdt-accent" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm text-pdt-neutral">
                    {commit.message}
                  </p>
                  <p className="text-xs text-pdt-neutral/50">
                    <code className="text-pdt-accent">
                      {commit.sha.slice(0, 7)}
                    </code>{' '}
                    &middot; {commit.author} &middot;{' '}
                    {new Date(commit.date).toLocaleDateString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </DataCard>

      {/* Parent Card */}
      {parent && (
        <DataCard title="Parent">
          <div className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-3">
            <div className="flex items-center gap-2">
              <Link to={`/dashboard/jira/${parent.key}`} className="text-sm font-medium text-pdt-accent hover:underline">
                {parent.key}
              </Link>
              <span className="text-sm text-pdt-neutral">{parent.summary}</span>
            </div>
            <StatusBadge variant={parent.status === 'Done' ? 'success' : 'neutral'}>
              {parent.status}
            </StatusBadge>
          </div>
        </DataCard>
      )}

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
                  <ListTree className="size-4 text-pdt-accent" />
                  <span className="text-sm font-medium text-pdt-accent">
                    {sub.key}
                  </span>
                  <span className="text-sm text-pdt-neutral">
                    {sub.summary}
                  </span>
                </div>
                <StatusBadge
                  variant={
                    sub.status === 'Done'
                      ? 'success'
                      : sub.status === 'In Progress'
                        ? 'warning'
                        : 'neutral'
                  }
                >
                  {sub.status}
                </StatusBadge>
              </div>
            ))}
          </div>
        </DataCard>
      )}

      {/* Comments */}
      <DataCard title={`Comments (${comments.length})`}>
        {comments.length === 0 ? (
          <EmptyState title="No comments" description="No comments on this card yet." />
        ) : (
          <div className="space-y-4">
            {comments.map((comment) => (
              <div key={comment.id} className="border-b border-pdt-neutral/10 pb-4 last:border-0 last:pb-0">
                <div className="flex items-center gap-2 mb-1">
                  <MessageSquare className="size-3 text-pdt-accent" />
                  <span className="text-sm font-medium text-pdt-accent">{comment.author}</span>
                  <span className="text-xs text-pdt-neutral/50">
                    {new Date(comment.commented_at).toLocaleString()}
                  </span>
                </div>
                <div className="text-sm pl-5 overflow-hidden">
                  <MessageResponse>{jiraToMarkdown(comment.body)}</MessageResponse>
                </div>
              </div>
            ))}
          </div>
        )}
      </DataCard>
    </div>
  )
}

export default JiraCardDetailPage
