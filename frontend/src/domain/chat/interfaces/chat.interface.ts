export interface IConversation {
  id: string
  user_id: number
  title: string
  created_at: string
  updated_at: string
  messages?: IChatMessage[]
}

export interface IChatMessage {
  id: string
  conversation_id: string
  role: 'user' | 'assistant' | 'tool'
  content: string
  tool_calls?: string
  tool_name?: string
  created_at: string
}

export interface IWSMessage {
  type: 'message'
  content: string
  conversation_id?: string
}

export interface IWSResponse {
  type: 'stream' | 'tool_status' | 'done' | 'error'
  content?: string
  conversation_id?: string
  tool?: string
  status?: 'executing' | 'completed'
}
