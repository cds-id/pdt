import { useEffect, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { API_CONSTANTS } from '@/infrastructure/constants/api.constants'

type PairingStatus = 'connecting' | 'waiting' | 'success' | 'error'

interface QrPairingModalProps {
  numberId: number
  token: string
  onClose: () => void
  onSuccess: () => void
}

export function QrPairingModal({ numberId, token, onClose, onSuccess }: QrPairingModalProps) {
  const [status, setStatus] = useState<PairingStatus>('connecting')
  const [qrCode, setQrCode] = useState<string | null>(null)
  const [errorMsg, setErrorMsg] = useState<string>('')
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const wsBase = API_CONSTANTS.BASE_URL.replace(/^http/, 'ws')
    const wsUrl = `${wsBase}${API_CONSTANTS.API_PREFIX}/ws/wa/pair/${numberId}?token=${encodeURIComponent(token)}`

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setStatus('waiting')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'qr') {
          setQrCode(data.qr)
          setStatus('waiting')
        } else if (data.type === 'success') {
          setStatus('success')
          setTimeout(() => {
            onSuccess()
            onClose()
          }, 1500)
        } else if (data.type === 'error') {
          setErrorMsg(data.message || 'Pairing failed.')
          setStatus('error')
        }
      } catch {
        // Non-JSON messages treated as QR strings for backward compat
        setQrCode(event.data)
        setStatus('waiting')
      }
    }

    ws.onerror = () => {
      setErrorMsg('WebSocket connection failed.')
      setStatus('error')
    }

    ws.onclose = (e) => {
      if (status !== 'success') {
        if (e.code !== 1000) {
          setErrorMsg('Connection closed unexpectedly.')
          setStatus('error')
        }
      }
    }

    return () => {
      ws.close()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [numberId, token])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="relative w-full max-w-sm rounded-xl border border-pdt-accent/20 bg-pdt-primary-light p-6 shadow-2xl">
        {/* Close button */}
        <button
          onClick={onClose}
          className="absolute right-3 top-3 text-pdt-neutral/60 transition-colors hover:text-pdt-neutral"
        >
          <X className="size-5" />
        </button>

        <h2 className="mb-4 text-lg font-semibold text-pdt-neutral">Pair WhatsApp Number</h2>

        {status === 'connecting' && (
          <div className="flex flex-col items-center gap-3 py-8">
            <div className="size-8 animate-spin rounded-full border-2 border-pdt-accent border-t-transparent" />
            <p className="text-sm text-pdt-neutral/60">Connecting...</p>
          </div>
        )}

        {status === 'waiting' && !qrCode && (
          <div className="flex flex-col items-center gap-3 py-8">
            <div className="size-8 animate-spin rounded-full border-2 border-pdt-accent border-t-transparent" />
            <p className="text-sm text-pdt-neutral/60">Waiting for QR code...</p>
          </div>
        )}

        {status === 'waiting' && qrCode && (
          <div className="flex flex-col items-center gap-4">
            <p className="text-sm text-pdt-neutral/60">
              Scan this QR code with WhatsApp to pair your number.
            </p>
            <div className="rounded-lg border border-pdt-accent/20 bg-white p-3">
              <img
                src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(qrCode)}`}
                alt="WhatsApp QR Code"
                className="size-[200px]"
              />
            </div>
            <p className="text-xs text-pdt-neutral/40">QR code refreshes automatically</p>
          </div>
        )}

        {status === 'success' && (
          <div className="flex flex-col items-center gap-3 py-8">
            <div className="flex size-12 items-center justify-center rounded-full bg-green-500/20">
              <span className="text-2xl text-green-400">&#10003;</span>
            </div>
            <p className="font-medium text-green-400">Paired successfully!</p>
          </div>
        )}

        {status === 'error' && (
          <div className="flex flex-col items-center gap-4 py-6">
            <div className="flex size-12 items-center justify-center rounded-full bg-red-500/20">
              <span className="text-2xl text-red-400">&#10007;</span>
            </div>
            <p className="text-sm text-red-400">{errorMsg || 'An error occurred.'}</p>
            <button
              onClick={onClose}
              className="rounded-lg bg-pdt-accent px-4 py-2 text-sm font-medium text-pdt-primary transition-colors hover:bg-pdt-accent/80"
            >
              Close
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default QrPairingModal
