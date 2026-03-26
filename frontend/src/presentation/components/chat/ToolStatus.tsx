import { Loader2, CheckCircle } from 'lucide-react'
import { ChatEvent, ChatEventAddon, ChatEventBody, ChatEventContent } from '@/components/chat/chat-event'

interface ToolStatusProps {
  toolName: string
  status: 'executing' | 'completed'
}

const toolLabels: Record<string, string> = {
  search_commits: 'Searching commits',
  get_commit_detail: 'Getting commit details',
  get_commit_changes: 'Fetching code changes',
  analyze_card_changes: 'Analyzing card changes',
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
  search_comments: 'Searching comments',
  get_card_comments: 'Fetching card comments',
  find_person_statements: 'Finding statements',
  get_comment_timeline: 'Building timeline',
  detect_quality_issues: 'Detecting quality issues',
  check_requirement_coverage: 'Checking requirement coverage',
}

export function ToolStatus({ toolName, status }: ToolStatusProps) {
  const label = toolLabels[toolName] || toolName

  return (
    <ChatEvent className="py-1">
      <ChatEventAddon className="w-10">
        <div className="size-5 flex items-center justify-center">
          {status === 'executing' ? (
            <Loader2 className="size-3.5 animate-spin text-pdt-accent" />
          ) : (
            <CheckCircle className="size-3.5 text-green-500" />
          )}
        </div>
      </ChatEventAddon>
      <ChatEventBody>
        <ChatEventContent className="text-xs text-muted-foreground">
          {label}...
        </ChatEventContent>
      </ChatEventBody>
    </ChatEvent>
  )
}
