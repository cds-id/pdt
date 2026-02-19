import { useState } from 'react'
import { Plus, Trash2, Github, Gitlab } from 'lucide-react'

import { useListReposQuery, useAddRepoMutation, useDeleteRepoMutation } from '@/infrastructure/services/repo.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PageHeader, DataCard, StatusBadge, EmptyState } from '@/presentation/components/common'

export function ReposPage() {
  const { data: repos, isLoading } = useListReposQuery()
  const [addRepo] = useAddRepoMutation()
  const [deleteRepo] = useDeleteRepoMutation()
  const [newRepoUrl, setNewRepoUrl] = useState('')
  const [isAdding, setIsAdding] = useState(false)

  const handleAddRepo = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newRepoUrl.trim()) return
    setIsAdding(true)
    try {
      await addRepo({ url: newRepoUrl }).unwrap()
      setNewRepoUrl('')
    } catch (error) {
      console.error('Failed to add repo:', error)
    } finally {
      setIsAdding(false)
    }
  }

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to remove this repository?')) {
      try {
        await deleteRepo(id).unwrap()
      } catch (error) {
        console.error('Failed to delete repo:', error)
      }
    }
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="Repositories" />

      {/* Add Repo Form */}
      <DataCard title="Add Repository">
        <form onSubmit={handleAddRepo} className="flex flex-col gap-2 sm:flex-row">
          <Input
            type="url"
            placeholder="https://github.com/owner/repo"
            value={newRepoUrl}
            onChange={(e) => setNewRepoUrl(e.target.value)}
            className="bg-pdt-primary-light border-pdt-accent/20 text-pdt-neutral placeholder:text-pdt-neutral/40"
          />
          <Button
            type="submit"
            disabled={isAdding || !newRepoUrl.trim()}
            variant="pdt"
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Repository
          </Button>
        </form>
      </DataCard>

      {/* Repo List */}
      {isLoading ? (
        <p className="text-pdt-neutral/60">Loading...</p>
      ) : repos?.length === 0 ? (
        <EmptyState
          title="No repositories tracked yet."
          description="Add a repository above to start tracking commits."
        />
      ) : (
        <div className="space-y-3">
          {repos?.map((repo) => (
            <DataCard key={repo.id}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  {/* Provider Icon */}
                  <div
                    className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                      repo.provider === 'github' ? 'bg-pdt-primary-light' : 'bg-[#FC6D26]'
                    }`}
                  >
                    {repo.provider === 'github' ? (
                      <Github className="h-5 w-5 text-pdt-neutral" />
                    ) : (
                      <Gitlab className="h-5 w-5 text-pdt-neutral" />
                    )}
                  </div>
                  <div>
                    <p className="font-semibold text-pdt-neutral">{repo.name}</p>
                    <p className="text-sm text-pdt-neutral/60">{repo.owner}</p>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  {repo.is_valid ? (
                    <StatusBadge variant="success">Valid</StatusBadge>
                  ) : (
                    <StatusBadge variant="danger">Invalid</StatusBadge>
                  )}
                  <button
                    onClick={() => handleDelete(String(repo.id))}
                    className="text-pdt-neutral/60 transition-colors hover:text-red-400"
                  >
                    <Trash2 className="h-5 w-5" />
                  </button>
                </div>
              </div>
            </DataCard>
          ))}
        </div>
      )}
    </div>
  )
}

export default ReposPage
