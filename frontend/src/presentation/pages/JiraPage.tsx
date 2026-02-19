import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  useListSprintsQuery,
  useGetActiveSprintQuery,
  useListCardsQuery
} from '@/infrastructure/services/jira.service'
import {
  PageHeader,
  DataCard,
  StatusBadge,
  EmptyState
} from '@/presentation/components/common'
import { Button } from '@/components/ui/button'

export function JiraPage() {
  const { data: sprints, isLoading: sprintsLoading } = useListSprintsQuery()
  const { data: activeSprint, isLoading: sprintLoading } =
    useGetActiveSprintQuery()
  const { data: cards = [] } = useListCardsQuery(activeSprint?.id)

  const [statusFilter, setStatusFilter] = useState<string>('all')

  const filteredCards =
    statusFilter === 'all'
      ? cards
      : cards.filter((c) => c.status === statusFilter)

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="Jira Integration" />

      {/* Active Sprint */}
      {sprintLoading ? (
        <p className="text-pdt-neutral/60">Loading...</p>
      ) : activeSprint ? (
        <DataCard>
          <div className="mb-4 flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-pdt-neutral">
                {activeSprint.name}
              </h2>
              <p className="text-sm text-pdt-neutral/60">
                {activeSprint.start_date &&
                  new Date(activeSprint.start_date).toLocaleDateString()}{' '}
                -{' '}
                {activeSprint.end_date &&
                  new Date(activeSprint.end_date).toLocaleDateString()}
              </p>
            </div>
            <StatusBadge variant="success">{activeSprint.state}</StatusBadge>
          </div>

          {/* Status Filter Tabs */}
          <div className="mb-4 flex flex-wrap gap-2">
            {['all', 'To Do', 'In Progress', 'Done'].map((status) => (
              <Button
                key={status}
                variant={statusFilter === status ? 'pdt' : 'pdtOutline'}
                size="sm"
                onClick={() => setStatusFilter(status)}
              >
                {status === 'all' ? 'All' : status}
              </Button>
            ))}
          </div>

          {/* Cards Grid */}
          {filteredCards.length === 0 ? (
            <p className="text-pdt-neutral/60">No cards in this sprint.</p>
          ) : (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
              {filteredCards.map((card) => (
                <Link to={`/dashboard/jira/cards/${card.key}`} key={card.key}>
                  <div className="rounded-lg border border-pdt-accent/20 bg-pdt-accent/5 p-4 transition-colors hover:border-pdt-accent/40 hover:bg-pdt-accent/10">
                    <div className="mb-2 flex items-start justify-between">
                      <span className="font-semibold text-pdt-accent">
                        {card.key}
                      </span>
                      <StatusBadge
                        variant={
                          card.status === 'Done'
                            ? 'success'
                            : card.status === 'In Progress'
                              ? 'info'
                              : 'neutral'
                        }
                      >
                        {card.status}
                      </StatusBadge>
                    </div>
                    <p className="mb-2 text-sm text-pdt-neutral">
                      {card.summary}
                    </p>
                    {card.assignee && (
                      <p className="text-xs text-pdt-neutral/60">
                        Assignee: {card.assignee}
                      </p>
                    )}
                  </div>
                </Link>
              ))}
            </div>
          )}
        </DataCard>
      ) : (
        <EmptyState
          title="No active sprint found."
          description="Configure your Jira integration in Settings."
        />
      )}

      {/* All Sprints */}
      <DataCard title="All Sprints">
        {sprintsLoading ? (
          <p className="text-pdt-neutral/60">Loading...</p>
        ) : sprints?.length === 0 ? (
          <EmptyState title="No sprints found." />
        ) : (
          <div className="space-y-2">
            {sprints?.map((sprint) => (
              <div
                key={sprint.id}
                className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-4"
              >
                <div>
                  <p className="font-medium text-pdt-neutral">{sprint.name}</p>
                  <p className="text-sm text-pdt-neutral/60">
                    {sprint.start_date &&
                      new Date(sprint.start_date).toLocaleDateString()}
                  </p>
                </div>
                <StatusBadge
                  variant={
                    sprint.state === 'active'
                      ? 'success'
                      : sprint.state === 'closed'
                        ? 'neutral'
                        : 'info'
                  }
                >
                  {sprint.state}
                </StatusBadge>
              </div>
            ))}
          </div>
        )}
      </DataCard>
    </div>
  )
}

export default JiraPage
