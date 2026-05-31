import { Check, CheckCheck, FileText, Download } from 'lucide-react'
import { IWaMessage } from '@/domain/whatsapp/interfaces/whatsapp.interface'
import { cn } from '@/lib/utils'

function formatTime(iso: string) {
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function StatusTicks({ status }: { status?: string }) {
  if (status === 'read') return <CheckCheck className="h-3.5 w-3.5 text-sky-400" />
  if (status === 'delivered') return <CheckCheck className="h-3.5 w-3.5 text-muted-foreground" />
  return <Check className="h-3.5 w-3.5 text-muted-foreground" />
}

function MediaBlock({ msg }: { msg: IWaMessage }) {
  const media = msg.media?.[0]
  const url = media?.file_url
  const mime = media?.mime_type || ''
  const type = msg.message_type

  if (!url) {
    // Media still downloading on the server.
    if (msg.has_media) {
      return (
        <div className="mb-1 rounded-md bg-black/5 px-3 py-6 text-center text-xs text-muted-foreground">
          Loading {type}…
        </div>
      )
    }
    return null
  }

  if (type === 'image' || type === 'sticker' || mime.startsWith('image/')) {
    return (
      <a href={url} target="_blank" rel="noreferrer">
        <img
          src={url}
          alt={media?.file_name || 'image'}
          className="mb-1 max-h-80 rounded-md object-cover"
        />
      </a>
    )
  }
  if (type === 'video' || mime.startsWith('video/')) {
    return <video src={url} controls className="mb-1 max-h-80 rounded-md" />
  }
  if (type === 'audio' || mime.startsWith('audio/')) {
    return <audio src={url} controls className="mb-1 w-64 max-w-full" />
  }
  // document / other
  return (
    <a
      href={url}
      target="_blank"
      rel="noreferrer"
      className="mb-1 flex items-center gap-2 rounded-md bg-black/5 px-3 py-2 text-sm hover:bg-black/10"
    >
      <FileText className="h-5 w-5 shrink-0" />
      <span className="min-w-0 flex-1 truncate">{media?.file_name || 'Document'}</span>
      <Download className="h-4 w-4 shrink-0" />
    </a>
  )
}

export function MessageBubble({ msg }: { msg: IWaMessage }) {
  const me = msg.from_me
  // Hide auto media-label content when an actual attachment is shown.
  const isLabel = /^\[(image|video|audio|document|sticker)\]$/.test(msg.content)
  const showText = msg.content && !(msg.has_media && isLabel)

  return (
    <div className={cn('flex px-4 py-0.5', me ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[75%] rounded-lg px-3 py-2 shadow-sm',
          me ? 'bg-emerald-100 text-emerald-950' : 'bg-card text-foreground border border-border'
        )}
      >
        {!me && msg.sender_name && (
          <div className="mb-0.5 text-xs font-semibold text-emerald-700">{msg.sender_name}</div>
        )}
        <MediaBlock msg={msg} />
        {showText && <p className="whitespace-pre-wrap break-words text-sm">{msg.content}</p>}
        <div className="mt-0.5 flex items-center justify-end gap-1">
          <span className="text-[10px] text-muted-foreground">{formatTime(msg.timestamp)}</span>
          {me && <StatusTicks status={msg.status} />}
        </div>
      </div>
    </div>
  )
}
