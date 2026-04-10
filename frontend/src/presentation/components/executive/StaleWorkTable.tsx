import { Topic, Suggestion } from '@/infrastructure/services/executiveReport.service';
import { Card } from '@/components/ui/card';

interface Props {
  topics: Topic[];
  suggestions: Suggestion[];
}

export function StaleWorkTable({ topics, suggestions }: Props) {
  const stale = topics.filter((t) => t.stale);
  if (stale.length === 0) {
    return <Card className="p-4 text-sm text-muted-foreground">No stale work in this range.</Card>;
  }
  const actionFor = (key: string) =>
    suggestions.find((s) => s.kind === 'stale' && s.refs.includes(`jira:${key}`))?.detail ?? '';

  return (
    <Card className="p-0 overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-muted">
          <tr>
            <th className="text-left p-2">Card</th>
            <th className="text-left p-2">Title</th>
            <th className="text-right p-2">Days idle</th>
            <th className="text-left p-2">Suggested action</th>
          </tr>
        </thead>
        <tbody>
          {stale.map((t) => (
            <tr key={t.anchor.card_key} className="border-t">
              <td className="p-2 font-mono">{t.anchor.card_key}</td>
              <td className="p-2">{t.anchor.title}</td>
              <td className="p-2 text-right">{t.days_idle}</td>
              <td className="p-2 text-muted-foreground">{actionFor(t.anchor.card_key) || '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  );
}
