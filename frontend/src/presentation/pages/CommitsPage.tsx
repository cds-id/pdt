import { useState } from 'react'
import { Link2, Filter } from 'lucide-react'

import { useListCommitsQuery, useGetMissingCommitsQuery, useLinkToJiraMutation } from '@/infrastructure/services/commit.service'
import { useListReposQuery } from '@/infrastructure/services/repo.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function CommitsPage() {
  const [repoId, setRepoId] = useState<string>('')
  const [jiraKey, setJiraKey] = useState('')
  const [showUnlinked, setShowUnlinked] = useState(false)
  const [linkKeyInput, setLinkKeyInput] = useState<{ [key: string]: string }>({})

  const filters = {
    ...(repoId && { repoId }),
    ...(jiraKey && { jiraKey })
  }

  const { data: commitsData, isLoading: commitsLoading } = useListCommitsQuery(filters)
  const { data: missingCommits, isLoading: missingLoading } = useGetMissingCommitsQuery(undefined, { skip: !showUnlinked })
  const { data: repos } = useListReposQuery()
  const [linkToJira, { isLoading: linking }] = useLinkToJiraMutation()

  const displayCommits = showUnlinked ? missingCommits : commitsData?.commits
  const isLoading = showUnlinked ? missingLoading : commitsLoading

  const handleLink = async (sha: string) => {
    const key = linkKeyInput[sha]
    if (!key?.trim()) return
    try {
      await linkToJira({ sha, jiraKey: key }).unwrap()
      setLinkKeyInput((prev) => ({ ...prev, [sha]: '' }))
    } catch (error) {
      console.error('Failed to link commit:', error)
    }
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <h1 className="text-2xl font-bold text-[#FBFFFE] md:text-3xl">Commits</h1>

      {/* Filters */}
      <div className="flex flex-wrap gap-4">
        <select
          value={repoId}
          onChange={(e) => setRepoId(e.target.value)}
          className="rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] px-4 py-2 text-[#FBFFFE]"
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
          className="w-40 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
        />

        <Button
          variant={showUnlinked ? 'default' : 'outline'}
          onClick={() => setShowUnlinked(!showUnlinked)}
          className={showUnlinked ? 'bg-[#F8C630] text-[#1B1B1E]' : 'border-[#F8C630] text-[#F8C630]'}
        >
          <Filter className="mr-2 h-4 w-4" />
          Show Unlinked Only
        </Button>
      </div>

      {/* Commits Table */}
      <div className="overflow-hidden rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E]">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-[#F8C630]/10">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-semibold text-[#FBFFFE]">SHA</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-[#FBFFFE]">Message</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-[#FBFFFE]">Author</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-[#FBFFFE]">Date</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-[#FBFFFE]">Jira</th>
                <th className="px-4 py-3 text-left text-sm font-semibold text-[#FBFFFE]">Actions</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-[#FBFFFE]/60">
                    Loading...
                  </td>
                </tr>
              ) : displayCommits?.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-[#FBFFFE]/60">
                    No commits found.
                  </td>
                </tr>
              ) : (
                displayCommits?.map((commit) => (
                  <tr key={commit.id} className="border-t border-[#FBFFFE]/10">
                    <td className="px-4 py-3">
                      <code className="text-sm text-[#F8C630]">{commit.sha.slice(0, 7)}</code>
                    </td>
                    <td className="max-w-xs truncate px-4 py-3 text-[#FBFFFE]">
                      {commit.message}
                    </td>
                    <td className="px-4 py-3 text-sm text-[#FBFFFE]/60">{commit.author}</td>
                    <td className="px-4 py-3 text-sm text-[#FBFFFE]/60">
                      {new Date(commit.date).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      {(commit as any).jiraCardKey ? (
                        <span className="rounded bg-[#F8C630]/20 px-2 py-1 text-xs text-[#F8C630]">
                          {(commit as any).jiraCardKey}
                        </span>
                      ) : (
                        <span className="text-sm text-[#FBFFFE]/40">-</span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {(commit as any).jiraCardKey ? (
                        <span className="text-xs text-[#FBFFFE]/40">Linked</span>
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
                            className="h-8 w-28 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
                          />
                          <Button
                            size="sm"
                            onClick={() => handleLink(commit.sha)}
                            disabled={linking || !linkKeyInput[commit.sha]?.trim()}
                            className="h-8 bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90"
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
      </div>
    </div>
  )
}

export default CommitsPage
