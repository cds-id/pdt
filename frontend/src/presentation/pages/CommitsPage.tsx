import { useState } from 'react'
import { Link2, Filter } from 'lucide-react'

import { useListCommitsQuery, useGetMissingCommitsQuery, useLinkToJiraMutation } from '@/infrastructure/services/commit.service'
import { useListReposQuery } from '@/infrastructure/services/repo.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PageHeader, DataCard, StatusBadge, FilterBar } from '@/presentation/components/common'

export function CommitsPage() {
  const [repoId, setRepoId] = useState<string>('')
  const [jiraKey, setJiraKey] = useState('')
  const [showUnlinked, setShowUnlinked] = useState(false)
  const [linkKeyInput, setLinkKeyInput] = useState<{ [key: string]: string }>({})

  const filters = {
    ...(repoId && { repo_id: repoId }),
    ...(jiraKey && { jira_card_key: jiraKey })
  }

  const { data: commitsData, isLoading: commitsLoading } = useListCommitsQuery(filters)
  const { data: missingCommits, isLoading: missingLoading } = useGetMissingCommitsQuery(undefined, { skip: !showUnlinked })
  const { data: repos } = useListReposQuery()
  const [linkToJira, { isLoading: linking }] = useLinkToJiraMutation()

  const displayCommits = showUnlinked ? missingCommits : commitsData
  const isLoading = showUnlinked ? missingLoading : commitsLoading

  const handleLink = async (sha: string) => {
    const key = linkKeyInput[sha]
    if (!key?.trim()) return
    try {
      await linkToJira({ sha, jira_card_key: key }).unwrap()
      setLinkKeyInput((prev) => ({ ...prev, [sha]: '' }))
    } catch (error) {
      console.error('Failed to link commit:', error)
    }
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="Commits" />

      {/* Filters */}
      <FilterBar>
        <select
          value={repoId}
          onChange={(e) => setRepoId(e.target.value)}
          className="rounded-lg border bg-pdt-primary-light border-pdt-accent/20 px-4 py-2 text-pdt-neutral"
        >
          <option value="">All Repositories</option>
          {repos?.map((repo) => (
            <option key={repo.id} value={repo.id}>
              {repo.owner}/{repo.name}
            </option>
          ))}
        </select>

        <Input
          placeholder="Filter by Jira key..."
          value={jiraKey}
          onChange={(e) => setJiraKey(e.target.value)}
          className="w-40 bg-pdt-primary-light border-pdt-accent/20 text-pdt-neutral placeholder:text-pdt-neutral/40"
        />

        <Button
          variant={showUnlinked ? 'pdt' : 'pdtOutline'}
          onClick={() => setShowUnlinked(!showUnlinked)}
        >
          <Filter className="mr-2 h-4 w-4" />
          Show Unlinked Only
        </Button>
      </FilterBar>

      {/* Commits Table */}
      <DataCard className="overflow-hidden p-0">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-pdt-accent/10">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-semibold text-pdt-neutral">SHA</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-pdt-neutral">Message</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-pdt-neutral">Author</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-pdt-neutral">Date</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-pdt-neutral">Jira</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-pdt-neutral">Actions</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-pdt-neutral/60">
                    Loading...
                  </td>
                </tr>
              ) : displayCommits?.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-pdt-neutral/60">
                    No commits found.
                  </td>
                </tr>
              ) : (
                displayCommits?.map((commit) => (
                  <tr key={commit.id} className="border-t border-pdt-neutral/10">
                    <td className="px-4 py-3">
                      <code className="text-sm text-pdt-accent">{commit.sha.slice(0, 7)}</code>
                    </td>
                    <td className="max-w-xs truncate px-4 py-3 text-pdt-neutral">
                      {commit.message}
                    </td>
                    <td className="px-4 py-3 text-sm text-pdt-neutral/60">{commit.author}</td>
                    <td className="px-4 py-3 text-sm text-pdt-neutral/60">
                      {new Date(commit.date).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      {commit.jira_card_key ? (
                        <StatusBadge variant="warning">{commit.jira_card_key}</StatusBadge>
                      ) : (
                        <span className="text-sm text-pdt-neutral/40">-</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {commit.jira_card_key ? (
                        <span className="text-xs text-pdt-neutral/40">Linked</span>
                      ) : (
                        <div className="flex items-center gap-2">
                          <Input
                            placeholder="PROJ-123"
                            value={linkKeyInput[commit.sha] || ''}
                            onChange={(e) =>
                              setLinkKeyInput((prev) => ({
                                ...prev,
                                [commit.sha]: e.target.value
                              }))
                            }
                            className="h-8 w-28 bg-pdt-primary-light border-pdt-accent/20 text-pdt-neutral placeholder:text-pdt-neutral/40"
                          />
                          <Button
                            size="sm"
                            onClick={() => handleLink(commit.sha)}
                            disabled={linking || !linkKeyInput[commit.sha]?.trim()}
                            variant="pdt"
                            className="h-8"
                          >
                            <Link2 className="h-3 w-3" />
                          </Button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </DataCard>
    </div>
  )
}

export default CommitsPage
