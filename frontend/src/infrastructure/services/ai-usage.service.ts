import { api } from './api'
import { API_CONSTANTS } from '../constants/api.constants'

export interface ProviderSummary {
  provider: string
  model: string
  feature: string
  calls: number
  prompt_tokens: number
  completion_tokens: number
}

export interface DailyUsage {
  date: string
  provider: string
  calls: number
  prompt_tokens: number
  completion_tokens: number
}

export interface AIUsageSummary {
  today: ProviderSummary[]
  month: ProviderSummary[]
  daily: DailyUsage[]
  totals: {
    calls: number
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
  rate_limits: {
    vision: {
      used: number
      limit: number
    }
  }
}

export const aiUsageApi = api.injectEndpoints({
  endpoints: (builder) => ({
    getAIUsage: builder.query<AIUsageSummary, void>({
      query: () => API_CONSTANTS.AI.USAGE,
      providesTags: ['AIUsage']
    })
  })
})

export const { useGetAIUsageQuery } = aiUsageApi
