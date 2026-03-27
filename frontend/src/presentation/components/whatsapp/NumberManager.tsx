import { useState } from 'react'
import { useSelector } from 'react-redux'
import { Plus, Trash2, ChevronDown, ChevronRight } from 'lucide-react'
import { RootState } from '@/application/store'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataCard, StatusBadge, EmptyState } from '@/presentation/components/common'
import {
  useListNumbersQuery,
  useAddNumberMutation,
  useDeleteNumberMutation
} from '@/infrastructure/services/whatsapp.service'
import { QrPairingModal } from './QrPairingModal'
import { ListenerManager } from './ListenerManager'

export function NumberManager() {
  const token = useSelector((state: RootState) => state.auth.token) ?? ''

  const { data: numbers = [], isLoading } = useListNumbersQuery()
  const [addNumber, { isLoading: isAdding }] = useAddNumberMutation()
  const [deleteNumber] = useDeleteNumberMutation()

  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({ phone_number: '', display_name: '' })
  const [pairingNumberId, setPairingNumberId] = useState<number | null>(null)
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const statusVariant = (status: string) => {
    if (status === 'connected') return 'success'
    if (status === 'pairing') return 'warning'
    return 'danger'
  }

  const handleAdd = async () => {
    if (!form.phone_number.trim() || !form.display_name.trim()) return
    try {
      const result = await addNumber(form).unwrap()
      setForm({ phone_number: '', display_name: '' })
      setShowForm(false)
      setPairingNumberId(result.id)
    } catch (error) {
      console.error('Failed to add number:', error)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Remove this WhatsApp number and all its data?')) return
    try {
      await deleteNumber(id).unwrap()
    } catch (error) {
      console.error('Failed to delete number:', error)
    }
  }

  return (
    <DataCard
      title="WhatsApp Numbers"
      action={
        !showForm ? (
          <Button variant="pdt" size="sm" onClick={() => setShowForm(true)}>
            <Plus className="mr-2 size-4" />
            Add Number
          </Button>
        ) : undefined
      }
    >
      {/* Add number form */}
      {showForm && (
        <div className="mb-4 space-y-2 rounded-lg border border-pdt-accent/20 bg-pdt-primary p-4">
          <Input
            type="tel"
            placeholder="Phone number (e.g. +1234567890)"
            value={form.phone_number}
            onChange={(e) => setForm({ ...form, phone_number: e.target.value })}
            className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
          />
          <Input
            type="text"
            placeholder="Display name"
            value={form.display_name}
            onChange={(e) => setForm({ ...form, display_name: e.target.value })}
            className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
          />
          <div className="flex gap-2">
            <Button variant="pdt" size="sm" onClick={handleAdd} disabled={isAdding}>
              {isAdding ? 'Adding...' : 'Add & Pair'}
            </Button>
            <Button
              variant="pdtOutline"
              size="sm"
              onClick={() => {
                setShowForm(false)
                setForm({ phone_number: '', display_name: '' })
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      )}

      {/* Numbers list */}
      {isLoading ? (
        <p className="text-sm text-pdt-neutral/60">Loading...</p>
      ) : numbers.length === 0 && !showForm ? (
        <EmptyState
          title="No WhatsApp numbers added."
          description="Add a number to start collecting messages."
        />
      ) : (
        <div className="space-y-2">
          {numbers.map((number) => (
            <div key={number.id} className="rounded-lg border border-pdt-neutral/10 bg-pdt-primary">
              {/* Number row */}
              <div className="flex items-center justify-between p-3">
                <button
                  className="flex flex-1 items-center gap-2 text-left"
                  onClick={() =>
                    setExpandedId(expandedId === number.id ? null : number.id)
                  }
                >
                  {expandedId === number.id ? (
                    <ChevronDown className="size-4 shrink-0 text-pdt-neutral/40" />
                  ) : (
                    <ChevronRight className="size-4 shrink-0 text-pdt-neutral/40" />
                  )}
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="truncate text-sm font-medium text-pdt-neutral">
                        {number.display_name}
                      </p>
                      <StatusBadge variant={statusVariant(number.status)}>
                        {number.status}
                      </StatusBadge>
                    </div>
                    <p className="mt-0.5 text-xs text-pdt-neutral/40">{number.phone_number}</p>
                  </div>
                </button>
                <div className="ml-3 flex items-center gap-2">
                  {number.status === 'disconnected' && (
                    <Button
                      variant="pdtOutline"
                      size="sm"
                      onClick={() => setPairingNumberId(number.id)}
                    >
                      Re-pair
                    </Button>
                  )}
                  <button
                    onClick={() => handleDelete(number.id)}
                    className="text-pdt-neutral/60 transition-colors hover:text-red-400"
                  >
                    <Trash2 className="size-4" />
                  </button>
                </div>
              </div>

              {/* Listeners section */}
              {expandedId === number.id && (
                <div className="border-t border-pdt-neutral/10 px-3 pb-3">
                  <ListenerManager numberId={number.id} />
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* QR Pairing Modal */}
      {pairingNumberId !== null && (
        <QrPairingModal
          numberId={pairingNumberId}
          token={token}
          onClose={() => setPairingNumberId(null)}
          onSuccess={() => setPairingNumberId(null)}
        />
      )}
    </DataCard>
  )
}

export default NumberManager
