export interface IWaNumber {
  id: number
  user_id: number
  phone_number: string
  display_name: string
  status: 'pairing' | 'connected' | 'disconnected'
  paired_at?: string
  created_at: string
  updated_at: string
}

export interface IWaListener {
  id: number
  wa_number_id: number
  jid: string
  name: string
  type: 'group' | 'personal'
  is_active: boolean
  source?: 'monitor' | 'inbox'
  last_message_at?: string
  last_message_preview?: string
  unread_count?: number
  message_count?: number
  created_at: string
  updated_at: string
}

// IWaChat is an alias for a conversation row (a listener acting as a chat).
export type IWaChat = IWaListener

export interface IWaMedia {
  id: number
  wa_message_id: number
  file_name: string
  mime_type: string
  file_size: number
  r2_key: string
  file_url: string
  created_at: string
}

export type WaMediaType = 'text' | 'image' | 'document' | 'audio' | 'video' | 'sticker'

export interface IWaMessage {
  id: number
  wa_listener_id: number
  message_id: string
  sender_jid: string
  sender_name: string
  content: string
  message_type: WaMediaType
  has_media: boolean
  from_me: boolean
  chat_jid: string
  status?: 'sent' | 'delivered' | 'read'
  media?: IWaMedia[]
  timestamp: string
  created_at: string
}

export interface IWaChatMessagePage {
  data: IWaMessage[]
  total: number
  page: number
  page_size: number
}

export interface IWaUploadResult {
  url: string
  mime: string
  media_type: WaMediaType
  file_name: string
}

export interface IWaWsEvent {
  type: 'message' | 'receipt' | 'chat_update'
  payload: IWaMessage | IWaListener | { message_id: string; status: string }
}

export interface IWaOutbox {
  id: number
  wa_number_id: number
  target_jid: string
  target_name: string
  content: string
  status: 'pending' | 'approved' | 'sent' | 'rejected'
  requested_by: 'agent' | 'user'
  context: string
  approved_at?: string
  sent_at?: string
  created_at: string
}

export interface IWaMessagePage {
  messages: IWaMessage[]
  total: number
  page: number
  limit: number
}
