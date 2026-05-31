import { useRef, useState } from 'react'
import { Paperclip, Send, X, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  useSendChatMessageMutation,
  useUploadWaMediaMutation
} from '@/infrastructure/services/whatsapp.service'

interface ComposerProps {
  numberId: number
  chatJid: string
}

export function Composer({ numberId, chatJid }: ComposerProps) {
  const [text, setText] = useState('')
  const [pending, setPending] = useState<{
    url: string
    media_type: string
    file_name: string
    mime: string
  } | null>(null)
  const fileRef = useRef<HTMLInputElement | null>(null)

  const [sendMessage, { isLoading: sending }] = useSendChatMessageMutation()
  const [uploadMedia, { isLoading: uploading }] = useUploadWaMediaMutation()

  const onPickFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const fd = new FormData()
    fd.append('file', file)
    try {
      const res = await uploadMedia(fd).unwrap()
      setPending(res)
    } catch {
      // swallow; UI stays unchanged
    } finally {
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const canSend = (text.trim().length > 0 || pending) && !sending

  const onSend = async () => {
    if (!canSend) return
    try {
      await sendMessage({
        numberId,
        chat_jid: chatJid,
        text: text.trim(),
        media_url: pending?.url,
        media_type: pending?.media_type
      }).unwrap()
      setText('')
      setPending(null)
    } catch {
      // keep input so user can retry
    }
  }

  return (
    <div className="border-t border-border bg-card p-3">
      {pending && (
        <div className="mb-2 flex items-center gap-2 rounded-md bg-accent px-3 py-2 text-sm">
          <Paperclip className="h-4 w-4 shrink-0" />
          <span className="min-w-0 flex-1 truncate">{pending.file_name}</span>
          <button onClick={() => setPending(null)} className="shrink-0 text-muted-foreground">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}
      <div className="flex items-center gap-2">
        <input
          ref={fileRef}
          type="file"
          className="hidden"
          accept="image/*,video/*,audio/*,application/*,.pdf,.doc,.docx,.xls,.xlsx,.zip"
          onChange={onPickFile}
        />
        <Button
          variant="ghost"
          size="icon"
          onClick={() => fileRef.current?.click()}
          disabled={uploading}
          title="Attach"
        >
          {uploading ? <Loader2 className="h-5 w-5 animate-spin" /> : <Paperclip className="h-5 w-5" />}
        </Button>
        <Input
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              onSend()
            }
          }}
          placeholder="Type a message"
          className="flex-1"
        />
        <Button onClick={onSend} disabled={!canSend} size="icon" title="Send">
          {sending ? <Loader2 className="h-5 w-5 animate-spin" /> : <Send className="h-5 w-5" />}
        </Button>
      </div>
    </div>
  )
}
