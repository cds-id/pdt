import { Suggestion } from '@/infrastructure/services/executiveReport.service';
import { Card } from '@/components/ui/card';

interface Props {
  suggestions: Suggestion[];
}

const GROUP_LABEL: Record<Suggestion['kind'], string> = {
  gap: 'GAPS',
  stale: 'STALE WORK',
  next_step: 'NEXT STEPS',
};

export function SuggestionList({ suggestions }: Props) {
  const grouped: Record<Suggestion['kind'], Suggestion[]> = {
    gap: [],
    stale: [],
    next_step: [],
  };
  for (const s of suggestions) grouped[s.kind].push(s);

  return (
    <div className="space-y-4">
      {(['gap', 'stale', 'next_step'] as const).map(
        (k) =>
          grouped[k].length > 0 && (
            <section key={k}>
              <h3 className="text-sm font-semibold text-muted-foreground uppercase mb-2">
                {GROUP_LABEL[k]}
              </h3>
              <div className="space-y-2">
                {grouped[k].map((s, i) => (
                  <Card key={i} className="p-3">
                    <div className="font-medium">{s.title}</div>
                    <div className="text-sm text-muted-foreground">{s.detail}</div>
                    {s.refs.length > 0 && (
                      <div className="mt-1 text-xs text-muted-foreground/80">
                        {s.refs.join(' · ')}
                      </div>
                    )}
                  </Card>
                ))}
              </div>
            </section>
          ),
      )}
    </div>
  );
}
