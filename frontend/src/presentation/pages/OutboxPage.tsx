import { useState } from 'react'
import { Send, Edit, Trash2, XCircle, Check, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  PageHeader,
  DataCard,
  StatusBadge,
  EmptyState
} from '@/presentation/components/common'
import {
  useListOutboxQuery,
  useUpdateOutboxMutation,
  useDeleteOutboxMutation
} from '@/infrastructure/services/whatsapp.service'
import { IWaOutbox } from '@/domain/whatsapp/interfaces/whatsapp.interface'

type StatusFilter = 'all' | 'pending' | 'sent' | 'rejected' | 'approved'

const statusVariant = (status: string) => {
  if (status === 'sent') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'approved') return 'info'
  if (status === 'rejected') return 'danger'
  return 'neutral'
}

export function OutboxPage() {
  const [filter, setFilter] = useState<StatusFilter>('all')
  const [editingId, setEditingId] = useState<number | null>(null)
  const [editContent, setEditContent] = useState('')

  const { data: items = [], isLoading } = useListOutboxQuery(
    filter === 'all' ? undefined : { status: filter }
  )
  const [updateOutbox, { isLoading: isUpdating }] = useUpdateOutboxMutation()
  const [deleteOutbox] = useDeleteOutboxMutation()

  const filterButtons: { label: string; value: StatusFilter }[] = [
    { label: 'All', value: 'all' },
    { label: 'Pending', value: 'pending' },
    { label: 'Approved', value: 'approved' },
    { label: 'Sent', value: 'sent' },
    { label: 'Rejected', value: 'rejected' }
  ]

  const handleApprove = async (item: IWaOutbox) => {
    try {
      await updateOutbox({ id: item.id, status: 'approved' }).unwrap()
    } catch (error) {
      console.error('Failed to approve:', error)
    }
  }

  const handleSend = async (item: IWaOutbox) => {
    try {
      await updateOutbox({ id: item.id, status: 'sent' }).unwrap()
    } catch (error) {
      console.error('Failed to send:', error)
    }
  }

  const handleReject = async (item: IWaOutbox) => {
    try {
      await updateOutbox({ id: item.id, status: 'rejected' }).unwrap()
    } catch (error) {
      console.error('Failed to reject:', error)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Delete this outbox item?')) return
    try {
      await deleteOutbox(id).unwrap()
    } catch (error) {
      console.error('Failed to delete:', error)
    }
  }

  const startEdit = (item: IWaOutbox) => {
    setEditingId(item.id)
    setEditContent(item.content)
  }

  const saveEdit = async () => {
    if (editingId === null) return
    try {
      await updateOutbox({ id: editingId, content: editContent }).unwrap()
      setEditingId(null)
      setEditContent('')
    } catch (error) {
      console.error('Failed to update content:', error)
    }
  }

  const cancelEdit = () => {
    setEditingId(null)
    setEditContent('')
  }

  return (
    <div className="min-w-0 space-y-4 md:space-y-6">
      <PageHeader title="Outbox" />

      {/* Filter tabs */}
      <div className="flex flex-wrap gap-2">
        {filterButtons.map(({ label, value }) => (
          <Button
            key={value}
            variant={filter === value ? 'pdt' : 'pdtOutline'}
            size="sm"
            onClick={() => setFilter(value)}
          >
            {label}
          </Button>
        ))}
      </div>

      <DataCard title="Outbox Messages">
        {isLoading ? (
          <p className="text-sm text-pdt-neutral/60">Loading...</p>
        ) : items.length === 0 ? (
          <EmptyState
            title="No outbox messages."
            description="Agent-generated messages will appear here for review."
          />
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <div
                key={item.id}
                className="rounded-lg border border-pdt-neutral/10 bg-pdt-primary-light p-4"
              >
                {/* Header row */}
                <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="text-sm font-medium text-pdt-neutral">{item.target_name}</p>
                    <span className="text-xs text-pdt-neutral/40">{item.target_jid}</span>
                    <StatusBadge variant={statusVariant(item.status)}>{item.status}</StatusBadge>
                    <StatusBadge variant="neutral">by {item.requested_by}</StatusBadge>
                  </div>
                  <p className="text-xs text-pdt-neutral/40">
                    {new Date(item.created_at).toLocaleString()}
                  </p>
                </div>

                {/* Context / reason */}
                {item.context && (
                  <p className="mb-2 rounded-md bg-pdt-primary/60 px-3 py-2 text-xs text-pdt-neutral/60">
                    <span className="font-medium text-pdt-accent">Context: </span>
                    {item.context}
                  </p>
                )}

                {/* Message content */}
                {editingId === item.id ? (
                  <div className="space-y-2">
                    <textarea
                      value={editContent}
                      onChange={(e) => setEditContent(e.target.value)}
                      className="min-h-[100px] w-full rounded-lg border border-pdt-accent/20 bg-pdt-primary p-3 text-sm text-pdt-neutral placeholder:text-pdt-neutral/40 focus:outline-none focus:ring-2 focus:ring-pdt-accent/40"
                    />
                    <div className="flex gap-2">
                      <Button
                        variant="pdt"
                        size="sm"
                        onClick={saveEdit}
                        disabled={isUpdating}
                      >
                        <Check className="mr-1 size-3" />
                        Save
                      </Button>
                      <Button variant="pdtOutline" size="sm" onClick={cancelEdit}>
                        <X className="mr-1 size-3" />
                        Cancel
                      </Button>
                    </div>
                  </div>
                ) : (
                  <p className="rounded-md bg-pdt-primary/40 px-3 py-2 text-sm text-pdt-neutral">
                    {item.content}
                  </p>
                )}

                {/* Timestamps */}
                <div className="mt-2 flex flex-wrap gap-4 text-xs text-pdt-neutral/40">
                  {item.approved_at && (
                    <span>Approved: {new Date(item.approved_at).toLocaleString()}</span>
                  )}
                  {item.sent_at && (
                    <span>Sent: {new Date(item.sent_at).toLocaleString()}</span>
                  )}
                </div>

                {/* Actions */}
                {item.status === 'pending' && editingId !== item.id && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    <Button
                      variant="pdt"
                      size="sm"
                      onClick={() => handleSend(item)}
                      disabled={isUpdating}
                    >
                      <Send className="mr-1 size-3" />
                      Send
                    </Button>
                    <Button
                      variant="pdtOutline"
                      size="sm"
                      onClick={() => handleApprove(item)}
                      disabled={isUpdating}
                    >
                      <Check className="mr-1 size-3" />
                      Approve
                    </Button>
                    <Button
                      variant="pdtOutline"
                      size="sm"
                      onClick={() => startEdit(item)}
                    >
                      <Edit className="mr-1 size-3" />
                      Edit
                    </Button>
                    <Button
                      variant="pdtOutline"
                      size="sm"
                      onClick={() => handleReject(item)}
                      disabled={isUpdating}
                    >
                      <XCircle className="mr-1 size-3" />
                      Reject
                    </Button>
                    <button
                      onClick={() => handleDelete(item.id)}
                      className="ml-auto text-pdt-neutral/60 transition-colors hover:text-red-400"
                    >
                      <Trash2 className="size-4" />
                    </button>
                  </div>
                )}

                {(item.status === 'sent' || item.status === 'rejected') && (
                  <div className="mt-3 flex justify-end">
                    <button
                      onClick={() => handleDelete(item.id)}
                      className="text-pdt-neutral/60 transition-colors hover:text-red-400"
                    >
                      <Trash2 className="size-4" />
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </DataCard>
    </div>
  )
}

export default OutboxPage
