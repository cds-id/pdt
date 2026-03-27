import { useState } from 'react'
import { Plus, Trash2, Pause, Play, Users, User, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { StatusBadge, EmptyState } from '@/presentation/components/common'
import {
  useListListenersQuery,
  useAddListenerMutation,
  useUpdateListenerMutation,
  useDeleteListenerMutation,
  useGetGroupsQuery,
  useGetContactsQuery
} from '@/infrastructure/services/whatsapp.service'

interface ListenerManagerProps {
  numberId: number
}

type PickerTab = 'groups' | 'contacts' | 'manual'

export function ListenerManager({ numberId }: ListenerManagerProps) {
  const { data: listeners = [], isLoading } = useListListenersQuery(numberId)
  const [addListener, { isLoading: isAdding }] = useAddListenerMutation()
  const [updateListener] = useUpdateListenerMutation()
  const [deleteListener] = useDeleteListenerMutation()

  const [showPicker, setShowPicker] = useState(false)
  const [pickerTab, setPickerTab] = useState<PickerTab>('groups')
  const [search, setSearch] = useState('')
  const [manualForm, setManualForm] = useState({ jid: '', name: '', type: 'group' as 'group' | 'personal' })

  const { data: groups = [], isLoading: loadingGroups } = useGetGroupsQuery(numberId, { skip: !showPicker || pickerTab !== 'groups' })
  const { data: contacts = [], isLoading: loadingContacts } = useGetContactsQuery(numberId, { skip: !showPicker || pickerTab !== 'contacts' })

  // Filter already added JIDs
  const existingJids = new Set(listeners.map((l) => l.jid))

  const filteredGroups = groups
    .filter((g) => !existingJids.has(g.jid))
    .filter((g) => !search || g.name.toLowerCase().includes(search.toLowerCase()))

  const filteredContacts = contacts
    .filter((c) => !existingJids.has(c.jid))
    .filter((c) => c.name && c.name.trim() !== '')
    .filter((c) => !search || c.name.toLowerCase().includes(search.toLowerCase()) || c.jid.includes(search))

  const handleAddFromPicker = async (jid: string, name: string, type: 'group' | 'personal') => {
    try {
      await addListener({ numberId, jid, name, type }).unwrap()
    } catch (error) {
      console.error('Failed to add listener:', error)
    }
  }

  const handleAddManual = async () => {
    if (!manualForm.jid.trim() || !manualForm.name.trim()) return
    try {
      await addListener({ numberId, ...manualForm }).unwrap()
      setManualForm({ jid: '', name: '', type: 'group' })
      setShowPicker(false)
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
      {/* Existing listeners */}
      {listeners.length === 0 && !showPicker ? (
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
                  {listener.is_active ? <Pause className="size-4" /> : <Play className="size-4" />}
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

      {/* Picker */}
      {showPicker ? (
        <div className="space-y-3 rounded-lg border border-pdt-accent/20 bg-pdt-primary p-3">
          {/* Tabs */}
          <div className="flex gap-1 rounded-lg bg-pdt-primary-light p-1">
            {(['groups', 'contacts', 'manual'] as PickerTab[]).map((tab) => (
              <button
                key={tab}
                onClick={() => { setPickerTab(tab); setSearch('') }}
                className={`flex-1 rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
                  pickerTab === tab
                    ? 'bg-pdt-accent text-pdt-primary'
                    : 'text-pdt-neutral/60 hover:text-pdt-neutral'
                }`}
              >
                {tab === 'groups' ? 'Groups' : tab === 'contacts' ? 'Contacts' : 'Manual'}
              </button>
            ))}
          </div>

          {/* Search (for groups/contacts tabs) */}
          {pickerTab !== 'manual' && (
            <div className="relative">
              <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-pdt-neutral/40" />
              <Input
                placeholder={`Search ${pickerTab}...`}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="border-pdt-accent/20 bg-pdt-primary-light pl-9 text-pdt-neutral placeholder:text-pdt-neutral/40"
              />
            </div>
          )}

          {/* Groups list */}
          {pickerTab === 'groups' && (
            <div className="max-h-60 space-y-1 overflow-y-auto">
              {loadingGroups ? (
                <p className="py-4 text-center text-xs text-pdt-neutral/40">Loading groups...</p>
              ) : filteredGroups.length === 0 ? (
                <p className="py-4 text-center text-xs text-pdt-neutral/40">
                  {search ? 'No matching groups' : 'No groups available'}
                </p>
              ) : (
                filteredGroups.map((g) => (
                  <button
                    key={g.jid}
                    onClick={() => handleAddFromPicker(g.jid, g.name, 'group')}
                    disabled={isAdding}
                    className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-pdt-primary-light"
                  >
                    <Users className="size-4 shrink-0 text-pdt-accent" />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-pdt-neutral">{g.name}</p>
                      <p className="truncate text-xs text-pdt-neutral/40">
                        {g.participant_count} members
                        {g.topic ? ` · ${g.topic}` : ''}
                      </p>
                    </div>
                    <Plus className="size-4 shrink-0 text-pdt-neutral/40" />
                  </button>
                ))
              )}
            </div>
          )}

          {/* Contacts list */}
          {pickerTab === 'contacts' && (
            <div className="max-h-60 space-y-1 overflow-y-auto">
              {loadingContacts ? (
                <p className="py-4 text-center text-xs text-pdt-neutral/40">Loading contacts...</p>
              ) : filteredContacts.length === 0 ? (
                <p className="py-4 text-center text-xs text-pdt-neutral/40">
                  {search ? 'No matching contacts' : 'No contacts available'}
                </p>
              ) : (
                filteredContacts.map((c) => (
                  <button
                    key={c.jid}
                    onClick={() => handleAddFromPicker(c.jid, c.name, 'personal')}
                    disabled={isAdding}
                    className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors hover:bg-pdt-primary-light"
                  >
                    <User className="size-4 shrink-0 text-pdt-accent" />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-pdt-neutral">{c.name}</p>
                      <p className="truncate text-xs text-pdt-neutral/40">{c.jid}</p>
                    </div>
                    <Plus className="size-4 shrink-0 text-pdt-neutral/40" />
                  </button>
                ))
              )}
            </div>
          )}

          {/* Manual form */}
          {pickerTab === 'manual' && (
            <div className="space-y-2">
              <Input
                placeholder="JID (e.g. 120363xxx@g.us or 628xxx@s.whatsapp.net)"
                value={manualForm.jid}
                onChange={(e) => setManualForm({ ...manualForm, jid: e.target.value })}
                className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
              />
              <Input
                placeholder="Display name"
                value={manualForm.name}
                onChange={(e) => setManualForm({ ...manualForm, name: e.target.value })}
                className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
              />
              <select
                value={manualForm.type}
                onChange={(e) => setManualForm({ ...manualForm, type: e.target.value as 'group' | 'personal' })}
                className="w-full rounded-lg border border-pdt-accent/20 bg-pdt-primary-light px-3 py-2 text-sm text-pdt-neutral"
              >
                <option value="group">Group</option>
                <option value="personal">Personal</option>
              </select>
              <Button variant="pdt" size="sm" onClick={handleAddManual} disabled={isAdding}>
                {isAdding ? 'Adding...' : 'Add'}
              </Button>
            </div>
          )}

          {/* Close picker */}
          <Button
            variant="pdtOutline"
            size="sm"
            onClick={() => { setShowPicker(false); setSearch('') }}
            className="w-full"
          >
            Close
          </Button>
        </div>
      ) : (
        <Button
          variant="pdtOutline"
          size="sm"
          onClick={() => setShowPicker(true)}
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
