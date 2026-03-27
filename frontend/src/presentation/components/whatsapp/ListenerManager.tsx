import { useState } from 'react'
import { Plus, Trash2, Pause, Play } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { StatusBadge, EmptyState } from '@/presentation/components/common'
import {
  useListListenersQuery,
  useAddListenerMutation,
  useUpdateListenerMutation,
  useDeleteListenerMutation
} from '@/infrastructure/services/whatsapp.service'

interface ListenerManagerProps {
  numberId: number
}

export function ListenerManager({ numberId }: ListenerManagerProps) {
  const { data: listeners = [], isLoading } = useListListenersQuery(numberId)
  const [addListener, { isLoading: isAdding }] = useAddListenerMutation()
  const [updateListener] = useUpdateListenerMutation()
  const [deleteListener] = useDeleteListenerMutation()

  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    jid: '',
    name: '',
    type: 'group' as 'group' | 'personal'
  })

  const handleAdd = async () => {
    if (!form.jid.trim() || !form.name.trim()) return
    try {
      await addListener({ numberId, ...form }).unwrap()
      setForm({ jid: '', name: '', type: 'group' })
      setShowForm(false)
    } catch (error) {
      console.error('Failed to add listener:', error)
    }
  }

  const handleToggle = async (id: number, isActive: boolean) => {
    try {
      await updateListener({ id, numberId, is_active: !isActive }).unwrap()
    } catch (error) {
      console.error('Failed to toggle listener:', error)
    }
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Remove this listener?')) return
    try {
      await deleteListener({ id, numberId }).unwrap()
    } catch (error) {
      console.error('Failed to delete listener:', error)
    }
  }

  if (isLoading) {
    return <p className="text-sm text-pdt-neutral/60">Loading listeners...</p>
  }

  return (
    <div className="mt-3 space-y-3">
      {listeners.length === 0 && !showForm ? (
        <EmptyState
          title="No listeners configured."
          description="Add a group or personal chat to start collecting messages."
        />
      ) : (
        <div className="space-y-2">
          {listeners.map((listener) => (
            <div
              key={listener.id}
              className="flex items-center justify-between rounded-lg border border-pdt-neutral/10 bg-pdt-primary p-3"
            >
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="truncate text-sm font-medium text-pdt-neutral">{listener.name}</p>
                  <StatusBadge variant={listener.type === 'group' ? 'info' : 'neutral'}>
                    {listener.type}
                  </StatusBadge>
                  <StatusBadge variant={listener.is_active ? 'success' : 'neutral'}>
                    {listener.is_active ? 'active' : 'paused'}
                  </StatusBadge>
                </div>
                <p className="mt-0.5 truncate text-xs text-pdt-neutral/40">{listener.jid}</p>
                {listener.message_count !== undefined && (
                  <p className="mt-0.5 text-xs text-pdt-neutral/40">
                    {listener.message_count} messages
                  </p>
                )}
              </div>
              <div className="ml-3 flex items-center gap-2">
                <button
                  onClick={() => handleToggle(listener.id, listener.is_active)}
                  title={listener.is_active ? 'Pause' : 'Resume'}
                  className="text-pdt-neutral/60 transition-colors hover:text-pdt-accent"
                >
                  {listener.is_active ? (
                    <Pause className="size-4" />
                  ) : (
                    <Play className="size-4" />
                  )}
                </button>
                <button
                  onClick={() => handleDelete(listener.id)}
                  className="text-pdt-neutral/60 transition-colors hover:text-red-400"
                >
                  <Trash2 className="size-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showForm ? (
        <div className="space-y-2 rounded-lg border border-pdt-accent/20 bg-pdt-primary p-3">
          <Input
            placeholder="JID (e.g. 123456789@g.us)"
            value={form.jid}
            onChange={(e) => setForm({ ...form, jid: e.target.value })}
            className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
          />
          <Input
            placeholder="Display name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
          />
          <select
            value={form.type}
            onChange={(e) => setForm({ ...form, type: e.target.value as 'group' | 'personal' })}
            className="w-full rounded-lg border border-pdt-accent/20 bg-pdt-primary-light px-3 py-2 text-sm text-pdt-neutral"
          >
            <option value="group">Group</option>
            <option value="personal">Personal</option>
          </select>
          <div className="flex gap-2">
            <Button variant="pdt" size="sm" onClick={handleAdd} disabled={isAdding}>
              {isAdding ? 'Adding...' : 'Add'}
            </Button>
            <Button
              variant="pdtOutline"
              size="sm"
              onClick={() => {
                setShowForm(false)
                setForm({ jid: '', name: '', type: 'group' })
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      ) : (
        <Button
          variant="pdtOutline"
          size="sm"
          onClick={() => setShowForm(true)}
          className="w-full"
        >
          <Plus className="mr-2 size-4" />
          Add Listener
        </Button>
      )}
    </div>
  )
}

export default ListenerManager
