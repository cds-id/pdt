import { useEffect, useRef } from 'react'
import { useAppDispatch } from '@/application/hooks/useAppDispatch'
import { useAppSelector } from '@/application/hooks/useAppSelector'
import { API_CONSTANTS } from '@/infrastructure/constants/api.constants'
import { whatsappApi } from '@/infrastructure/services/whatsapp.service'
import {
  IWaMessage,
  IWaListener,
  IWaChatMessagePage
} from '@/domain/whatsapp/interfaces/whatsapp.interface'

interface WsEvent {
  type: 'message' | 'receipt' | 'chat_update'
  payload: unknown
}

/**
 * useWhatsAppSocket opens the inbox realtime WebSocket for the authenticated
 * user and patches RTK Query caches in place as events arrive:
 *  - message      → append to the conversation's message list + refresh chat list
 *  - receipt      → update an outgoing message's delivery status
 *  - chat_update  → upsert/reorder a conversation in the chat list
 *
 * Reconnects automatically with capped backoff.
 */
export function useWhatsAppSocket(numberId: number | null) {
  const dispatch = useAppDispatch()
  const token = useAppSelector((s) => s.auth.token)
  const wsRef = useRef<WebSocket | null>(null)
  const retryRef = useRef(0)
  const closedRef = useRef(false)

  useEffect(() => {
    if (!token || numberId == null) return
    closedRef.current = false

    const connect = () => {
      const wsBase = API_CONSTANTS.BASE_URL.replace(/^http/, 'ws')
      const url = `${wsBase}${API_CONSTANTS.API_PREFIX}${API_CONSTANTS.WA.WS}?token=${encodeURIComponent(token)}`
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        retryRef.current = 0
      }

      ws.onmessage = (e) => {
        let evt: WsEvent
        try {
          evt = JSON.parse(e.data)
        } catch {
          return
        }
        handleEvent(evt)
      }

      ws.onclose = () => {
        if (closedRef.current) return
        const delay = Math.min(1000 * 2 ** retryRef.current, 30000)
        retryRef.current += 1
        setTimeout(connect, delay)
      }

      ws.onerror = () => ws.close()
    }

    const handleEvent = (evt: WsEvent) => {
      if (evt.type === 'message') {
        const msg = evt.payload as IWaMessage
        // Append to the open conversation's cached messages (any page).
        dispatch(
          whatsappApi.util.updateQueryData(
            'getChatMessages',
            { listenerId: msg.wa_listener_id, page: 1, pageSize: 50 },
            (draft: IWaChatMessagePage) => {
              if (!draft?.data) return
              const exists = draft.data.some(
                (m) => m.id === msg.id || (m.message_id && m.message_id === msg.message_id)
              )
              if (!exists) {
                draft.data.push(msg)
                draft.total += 1
              } else {
                // media-arrival update: replace existing row
                const idx = draft.data.findIndex(
                  (m) => m.id === msg.id || m.message_id === msg.message_id
                )
                if (idx >= 0) draft.data[idx] = msg
              }
            }
          )
        )
        // Refresh chat list ordering/preview.
        dispatch(
          whatsappApi.util.invalidateTags([
            { type: 'WhatsApp' as const, id: `CHATS_${numberId}` }
          ])
        )
      } else if (evt.type === 'receipt') {
        // Receipts are infrequent; invalidate cached messages so delivery
        // ticks re-fetch with the latest status.
        dispatch(whatsappApi.util.invalidateTags(['WhatsApp']))
      } else if (evt.type === 'chat_update') {
        const _listener = evt.payload as IWaListener
        void _listener
        dispatch(
          whatsappApi.util.invalidateTags([
            { type: 'WhatsApp' as const, id: `CHATS_${numberId}` }
          ])
        )
      }
    }

    connect()

    return () => {
      closedRef.current = true
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [token, numberId, dispatch])
}
