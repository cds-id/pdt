import { useCallback, useRef, useState } from 'react'
import { useAppDispatch } from '@/application/hooks/useAppDispatch'
import { useAppSelector } from '@/application/hooks/useAppSelector'
import {
  executiveReportApi,
  CorrelatedDataset,
  Suggestion,
} from '@/infrastructure/services/executiveReport.service'
import { API_CONSTANTS } from '@/infrastructure/constants/api.constants'

export type Phase =
  | 'idle'
  | 'correlating'
  | 'thinking'
  | 'streaming'
  | 'persisting'
  | 'done'
  | 'error'

export interface GenerateArgs {
  rangeStart: string // ISO
  rangeEnd: string
  staleThresholdDays?: number
  workspaceId?: number
}

const GENERATE_URL =
  API_CONSTANTS.BASE_URL +
  API_CONSTANTS.API_PREFIX +
  '/protected/reports/executive/generate'

export function useGenerateExecutiveReport() {
  const dispatch = useAppDispatch()
  const token = useAppSelector((state) => state.auth.token)

  const [phase, setPhase] = useState<Phase>('idle')
  const [dataset, setDataset] = useState<CorrelatedDataset | null>(null)
  const [narrative, setNarrative] = useState('')
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [error, setError] = useState<string | null>(null)
  const [reportId, setReportId] = useState<number | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  const reset = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    setPhase('idle')
    setDataset(null)
    setNarrative('')
    setSuggestions([])
    setError(null)
    setReportId(null)
  }, [])

  const start = useCallback(
    (args: GenerateArgs) => {
      reset()
      setPhase('correlating')

      const ctrl = new AbortController()
      abortRef.current = ctrl

      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      }
      if (token) headers['authorization'] = `Bearer ${token}`

      fetch(GENERATE_URL, {
        method: 'POST',
        headers,
        credentials: 'include',
        signal: ctrl.signal,
        body: JSON.stringify({
          range_start: args.rangeStart,
          range_end: args.rangeEnd,
          stale_threshold_days: args.staleThresholdDays,
          workspace_id: args.workspaceId,
        }),
      })
        .then(async (res) => {
          if (!res.ok || !res.body) {
            const text = await res.text().catch(() => '')
            throw new Error(text || `HTTP ${res.status}`)
          }
          const reader = res.body.getReader()
          const decoder = new TextDecoder()
          let buffer = ''
          for (;;) {
            const { value, done } = await reader.read()
            if (done) break
            buffer += decoder.decode(value, { stream: true })
            let idx: number
            while ((idx = buffer.indexOf('\n\n')) !== -1) {
              const raw = buffer.slice(0, idx)
              buffer = buffer.slice(idx + 2)
              handleFrame(raw)
            }
          }
        })
        .catch((e) => {
          if ((e as Error).name === 'AbortError') return
          setPhase('error')
          setError(e instanceof Error ? e.message : String(e))
        })

      function handleFrame(raw: string) {
        const lines = raw.split('\n')
        let event = 'message'
        let data = ''
        for (const line of lines) {
          if (line.startsWith('event: ')) event = line.slice(7)
          else if (line.startsWith('data: ')) data += line.slice(6)
        }
        if (!data) return
        let parsed: unknown
        try {
          parsed = JSON.parse(data)
        } catch {
          return
        }
        const p = parsed as Record<string, unknown>
        switch (event) {
          case 'status':
            if (p.phase === 'correlating') setPhase('correlating')
            else if (p.phase === 'thinking') setPhase('thinking')
            else if (p.phase === 'persisting') setPhase('persisting')
            break
          case 'dataset':
            setDataset(parsed as CorrelatedDataset)
            setPhase('streaming')
            break
          case 'delta':
            setNarrative((n) => n + ((p.text as string) ?? ''))
            break
          case 'suggestion':
            setSuggestions((s) => [...s, parsed as Suggestion])
            break
          case 'done':
            setPhase('done')
            setReportId((p.id as number) ?? null)
            dispatch(executiveReportApi.util.invalidateTags(['ExecutiveReport']))
            break
          case 'error':
            setPhase('error')
            setError((p.message as string) ?? 'unknown error')
            break
        }
      }
    },
    [dispatch, reset, token],
  )

  return { phase, dataset, narrative, suggestions, error, reportId, start, reset }
}
