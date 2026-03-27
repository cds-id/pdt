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
  message_count?: number
  created_at: string
  updated_at: string
}

export interface IWaMessage {
  id: number
  wa_listener_id: number
  message_id: string
  sender_jid: string
  sender_name: string
  content: string
  message_type: 'text' | 'image' | 'document' | 'audio' | 'video'
  has_media: boolean
  timestamp: string
  created_at: string
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
