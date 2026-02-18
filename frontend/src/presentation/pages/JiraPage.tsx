import { useListSprintsQuery, useGetActiveSprintQuery, useListCardsQuery } from '@/infrastructure/services/jira.service'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function JiraPage() {
  const { data: sprints, isLoading: sprintsLoading } = useListSprintsQuery()
  const { data: activeSprint, isLoading: sprintLoading } = useGetActiveSprintQuery()
  const { data: cardsData } = useListCardsQuery(activeSprint?.id)

  const cards = cardsData?.cards || []

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <h1 className="text-2xl font-bold text-[#FBFFFE] md:text-3xl">Jira</h1>

      {/* Active Sprint */}
      {sprintLoading ? (
        <p className="text-[#FBFFFE]/60">Loading...</p>
      ) : activeSprint ? (
        <div className="rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-4">
          <div className="mb-4 flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold text-[#FBFFFE]">{activeSprint.name}</h2>
              <p className="text-sm text-[#FBFFFE]/60">
                {activeSprint.startDate && new Date(activeSprint.startDate).toLocaleDateString()} -{' '}
                {activeSprint.endDate && new Date(activeSprint.endDate).toLocaleDateString()}
              </p>
            </div>
            <span className="rounded bg-green-500/20 px-3 py-1 text-sm text-green-400">
              {activeSprint.state}
            </span>
          </div>

          {/* Cards Grid */}
          {cards.length === 0 ? (
            <p className="text-[#FBFFFE]/60">No cards in this sprint.</p>
          ) : (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
              {cards.map((card) => (
                <Card key={card.key} className="border-[#F8C630]/20 bg-[#F8C630]/5">
                  <CardHeader className="pb-2 pt-4">
                    <div className="flex items-start justify-between">
                      <span className="font-semibold text-[#F8C630]">{card.key}</span>
                      <span
                        className={`rounded px-2 py-0.5 text-xs ${
                          card.status === 'Done'
                            ? 'bg-green-500/20 text-green-400'
                            : card.status === 'In Progress'
                            ? 'bg-blue-500/20 text-blue-400'
                            : 'bg-gray-500/20 text-gray-400'
                        }`}
                      >
                        {card.status}
                      </span>
                    </div>
                  </CardHeader>
                  <CardContent className="pb-4">
                    <p className="mb-2 text-sm text-[#FBFFFE]">{card.summary}</p>
                    {card.assignee && (
                      <p className="text-xs text-[#FBFFFE]/60">Assignee: {card.assignee}</p>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      ) : (
        <div className="rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-8 text-center">
          <p className="text-[#FBFFFE]/60">No active sprint found.</p>
          <p className="mt-2 text-sm text-[#FBFFFE]/40">
            Configure your Jira integration in Settings.
          </p>
        </div>
      )}

      {/* All Sprints */}
      <div>
        <h2 className="mb-4 text-lg font-semibold text-[#FBFFFE]">All Sprints</h2>
        {sprintsLoading ? (
          <p className="text-[#FBFFFE]/60">Loading...</p>
        ) : sprints?.length === 0 ? (
          <p className="text-[#FBFFFE]/60">No sprints found.</p>
        ) : (
          <div className="space-y-2">
            {sprints?.map((sprint) => (
              <div
                key={sprint.id}
                className="flex items-center justify-between rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-4"
              >
                <div>
                  <p className="font-medium text-[#FBFFFE]">{sprint.name}</p>
                  <p className="text-sm text-[#FBFFFE]/60">
                    {sprint.startDate && new Date(sprint.startDate).toLocaleDateString()}
                  </p>
                </div>
                <span
                  className={`rounded px-2 py-1 text-xs ${
                    sprint.state === 'active'
                      ? 'bg-green-500/20 text-green-400'
                      : sprint.state === 'closed'
                      ? 'bg-gray-500/20 text-gray-400'
                      : 'bg-blue-500/20 text-blue-400'
                  }`}
                >
                  {sprint.state}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default JiraPage
