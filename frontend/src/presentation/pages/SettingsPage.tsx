import { useState } from 'react'
import { Save } from 'lucide-react'

import {
  useGetProfileQuery,
  useUpdateProfileMutation,
  useValidateIntegrationsMutation
} from '@/infrastructure/services/user.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  PageHeader,
  DataCard,
  StatusBadge
} from '@/presentation/components/common'
import { cn } from '@/lib/utils'

export function SettingsPage() {
  const { data: profile, isLoading } = useGetProfileQuery()
  const [updateProfile, { isLoading: isUpdating }] = useUpdateProfileMutation()
  const [validate, { isLoading: isValidating }] =
    useValidateIntegrationsMutation()

  const [formData, setFormData] = useState({
    github_token: '',
    gitlab_token: '',
    gitlab_url: 'https://gitlab.com',
    jira_email: '',
    jira_token: '',
    jira_workspace: '',
    jira_username: '',
    jira_project_keys: ''
  })

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

        {/* Jira */}
        <DataCard title="Jira">
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
              placeholder="Workspace (e.g., myteam.atlassian.net)"
              value={formData.jira_workspace}
              onChange={(e) =>
                setFormData({ ...formData, jira_workspace: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <Input
              type="text"
              placeholder="Username"
              value={formData.jira_username}
              onChange={(e) =>
                setFormData({ ...formData, jira_username: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <Input
              type="text"
              placeholder="Project keys (e.g., PDT, CORE)"
              value={formData.jira_project_keys}
              onChange={(e) =>
                setFormData({ ...formData, jira_project_keys: e.target.value })
              }
              className="mb-2 border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
            />
            <p className="mb-2 text-xs text-pdt-neutral/40">
              Comma-separated project key prefixes. Leave empty to show all.
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
    </div>
  )
}

export default SettingsPage
