import { useState } from 'react'
import { Save, CheckCircle, XCircle } from 'lucide-react'

import { useGetProfileQuery, useUpdateProfileMutation, useValidateIntegrationsMutation } from '@/infrastructure/services/user.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

export function SettingsPage() {
  const { data: profile, isLoading } = useGetProfileQuery()
  const [updateProfile, { isLoading: isUpdating }] = useUpdateProfileMutation()
  const [validate, { isLoading: isValidating }] = useValidateIntegrationsMutation()

  const [formData, setFormData] = useState({
    github_token: '',
    gitlab_token: '',
    gitlab_url: 'https://gitlab.com',
    jira_email: '',
    jira_token: '',
    jira_workspace: '',
    jira_username: ''
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const data = Object.fromEntries(
      Object.entries(formData).filter(([_, v]) => v.trim() !== '')
    )
    try {
      await updateProfile(data).unwrap()
      alert('Settings saved successfully!')
    } catch (error) {
      console.error('Failed to save settings:', error)
      alert('Failed to save settings')
    }
  }

  const handleValidate = async () => {
    try {
      await validate().unwrap()
      alert('Validation complete!')
    } catch (error) {
      console.error('Validation failed:', error)
    }
  }

  if (isLoading) {
    return <p className="text-[#FBFFFE]/60">Loading...</p>
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <h1 className="text-2xl font-bold text-[#FBFFFE] md:text-3xl">Settings</h1>

      {/* Profile */}
      <div className="rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-4">
        <h2 className="mb-4 text-lg font-semibold text-[#FBFFFE]">Profile</h2>
        <p className="text-[#FBFFFE]/60">{profile?.email}</p>
      </div>

      {/* Integrations */}
      <form onSubmit={handleSubmit} className="space-y-4 rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-4">
        <h2 className="text-lg font-semibold text-[#FBFFFE]">Integrations</h2>

        {/* GitHub */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-[#FBFFFE]">GitHub</label>
          <Input
            type="password"
            placeholder="ghp_xxxxxxxxxxxx"
            value={formData.github_token}
            onChange={(e) => setFormData({ ...formData, github_token: e.target.value })}
            className="mb-2 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <p className="flex items-center gap-1 text-xs text-[#FBFFFE]/40">
            {(profile as any)?.hasGithubToken ? (
              <>
                <CheckCircle className="h-3 w-3 text-green-400" />
                <span>Configured</span>
              </>
            ) : (
              <>
                <XCircle className="h-3 w-3 text-red-400" />
                <span>Not configured</span>
              </>
            )}
          </p>
        </div>

        {/* GitLab */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-[#FBFFFE]">GitLab</label>
          <Input
            type="password"
            placeholder="Personal Access Token"
            value={formData.gitlab_token}
            onChange={(e) => setFormData({ ...formData, gitlab_token: e.target.value })}
            className="mb-2 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <Input
            type="url"
            placeholder="https://gitlab.com"
            value={formData.gitlab_url}
            onChange={(e) => setFormData({ ...formData, gitlab_url: e.target.value })}
            className="mb-2 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <p className="flex items-center gap-1 text-xs text-[#FBFFFE]/40">
            {(profile as any)?.hasGitlabToken ? (
              <>
                <CheckCircle className="h-3 w-3 text-green-400" />
                <span>Configured</span>
              </>
            ) : (
              <>
                <XCircle className="h-3 w-3 text-red-400" />
                <span>Not configured</span>
              </>
            )}
          </p>
        </div>

        {/* Jira */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-[#FBFFFE]">Jira</label>
          <Input
            type="email"
            placeholder="Email"
            value={formData.jira_email}
            onChange={(e) => setFormData({ ...formData, jira_email: e.target.value })}
            className="mb-2 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <Input
            type="password"
            placeholder="API Token"
            value={formData.jira_token}
            onChange={(e) => setFormData({ ...formData, jira_token: e.target.value })}
            className="mb-2 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <Input
            type="text"
            placeholder="Workspace (e.g., myteam.atlassian.net)"
            value={formData.jira_workspace}
            onChange={(e) => setFormData({ ...formData, jira_workspace: e.target.value })}
            className="mb-2 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <Input
            type="text"
            placeholder="Username"
            value={formData.jira_username}
            onChange={(e) => setFormData({ ...formData, jira_username: e.target.value })}
            className="mb-2 bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE] placeholder:text-[#FBFFFE]/40"
          />
          <p className="flex items-center gap-1 text-xs text-[#FBFFFE]/40">
            {(profile as any)?.hasJiraToken ? (
              <>
                <CheckCircle className="h-3 w-3 text-green-400" />
                <span>Configured</span>
              </>
            ) : (
              <>
                <XCircle className="h-3 w-3 text-red-400" />
                <span>Not configured</span>
              </>
            )}
          </p>
        </div>

        {/* Actions */}
        <div className="flex gap-2 pt-4">
          <Button
            type="submit"
            disabled={isUpdating}
            className="bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90"
          >
            <Save className="mr-2 h-4 w-4" />
            {isUpdating ? 'Saving...' : 'Save Changes'}
          </Button>
          <Button
            type="button"
            onClick={handleValidate}
            disabled={isValidating}
            variant="outline"
            className="border-[#F8C630] text-[#F8C630] hover:bg-[#F8C630] hover:text-[#1B1B1E]"
          >
            {isValidating ? 'Validating...' : 'Validate'}
          </Button>
        </div>
      </form>
    </div>
  )
}

export default SettingsPage
