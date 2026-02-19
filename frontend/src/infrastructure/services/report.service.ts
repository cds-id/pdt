import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface Report {
  id: number
  user_id: number
  template_id?: number
  date: string
  title: string
  content: string
  file_url: string
  created_at: string
}

export interface ReportTemplate {
  id: number
  name: string
  content: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export const reportApi = api.injectEndpoints({
  endpoints: (builder) => ({
    generateReport: builder.mutation<Report, string>({
      query: (date) => ({
        url: API_CONSTANTS.REPORTS.GENERATE,
        method: 'POST',
        body: { date }
      }),
      invalidatesTags: [{ type: 'Report' as const, id: 'LIST' }]
    }),
    listReports: builder.query<Report[], { from?: string; to?: string } | void>(
      {
        query: (filters) => {
          const params = new URLSearchParams()
          if (filters?.from) params.append('from', filters.from)
          if (filters?.to) params.append('to', filters.to)
          const query = params.toString()
          return `${API_CONSTANTS.REPORTS.LIST}${query ? `?${query}` : ''}`
        },
        providesTags: (result) =>
          result
            ? [
                ...result.map(({ id }) => ({ type: 'Report' as const, id })),
                { type: 'Report', id: 'LIST' }
              ]
            : [{ type: 'Report', id: 'LIST' }]
      }
    ),
    getReport: builder.query<Report, string>({
      query: (id) => API_CONSTANTS.REPORTS.GET(id),
      providesTags: (_, __, id) => [{ type: 'Report' as const, id }]
    }),
    deleteReport: builder.mutation<void, string>({
      query: (id) => ({
        url: API_CONSTANTS.REPORTS.DELETE(id),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'Report', id: 'LIST' }]
    }),
    // Templates
    createTemplate: builder.mutation<
      ReportTemplate,
      { name: string; content: string }
    >({
      query: (data) => ({
        url: API_CONSTANTS.REPORTS.TEMPLATES_CREATE,
        method: 'POST',
        body: data
      }),
      invalidatesTags: [{ type: 'ReportTemplate', id: 'LIST' }]
    }),
    listTemplates: builder.query<ReportTemplate[], void>({
      query: () => API_CONSTANTS.REPORTS.TEMPLATES_LIST,
      providesTags: [{ type: 'ReportTemplate', id: 'LIST' }]
    }),
    updateTemplate: builder.mutation<
      ReportTemplate,
      { id: string; name: string; content: string }
    >({
      query: ({ id, name, content }) => ({
        url: API_CONSTANTS.REPORTS.TEMPLATES_UPDATE(id),
        method: 'PUT',
        body: { name, content }
      }),
      invalidatesTags: [{ type: 'ReportTemplate', id: 'LIST' }]
    }),
    deleteTemplate: builder.mutation<void, string>({
      query: (id) => ({
        url: API_CONSTANTS.REPORTS.TEMPLATES_DELETE(id),
        method: 'DELETE'
      }),
      invalidatesTags: [{ type: 'ReportTemplate', id: 'LIST' }]
    }),
    previewTemplate: builder.query<
      { preview: string },
      { content: string; date: string }
    >({
      query: ({ content, date }) => ({
        url: API_CONSTANTS.REPORTS.TEMPLATES_PREVIEW,
        method: 'POST',
        body: { content, date }
      })
    })
  })
})

export const {
  useGenerateReportMutation,
  useListReportsQuery,
  useGetReportQuery,
  useDeleteReportMutation,
  useCreateTemplateMutation,
  useListTemplatesQuery,
  useUpdateTemplateMutation,
  useDeleteTemplateMutation,
  usePreviewTemplateQuery
} = reportApi
