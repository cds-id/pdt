import { useState } from 'react'
import { FilePlus, Trash2, Download } from 'lucide-react'

import { useListReportsQuery, useGenerateReportMutation, useDeleteReportMutation } from '@/infrastructure/services/report.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PageHeader, DataCard, EmptyState } from '@/presentation/components/common'

export function ReportsPage() {
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const { data: reportsData, isLoading } = useListReportsQuery()
  const [generateReport, { isLoading: isGenerating }] = useGenerateReportMutation()
  const [deleteReport] = useDeleteReportMutation()

  const reports = reportsData?.reports || []

  const handleGenerate = async () => {
    try {
      await generateReport(date).unwrap()
    } catch (error) {
      console.error('Failed to generate report:', error)
    }
  }

  const handleDelete = async (id: string) => {
    if (confirm('Are you sure you want to delete this report?')) {
      try {
        await deleteReport(id).unwrap()
      } catch (error) {
        console.error('Failed to delete report:', error)
      }
    }
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="Reports" />

      {/* Generate Report */}
      <DataCard title="Generate Report">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
          <div>
            <label className="mb-1 block text-sm text-pdt-neutral/60">Date</label>
            <Input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="bg-pdt-primary-light border-pdt-background/20 text-pdt-neutral"
            />
          </div>
          <Button
            onClick={handleGenerate}
            disabled={isGenerating}
            variant="pdt"
          >
            <FilePlus className="mr-2 h-4 w-4" />
            {isGenerating ? 'Generating...' : 'Generate'}
          </Button>
        </div>
      </DataCard>

      {/* Reports List */}
      <DataCard title="Past Reports">
        {isLoading ? (
          <p className="text-pdt-neutral/60">Loading...</p>
        ) : reports.length === 0 ? (
          <EmptyState
            title="No reports generated yet."
            description="Generate a report above to get started."
          />
        ) : (
          <div className="space-y-2">
            {reports.map((report) => (
              <div
                key={report.id}
                className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-4"
              >
                <div>
                  <p className="font-medium text-pdt-neutral">
                    {(report as any).title || `Report - ${report.date}`}
                  </p>
                  <p className="text-sm text-pdt-neutral/60">
                    {(report as any).commitsCount || 0} commits &middot;{' '}
                    {(report as any).jiraCardsCount || 0} Jira cards
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {(report as any).fileUrl && (
                    <a
                      href={(report as any).fileUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-pdt-background hover:text-pdt-background/80"
                    >
                      <Download className="h-5 w-5" />
                    </a>
                  )}
                  <button
                    onClick={() => handleDelete(report.id)}
                    className="text-pdt-neutral/60 transition-colors hover:text-red-400"
                  >
                    <Trash2 className="h-5 w-5" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </DataCard>
    </div>
  )
}

export default ReportsPage
