/**
 * User interface definitions
 */

export interface IUser {
  id: number
  email: string
  has_github_token: boolean
  has_gitlab_token: boolean
  gitlab_url: string
  jira_email: string
  has_jira_token: boolean
  jira_workspace: string
  jira_username: string
  jira_project_keys: string
}
