import { Loader2, CheckCircle } from 'lucide-react'

interface ToolStatusProps {
  toolName: string
  status: 'executing' | 'completed'
}

const toolLabels: Record<string, string> = {
  search_commits: 'Searching commits',
  get_commit_detail: 'Getting commit details',
  list_repos: 'Listing repositories',
  get_repo_stats: 'Getting repo statistics',
  get_sprints: 'Fetching sprints',
  get_cards: 'Fetching Jira cards',
  get_card_detail: 'Getting card details',
  search_cards: 'Searching cards',
  link_commit_to_card: 'Linking commit to card',
  generate_daily_report: 'Generating daily report',
  generate_monthly_report: 'Generating monthly report',
  list_reports: 'Listing reports',
  get_report: 'Getting report',
  preview_template: 'Previewing template',
}

export function ToolStatus({ toolName, status }: ToolStatusProps) {
  const label = toolLabels[toolName] || toolName

  return (
    <div className="flex items-center gap-2 text-xs text-pdt-neutral-400 py-1 px-3">
      {status === 'executing' ? (
        <Loader2 className="w-3 h-3 animate-spin" />
      ) : (
        <CheckCircle className="w-3 h-3 text-green-500" />
      )}
      <span>{label}...</span>
    </div>
  )
}
