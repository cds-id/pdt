import { useMemo, useState } from 'react';
import {
  useListExecutiveReportsQuery,
  useGetExecutiveReportQuery,
  useDeleteExecutiveReportMutation,
} from '@/infrastructure/services/executiveReport.service';
import { useGenerateExecutiveReport } from '@/presentation/hooks/useGenerateExecutiveReport';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { ExecutiveReportView } from './ExecutiveReportView';

function isoFromInput(v: string): string {
  // v is a "YYYY-MM-DD" from <input type="date">
  // Return an ISO string for the start of that day in UTC.
  return new Date(v + 'T00:00:00Z').toISOString();
}

function dateInputValue(d: Date): string {
  const yyyy = d.getUTCFullYear();
  const mm = String(d.getUTCMonth() + 1).padStart(2, '0');
  const dd = String(d.getUTCDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

export function ExecutiveReportTab() {
  const now = new Date();
  const fourteenDaysAgo = new Date(now.getTime() - 14 * 24 * 60 * 60 * 1000);

  const [startInput, setStartInput] = useState(dateInputValue(fourteenDaysAgo));
  const [endInput, setEndInput] = useState(dateInputValue(now));
  const [staleDays, setStaleDays] = useState(7);
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const { data: list } = useListExecutiveReportsQuery();
  const { data: selected } = useGetExecutiveReportQuery(selectedId ?? 0, {
    skip: selectedId === null,
  });
  const [deleteReport] = useDeleteExecutiveReportMutation();

  const stream = useGenerateExecutiveReport();

  const activeView = useMemo(() => {
    if (selectedId && selected) {
      return {
        dataset: selected.dataset,
        narrative: selected.narrative,
        suggestions: selected.suggestions,
        phase: selected.status === 'completed' ? 'done' : selected.status,
        error:
          selected.status === 'failed'
            ? selected.error_message ?? 'failed'
            : null,
      };
    }
    return {
      dataset: stream.dataset,
      narrative: stream.narrative,
      suggestions: stream.suggestions,
      phase: stream.phase,
      error: stream.error,
    };
  }, [selectedId, selected, stream]);

  function handleGenerate() {
    setSelectedId(null);
    stream.start({
      rangeStart: isoFromInput(startInput),
      rangeEnd: isoFromInput(endInput),
      staleThresholdDays: staleDays,
    });
  }

  const busy =
    stream.phase !== 'idle' &&
    stream.phase !== 'done' &&
    stream.phase !== 'error';

  return (
    <div className="grid grid-cols-1 md:grid-cols-[240px_1fr] gap-4">
      <aside className="space-y-2">
        <h3 className="text-sm font-semibold text-muted-foreground">History</h3>
        {(list ?? []).length === 0 && (
          <div className="text-xs text-muted-foreground">No reports yet.</div>
        )}
        {(list ?? []).map((item) => (
          <Card
            key={item.id}
            role="button"
            onClick={() => setSelectedId(item.id)}
            className={`p-2 cursor-pointer ${selectedId === item.id ? 'border-primary' : ''}`}
          >
            <div className="text-xs font-mono">#{item.id}</div>
            <div className="text-xs">
              {item.range_start.slice(0, 10)} → {item.range_end.slice(0, 10)}
            </div>
            <div className="text-xs text-muted-foreground">{item.status}</div>
            <button
              className="text-xs text-destructive mt-1"
              onClick={(e) => {
                e.stopPropagation();
                deleteReport(item.id);
                if (selectedId === item.id) setSelectedId(null);
              }}
            >
              delete
            </button>
          </Card>
        ))}
      </aside>

      <div className="space-y-4">
        <Card className="p-4 flex flex-wrap items-end gap-4">
          <label className="text-sm flex flex-col">
            Start
            <Input
              type="date"
              value={startInput}
              onChange={(e) => setStartInput(e.target.value)}
              className="w-40"
            />
          </label>
          <label className="text-sm flex flex-col">
            End
            <Input
              type="date"
              value={endInput}
              onChange={(e) => setEndInput(e.target.value)}
              className="w-40"
            />
          </label>
          <label className="text-sm flex flex-col">
            Stale threshold (days)
            <Input
              type="number"
              min={1}
              value={staleDays}
              onChange={(e) => setStaleDays(parseInt(e.target.value, 10) || 7)}
              className="w-24"
            />
          </label>
          <Button onClick={handleGenerate} disabled={busy}>
            {busy ? 'Generating…' : 'Generate'}
          </Button>
        </Card>

        <ExecutiveReportView {...activeView} />
      </div>
    </div>
  );
}
