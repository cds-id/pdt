import { useState } from 'react'
import { Save, Plus, Trash2, Pencil, RefreshCw } from 'lucide-react'

import { ComposioSettings } from '@/presentation/components/settings/ComposioSettings'

import {
  useGetProfileQuery,
  useUpdateProfileMutation,
  useValidateIntegrationsMutation
} from '@/infrastructure/services/user.service'
import {
  useListWorkspacesQuery,
  useAddWorkspaceMutation,
  useUpdateWorkspaceMutation,
  useDeleteWorkspaceMutation
} from '@/infrastructure/services/jira.service'
import { useTriggerJiraSyncMutation } from '@/infrastructure/services/sync.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  PageHeader,
  DataCard,
  StatusBadge
} from '@/presentation/components/common'
import { cn } from '@/lib/utils'
import { NumberManager } from '@/presentation/components/whatsapp/NumberManager'

export function SettingsPage() {
  const { data: profile, isLoading } = useGetProfileQuery()
  const [updateProfile, { isLoading: isUpdating }] = useUpdateProfileMutation()
  const [validate, { isLoading: isValidating }] =
    useValidateIntegrationsMutation()

  const { data: workspaces = [] } = useListWorkspacesQuery()
  const [addWorkspace] = useAddWorkspaceMutation()
  const [updateWorkspace] = useUpdateWorkspaceMutation()
  const [deleteWorkspace] = useDeleteWorkspaceMutation()
  const [syncJira, { isLoading: isSyncingJira }] = useTriggerJiraSyncMutation()
  const [syncingWsId, setSyncingWsId] = useState<number | null>(null)

  const [formData, setFormData] = useState({
    github_token: '',
    gitlab_token: '',
    gitlab_url: 'https://gitlab.com',
    jira_email: '',
    jira_token: '',
    jira_username: ''
  })

  const [newWs, setNewWs] = useState({
    workspace: '',
    name: '',
    project_keys: ''
  })
  const [showAddWs, setShowAddWs] = useState(false)
  const [editingWsId, setEditingWsId] = useState<number | null>(null)
  const [editWs, setEditWs] = useState({ name: '', project_keys: '' })

  const [saveMessage, setSaveMessage] = useState<{
    type: 'success' | 'error'
    text: string
  } | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaveMessage(null)
    const data = Object.fromEntries(
      Object.entries(formData).filter(([, v]) => v.trim() !== '')
    )
    try {
      await updateProfile(data).unwrap()
      setSaveMessage({ type: 'success', text: 'Settings saved successfully!' })
    } catch (error) {
      console.error('Failed to save settings:', error)
      setSaveMessage({ type: 'error', text: 'Failed to save settings.' })
    }
  }

  const handleValidate = async () => {
    setSaveMessage(null)
    try {
      await validate().unwrap()
      setSaveMessage({
        type: 'success',
        text: 'All connections validated successfully!'
      })
    } catch (error) {
      console.error('Validation failed:', error)
      setSaveMessage({ type: 'error', text: 'Connection validation failed.' })
    }
  }

  const handleAddWorkspace = async () => {
    if (!newWs.workspace) return
    try {
      await addWorkspace({
        workspace: newWs.workspace,
        name: newWs.name || newWs.workspace,
        project_keys: newWs.project_keys
      }).unwrap()
      setNewWs({ workspace: '', name: '', project_keys: '' })
      setShowAddWs(false)
    } catch (error) {
      console.error('Failed to add workspace:', error)
    }
  }

  const handleUpdateWorkspace = async (id: number) => {
    try {
      await updateWorkspace({
        id,
        name: editWs.name,
        project_keys: editWs.project_keys
      }).unwrap()
      setEditingWsId(null)
    } catch (error) {
      console.error('Failed to update workspace:', error)
    }
  }

  if (isLoading) {
    return <p className="text-pdt-neutral/60">Loading...</p>
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="Settings" />

      {/* Profile */}
      <DataCard title="Profile">
        <p className="text-pdt-neutral/60">{profile?.email}</p>
      </DataCard>

      <form onSubmit={handleSubmit} className="space-y-4 md:space-y-6">
        {/* GitHub */}
        <DataCard title="GitHub">
          <div className="space-y-2">
            <Input
              type="password"
              placeholder="ghp_xxxxxxxxxxxx"
              value={formData.github_token}
              onChange={(e) =>
                setFormData({ ...formData, github_token: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <div className="flex items-center gap-1 text-xs">
              {profile?.has_github_token ? (
                <StatusBadge variant="success">Configured</StatusBadge>
              ) : (
                <StatusBadge variant="danger">Not configured</StatusBadge>
              )}
            </div>
          </div>
        </DataCard>

        {/* GitLab */}
        <DataCard title="GitLab">
          <div className="space-y-2">
            <Input
              type="password"
              placeholder="Personal Access Token"
              value={formData.gitlab_token}
              onChange={(e) =>
                setFormData({ ...formData, gitlab_token: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <Input
              type="url"
              placeholder="https://gitlab.com"
              value={formData.gitlab_url}
              onChange={(e) =>
                setFormData({ ...formData, gitlab_url: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <div className="flex items-center gap-1 text-xs">
              {profile?.has_gitlab_token ? (
                <StatusBadge variant="success">Configured</StatusBadge>
              ) : (
                <StatusBadge variant="danger">Not configured</StatusBadge>
              )}
            </div>
          </div>
        </DataCard>

        {/* Jira Credentials */}
        <DataCard title="Jira Credentials">
          <div className="space-y-2">
            <Input
              type="email"
              placeholder="Email"
              value={formData.jira_email}
              onChange={(e) =>
                setFormData({ ...formData, jira_email: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <Input
              type="password"
              placeholder="API Token"
              value={formData.jira_token}
              onChange={(e) =>
                setFormData({ ...formData, jira_token: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <Input
              type="text"
              placeholder="Username (optional)"
              value={formData.jira_username}
              onChange={(e) =>
                setFormData({ ...formData, jira_username: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <p className="text-xs text-pdt-neutral/40">
              These credentials are shared across all Jira workspaces below.
            </p>
            <div className="flex items-center gap-1 text-xs">
              {profile?.has_jira_token ? (
                <StatusBadge variant="success">Configured</StatusBadge>
              ) : (
                <StatusBadge variant="danger">Not configured</StatusBadge>
              )}
            </div>
          </div>
        </DataCard>

        {/* Actions */}
        <div className="flex flex-wrap items-center gap-2">
          <Button type="submit" disabled={isUpdating} variant="pdt">
            <Save className="mr-2 size-4" />
            {isUpdating ? 'Saving...' : 'Save Changes'}
          </Button>
          <Button
            type="button"
            onClick={handleValidate}
            disabled={isValidating}
            variant="pdtOutline"
            size="sm"
          >
            {isValidating ? 'Testing...' : 'Test Connection'}
          </Button>
          {saveMessage && (
            <p
              className={cn(
                'text-sm',
                saveMessage.type === 'success'
                  ? 'text-green-400'
                  : 'text-red-400'
              )}
            >
              {saveMessage.text}
            </p>
          )}
        </div>
      </form>

      {/* Composio */}
      <ComposioSettings />

      {/* Jira Workspaces */}
      <DataCard
        title="Jira Workspaces"
      >
        <div className="space-y-3">
          {workspaces.length === 0 ? (
            <p className="text-sm text-pdt-neutral/40">
              No workspaces configured. Add one to start syncing Jira data.
            </p>
          ) : (
            workspaces.map((ws) => (
              <div
                key={ws.id}
                className="flex items-center justify-between border-b border-pdt-neutral/10 py-2 last:border-0"
              >
                {editingWsId === ws.id ? (
                  <div className="flex flex-1 items-center gap-2">
                    <Input
                      type="text"
                      value={editWs.name}
                      onChange={(e) =>
                        setEditWs({ ...editWs, name: e.target.value })
                      }
                      placeholder="Display name"
                      className="h-8 w-32 border-pdt-accent/20 bg-pdt-primary-light text-sm text-pdt-neutral"
                    />
                    <Input
                      type="text"
                      value={editWs.project_keys}
                      onChange={(e) =>
                        setEditWs({ ...editWs, project_keys: e.target.value })
                      }
                      placeholder="Project keys (e.g., PDT,CORE)"
                      className="h-8 flex-1 border-pdt-accent/20 bg-pdt-primary-light text-sm text-pdt-neutral"
                    />
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => handleUpdateWorkspace(ws.id)}
                    >
                      <Save className="size-3" />
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => setEditingWsId(null)}
                    >
                      Cancel
                    </Button>
                  </div>
                ) : (
                  <>
                    <div>
                      <p className="text-sm font-medium text-pdt-neutral">
                        {ws.name}
                      </p>
                      <p className="text-xs text-pdt-neutral/50">
                        {ws.workspace}
                        {ws.project_keys && ` — keys: ${ws.project_keys}`}
                      </p>
                    </div>
                    <div className="flex items-center gap-1">
                      <StatusBadge
                        variant={ws.is_active ? 'success' : 'warning'}
                      >
                        {ws.is_active ? 'Active' : 'Inactive'}
                      </StatusBadge>
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        disabled={isSyncingJira && syncingWsId === ws.id}
                        onClick={async () => {
                          setSyncingWsId(ws.id)
                          try {
                            await syncJira(ws.id).unwrap()
                          } finally {
                            setSyncingWsId(null)
                          }
                        }}
                      >
                        <RefreshCw
                          className={cn(
                            'size-3',
                            isSyncingJira && syncingWsId === ws.id && 'animate-spin'
                          )}
                        />
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                          setEditingWsId(ws.id)
                          setEditWs({
                            name: ws.name,
                            project_keys: ws.project_keys
                          })
                        }}
                      >
                        <Pencil className="size-3" />
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                          if (confirm(`Delete workspace "${ws.name}"?`)) {
                            deleteWorkspace(ws.id)
                          }
                        }}
                      >
                        <Trash2 className="size-3 text-red-400" />
                      </Button>
                    </div>
                  </>
                )}
              </div>
            ))
          )}

          {showAddWs ? (
            <div className="space-y-2 rounded-lg border border-pdt-accent/20 p-3">
              <Input
                type="text"
                placeholder="Workspace URL (e.g., myteam.atlassian.net)"
                value={newWs.workspace}
                onChange={(e) =>
                  setNewWs({ ...newWs, workspace: e.target.value })
                }
                className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
              />
              <Input
                type="text"
                placeholder="Display name (optional)"
                value={newWs.name}
                onChange={(e) =>
                  setNewWs({ ...newWs, name: e.target.value })
                }
                className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
              />
              <Input
                type="text"
                placeholder="Project keys (e.g., PDT,CORE) — optional"
                value={newWs.project_keys}
                onChange={(e) =>
                  setNewWs({ ...newWs, project_keys: e.target.value })
                }
                className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
              />
              <div className="flex gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="pdt"
                  onClick={handleAddWorkspace}
                >
                  Add
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  onClick={() => setShowAddWs(false)}
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <Button
              type="button"
              size="sm"
              variant="pdtOutline"
              onClick={() => setShowAddWs(true)}
            >
              <Plus className="mr-1 size-3" /> Add Workspace
            </Button>
          )}
        </div>
      </DataCard>

      {/* WhatsApp */}
      <NumberManager />
    </div>
  )
}

export default SettingsPage
