import { useState } from 'react'
import { FilePlus, Trash2, Download, FileCode, Edit, Plus, Eye } from 'lucide-react'

import {
  useListReportsQuery, useGenerateReportMutation, useDeleteReportMutation,
  useListTemplatesQuery, useCreateTemplateMutation, useUpdateTemplateMutation, useDeleteTemplateMutation
} from '@/infrastructure/services/report.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { PageHeader, DataCard, EmptyState } from '@/presentation/components/common'

export function ReportsPage() {
  const [date, setDate] = useState(new Date().toISOString().split('T')[0])
  const { data: reportsData, isLoading } = useListReportsQuery()
  const [generateReport, { isLoading: isGenerating }] = useGenerateReportMutation()
  const [deleteReport] = useDeleteReportMutation()

  const reports = reportsData || []

  // Tab state
  const [activeTab, setActiveTab] = useState<'reports' | 'templates'>('reports')

  // Template state
  const { data: templates = [] } = useListTemplatesQuery()
  const [createTemplate] = useCreateTemplateMutation()
  const [updateTemplate] = useUpdateTemplateMutation()
  const [deleteTemplateApi] = useDeleteTemplateMutation()
  const [showTemplateForm, setShowTemplateForm] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [templateForm, setTemplateForm] = useState({ name: '', content: '' })

  // Report content viewer
  const [expandedReport, setExpandedReport] = useState<number | null>(null)

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

  const handleCreateTemplate = async () => {
    if (!templateForm.name.trim() || !templateForm.content.trim()) return
    try {
      await createTemplate({ name: templateForm.name, content: templateForm.content }).unwrap()
      setTemplateForm({ name: '', content: '' })
      setShowTemplateForm(false)
    } catch (error) {
      console.error('Failed to create template:', error)
    }
  }

  const handleUpdateTemplate = async () => {
    if (editingId === null) return
    if (!templateForm.name.trim() || !templateForm.content.trim()) return
    try {
      await updateTemplate({ id: String(editingId), name: templateForm.name, content: templateForm.content }).unwrap()
      setEditingId(null)
      setTemplateForm({ name: '', content: '' })
    } catch (error) {
      console.error('Failed to update template:', error)
    }
  }

  const handleDeleteTemplate = async (id: number) => {
    if (confirm('Are you sure you want to delete this template?')) {
      try {
        await deleteTemplateApi(String(id)).unwrap()
      } catch (error) {
        console.error('Failed to delete template:', error)
      }
    }
  }

  const startEditing = (template: { id: number; name: string; content: string }) => {
    setEditingId(template.id)
    setTemplateForm({ name: template.name, content: template.content })
    setShowTemplateForm(false)
  }

  const cancelEditing = () => {
    setEditingId(null)
    setTemplateForm({ name: '', content: '' })
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="Reports" />

      {/* Tab buttons */}
      <div className="flex gap-2">
        <Button
          variant={activeTab === 'reports' ? 'pdt' : 'pdtOutline'}
          size="sm"
          onClick={() => setActiveTab('reports')}
        >
          <FileCode className="mr-2 h-4 w-4" />
          Reports
        </Button>
        <Button
          variant={activeTab === 'templates' ? 'pdt' : 'pdtOutline'}
          size="sm"
          onClick={() => setActiveTab('templates')}
        >
          <FilePlus className="mr-2 h-4 w-4" />
          Templates
        </Button>
      </div>

      {activeTab === 'reports' ? (
        <>
          {/* Generate Report */}
          <DataCard title="Generate Report">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
              <div>
                <label className="mb-1 block text-sm text-pdt-neutral/60">Date</label>
                <Input
                  type="date"
                  value={date}
                  onChange={(e) => setDate(e.target.value)}
                  className="bg-pdt-primary-light border-pdt-accent/20 text-pdt-neutral"
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
                  <div key={report.id}>
                    <div
                      className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-4 cursor-pointer"
                      onClick={() => setExpandedReport(expandedReport === report.id ? null : report.id)}
                    >
                      <div className="flex items-center gap-3">
                        <Eye className="h-4 w-4 text-pdt-neutral/40" />
                        <div>
                          <p className="font-medium text-pdt-neutral">
                            {report.title || `Report - ${report.date}`}
                          </p>
                          <p className="text-sm text-pdt-neutral/60">
                            {report.date}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {report.file_url && (
                          <a
                            href={report.file_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-pdt-accent hover:text-pdt-accent/80"
                            onClick={(e) => e.stopPropagation()}
                          >
                            <Download className="h-5 w-5" />
                          </a>
                        )}
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDelete(String(report.id))
                          }}
                          className="text-pdt-neutral/60 transition-colors hover:text-red-400"
                        >
                          <Trash2 className="h-5 w-5" />
                        </button>
                      </div>
                    </div>
                    {expandedReport === report.id && report.content && (
                      <div className="mt-1 rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light/50 p-4">
                        <pre className="whitespace-pre-wrap text-sm text-pdt-neutral font-mono">
                          {report.content}
                        </pre>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </DataCard>
        </>
      ) : (
        <>
          {/* Templates Tab */}
          <DataCard title="Report Templates">
            <div className="space-y-4">
              {/* New Template button */}
              {!showTemplateForm && editingId === null && (
                <Button
                  variant="pdt"
                  size="sm"
                  onClick={() => {
                    setShowTemplateForm(true)
                    setTemplateForm({ name: '', content: '' })
                  }}
                >
                  <Plus className="mr-2 h-4 w-4" />
                  New Template
                </Button>
              )}

              {/* Inline create form */}
              {showTemplateForm && (
                <div className="space-y-3 rounded-lg border border-pdt-accent/20 bg-pdt-primary-light p-4">
                  <Input
                    placeholder="Template name"
                    value={templateForm.name}
                    onChange={(e) => setTemplateForm({ ...templateForm, name: e.target.value })}
                    className="bg-pdt-primary border-pdt-accent/20 text-pdt-neutral placeholder:text-pdt-neutral/40"
                  />
                  <textarea
                    placeholder="Template content..."
                    value={templateForm.content}
                    onChange={(e) => setTemplateForm({ ...templateForm, content: e.target.value })}
                    className="min-h-[200px] w-full rounded-lg border border-pdt-accent/20 bg-pdt-primary p-3 font-mono text-sm text-pdt-neutral placeholder:text-pdt-neutral/40 focus:outline-none focus:ring-2 focus:ring-pdt-accent/40"
                  />
                  <div className="flex gap-2">
                    <Button variant="pdt" size="sm" onClick={handleCreateTemplate}>
                      Create
                    </Button>
                    <Button
                      variant="pdtOutline"
                      size="sm"
                      onClick={() => {
                        setShowTemplateForm(false)
                        setTemplateForm({ name: '', content: '' })
                      }}
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              )}

              {/* Templates list */}
              {templates.length === 0 && !showTemplateForm ? (
                <EmptyState
                  title="No templates yet."
                  description="Create a template to use when generating reports."
                />
              ) : (
                <div className="space-y-2">
                  {templates.map((template) => (
                    <div
                      key={template.id}
                      className="rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-4"
                    >
                      {editingId === template.id ? (
                        <div className="space-y-3">
                          <Input
                            value={templateForm.name}
                            onChange={(e) => setTemplateForm({ ...templateForm, name: e.target.value })}
                            className="bg-pdt-primary border-pdt-accent/20 text-pdt-neutral"
                          />
                          <textarea
                            value={templateForm.content}
                            onChange={(e) => setTemplateForm({ ...templateForm, content: e.target.value })}
                            className="min-h-[200px] w-full rounded-lg border border-pdt-accent/20 bg-pdt-primary p-3 font-mono text-sm text-pdt-neutral focus:outline-none focus:ring-2 focus:ring-pdt-accent/40"
                          />
                          <div className="flex gap-2">
                            <Button variant="pdt" size="sm" onClick={handleUpdateTemplate}>
                              Save
                            </Button>
                            <Button variant="pdtOutline" size="sm" onClick={cancelEditing}>
                              Cancel
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <div className="flex items-start justify-between">
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <p className="font-medium text-pdt-neutral">{template.name}</p>
                              {template.is_default && (
                                <span className="rounded bg-pdt-accent/20 px-2 py-0.5 text-xs text-pdt-accent">
                                  Default
                                </span>
                              )}
                            </div>
                            <p className="mt-1 text-sm text-pdt-neutral/60 truncate">
                              {template.content.slice(0, 80)}{template.content.length > 80 ? '...' : ''}
                            </p>
                          </div>
                          <div className="flex items-center gap-2 ml-4">
                            <button
                              onClick={() => startEditing(template)}
                              className="text-pdt-neutral/60 transition-colors hover:text-pdt-accent"
                            >
                              <Edit className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => handleDeleteTemplate(template.id)}
                              className="text-pdt-neutral/60 transition-colors hover:text-red-400"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </DataCard>
        </>
      )}
    </div>
  )
}

export default ReportsPage
