import { useState } from 'react'
import { Plus, Trash2, Github, Gitlab } from 'lucide-react'

import { useListReposQuery, useAddRepoMutation, useDeleteRepoMutation } from '@/infrastructure/services/repo.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

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
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-[#FBFFFE] md:text-3xl">Repositories</h1>
      </div>

      {/* Add Repo Form */}
      <form onSubmit={handleAddRepo} className="flex flex-col gap-2 sm:flex-row">
        <Input
          type="url"
          placeholder="https://github.com/owner/repo"
          value={newRepoUrl}
          onChange={(e) => setNewRepoUrl(e.target.value)}
          className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
        />
        <Button
          type="submit"
          disabled={isAdding || !newRepoUrl.trim()}
          className="bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90"
        >
          <Plus className="mr-2 h-4 w-4" />
          Add Repository
        </Button>
      </form>

      {/* Repo List */}
      {isLoading ? (
        <p className="text-[#FBFFFE]/60">Loading...</p>
      ) : repos?.length === 0 ? (
        <div className="rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-8 text-center">
          <p className="text-[#FBFFFE]/60">No repositories tracked yet.</p>
          <p className="mt-2 text-sm text-[#FBFFFE]/40">
            Add a repository above to start tracking commits.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {repos?.map((repo) => (
            <div
              key={repo.id}
              className="flex items-center justify-between rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-4"
            >
              <div className="flex items-center gap-4">
                {/* Provider Icon */}
                <div
                  className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                    repo.provider === 'github' ? 'bg-[#1B1B1E]' : 'bg-[#FC6D26]'
                  }`}
                >
                  {repo.provider === 'github' ? (
                    <Github className="h-5 w-5 text-[#FBFFFE]" />
                  ) : (
                    <Gitlab className="h-5 w-5 text-[#FBFFFE]" />
                  )}
                </div>
                <div>
                  <p className="font-semibold text-[#FBFFFE]">{repo.name}</p>
                  <p className="text-sm text-[#FBFFFE]/60">{repo.owner}</p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <span
                  className={`rounded px-2 py-1 text-xs ${
                    (repo as any).isValid
                      ? 'bg-green-500/20 text-green-400'
                      : 'bg-red-500/20 text-red-400'
                  }`}
                >
                  {(repo as any).isValid ? 'Valid' : 'Invalid'}
                </span>
                <button
                  onClick={() => handleDelete(repo.id)}
                  className="text-[#FBFFFE]/60 transition-colors hover:text-red-400"
                >
                  <Trash2 className="h-5 w-5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default ReposPage
