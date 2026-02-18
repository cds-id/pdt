import { useState } from 'react'
import { FilePlus, Trash2, Download } from 'lucide-react'

import { useListReportsQuery, useGenerateReportMutation, useDeleteReportMutation } from '@/infrastructure/services/report.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

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
      <h1 className="text-2xl font-bold text-[#FBFFFE] md:text-3xl">Reports</h1>

      {/* Generate Report */}
      <div className="rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-4">
        <h2 className="mb-4 text-lg font-semibold text-[#FBFFFE]">Generate Report</h2>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
          <div>
            <label className="mb-1 block text-sm text-[#FBFFFE]/60">Date</label>
            <Input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="bg-[#1B1B1E] border-[#F8C630]/20 text-[#FBFFFE]"
            />
          </div>
          <Button
            onClick={handleGenerate}
            disabled={isGenerating}
            className="bg-[#F8C630] text-[#1B1B1E] hover:bg-[#F8C630]/90"
          >
            <FilePlus className="mr-2 h-4 w-4" />
            {isGenerating ? 'Generating...' : 'Generate'}
          </Button>
        </div>
      </div>

      {/* Reports List */}
      <div>
        <h2 className="mb-4 text-lg font-semibold text-[#FBFFFE]">Past Reports</h2>
        {isLoading ? (
          <p className="text-[#FBFFFE]/60">Loading...</p>
        ) : reports.length === 0 ? (
          <div className="rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-8 text-center">
            <p className="text-[#FBFFFE]/60">No reports generated yet.</p>
            <p className="mt-2 text-sm text-[#FBFFFE]/40">
              Generate a report above to get started.
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {reports.map((report) => (
              <div
                key={report.id}
                className="flex items-center justify-between rounded-lg border border-[#F8C630]/20 bg-[#1B1B1E] p-4"
              >
                <div>
                  <p className="font-medium text-[#FBFFFE]">
                    {(report as any).title || `Report - ${report.date}`}
                  </p>
                  <p className="text-sm text-[#FBFFFE]/60">
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
                      className="text-[#F8C630] transition-colors hover:text-[#F8C630]/80"
                    >
                      <Download className="h-5 w-5" />
                    </a>
                  )}
                  <button
                    onClick={() => handleDelete(report.id)}
                    className="text-[#FBFFFE]/60 transition-colors hover:text-red-400"
                  >
                    <Trash2 className="h-5 w-5" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

export default ReportsPage
