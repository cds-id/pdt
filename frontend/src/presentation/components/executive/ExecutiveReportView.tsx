import { CorrelatedDataset, Suggestion, Metrics } from '@/infrastructure/services/executiveReport.service';
import { Card } from '@/components/ui/card';
import { ExecutiveActivityChart } from './ExecutiveActivityChart';
import { StaleWorkTable } from './StaleWorkTable';
import { SuggestionList } from './SuggestionList';
import { LinkageGaugeChart } from '@/presentation/components/charts/LinkageGaugeChart';

interface Props {
  dataset: CorrelatedDataset | null;
  narrative: string;
  suggestions: Suggestion[];
  phase: string;
  error: string | null;
}

export function ExecutiveReportView({ dataset, narrative, suggestions, phase, error }: Props) {
  if (error) {
    return (
      <Card className="p-4 border-destructive text-destructive">
        Error: {error}
      </Card>
    );
  }
  if (!dataset) {
    return <Card className="p-4 text-muted-foreground">{phaseLabel(phase)}</Card>;
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-4">
      <div className="space-y-4">
        <MetricsRow metrics={dataset.metrics} />
        <Card className="p-4">
          <h3 className="font-semibold mb-2">Activity Timeline</h3>
          <ExecutiveActivityChart buckets={dataset.daily_buckets} />
        </Card>
        <StaleWorkTable topics={dataset.topics} suggestions={suggestions} />
        {narrative && (
          <Card className="p-4 prose prose-sm max-w-none dark:prose-invert">
            {/* react-markdown is not installed; using pre as a readable fallback */}
            <pre className="whitespace-pre-wrap font-sans">{narrative}</pre>
          </Card>
        )}
        {!narrative && phase !== 'done' && (
          <Card className="p-4 text-muted-foreground text-sm">{phaseLabel(phase)}</Card>
        )}
      </div>
      <aside className="space-y-4">
        <SuggestionList suggestions={suggestions} />
      </aside>
    </div>
  );
}

function MetricsRow({ metrics }: { metrics: Metrics }) {
  const waTotal = metrics.wa_topics_ticketed + metrics.wa_topics_orphan;
  return (
    <div className="grid grid-cols-3 gap-4">
      <Card className="p-3">
        <div className="text-xs text-muted-foreground mb-1">Commit linkage</div>
        <LinkageGaugeChart linked={metrics.commits_linked} total={metrics.commits_total} />
      </Card>
      <Card className="p-3">
        <div className="text-xs text-muted-foreground mb-1">Card linkage</div>
        <LinkageGaugeChart linked={metrics.cards_with_commits} total={metrics.cards_active} />
      </Card>
      <Card className="p-3">
        <div className="text-xs text-muted-foreground mb-1">WA topics ticketed</div>
        <LinkageGaugeChart linked={metrics.wa_topics_ticketed} total={waTotal} />
      </Card>
    </div>
  );
}

function phaseLabel(phase: string): string {
  switch (phase) {
    case 'correlating':
      return 'Correlating data from Jira, commits, and WhatsApp…';
    case 'thinking':
      return 'Generating narrative…';
    case 'streaming':
      return 'Writing report…';
    case 'persisting':
      return 'Saving…';
    case 'done':
      return 'Ready.';
    default:
      return 'Idle.';
  }
}
