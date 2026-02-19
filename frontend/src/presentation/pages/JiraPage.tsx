import { useListSprintsQuery, useGetActiveSprintQuery, useListCardsQuery } from '@/infrastructure/services/jira.service'
import { PageHeader, DataCard, StatusBadge, EmptyState } from '@/presentation/components/common'

export function JiraPage() {
  const { data: sprints, isLoading: sprintsLoading } = useListSprintsQuery()
  const { data: activeSprint, isLoading: sprintLoading } = useGetActiveSprintQuery()
  const { data: cards = [] } = useListCardsQuery(activeSprint?.id)

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
              <h2 className="text-lg font-semibold text-pdt-neutral">{activeSprint.name}</h2>
              <p className="text-sm text-pdt-neutral/60">
                {activeSprint.start_date && new Date(activeSprint.start_date).toLocaleDateString()} -{' '}
                {activeSprint.end_date && new Date(activeSprint.end_date).toLocaleDateString()}
              </p>
            </div>
            <StatusBadge variant="success">{activeSprint.state}</StatusBadge>
          </div>

          {/* Cards Grid */}
          {cards.length === 0 ? (
            <p className="text-pdt-neutral/60">No cards in this sprint.</p>
          ) : (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
              {cards.map((card) => (
                <div key={card.key} className="rounded-lg border border-pdt-background/20 bg-pdt-background/5 p-4">
                  <div className="mb-2 flex items-start justify-between">
                    <span className="font-semibold text-pdt-background">{card.key}</span>
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
                  <p className="mb-2 text-sm text-pdt-neutral">{card.summary}</p>
                  {card.assignee && (
                    <p className="text-xs text-pdt-neutral/60">Assignee: {card.assignee}</p>
                  )}
                </div>
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
                    {sprint.start_date && new Date(sprint.start_date).toLocaleDateString()}
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
