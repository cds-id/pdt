import { useState } from 'react'
import { Save, Trash2, ExternalLink, RefreshCw } from 'lucide-react'

import {
  useGetComposioConfigQuery,
  useSaveComposioConfigMutation,
  useDeleteComposioConfigMutation,
  useListComposioConnectionsQuery,
  useSaveComposioAuthConfigMutation,
  useInitiateComposioConnectionMutation,
  useSyncComposioConnectionsMutation,
  useDeleteComposioConnectionMutation
} from '@/infrastructure/services/composio.service'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { DataCard, StatusBadge } from '@/presentation/components/common'

const TOOLKITS = [
  { slug: 'gmail', name: 'Gmail', description: 'Send and read emails' },
  { slug: 'notion', name: 'Notion', description: 'Create and query pages' },
  { slug: 'googlecalendar', name: 'Google Calendar', description: 'Manage events' },
  { slug: 'linkedin', name: 'LinkedIn', description: 'Create posts, view profile' }
]

export function ComposioSettings() {
  const { data: config } = useGetComposioConfigQuery()
  const { data: connections = [] } = useListComposioConnectionsQuery(undefined, {
    skip: !config?.configured
  })
  const [saveConfig, { isLoading: isSaving }] = useSaveComposioConfigMutation()
  const [deleteConfig] = useDeleteComposioConfigMutation()
  const [saveAuthConfig] = useSaveComposioAuthConfigMutation()
  const [initiateConnection] = useInitiateComposioConnectionMutation()
  const [syncConnections, { isLoading: isSyncing }] = useSyncComposioConnectionsMutation()
  const [deleteConnection] = useDeleteComposioConnectionMutation()

  const [apiKey, setApiKey] = useState('')
  const [authConfigInputs, setAuthConfigInputs] = useState<Record<string, string>>({})
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const handleSaveKey = async () => {
    setMessage(null)
    try {
      await saveConfig({ api_key: apiKey }).unwrap()
      setApiKey('')
      setMessage({ type: 'success', text: 'API key saved and validated!' })
    } catch {
      setMessage({ type: 'error', text: 'Invalid API key or connection failed.' })
    }
  }

  const handleRemoveKey = async () => {
    if (!confirm('Remove Composio API key and all connections?')) return
    try {
      await deleteConfig().unwrap()
      setMessage(null)
    } catch {
      setMessage({ type: 'error', text: 'Failed to remove API key.' })
    }
  }

  const handleSaveAuthConfig = async (toolkit: string) => {
    const authConfigId = authConfigInputs[toolkit]
    if (!authConfigId) return
    try {
      await saveAuthConfig({ toolkit, auth_config_id: authConfigId }).unwrap()
      setAuthConfigInputs((prev) => ({ ...prev, [toolkit]: '' }))
    } catch (err) {
      console.error('Failed to save auth config:', err)
    }
  }

  const handleConnect = async (toolkit: string) => {
    try {
      const redirectURI = window.location.origin + '/dashboard/settings'
      const result = await initiateConnection({
        toolkit,
        redirect_uri: redirectURI
      }).unwrap()

      if (result.redirect_url) {
        const popup = window.open(result.redirect_url, '_blank', 'width=600,height=700')
        // Poll for popup close and sync connections
        const interval = setInterval(() => {
          if (popup?.closed) {
            clearInterval(interval)
            syncConnections()
          }
        }, 1000)
      }
    } catch (err) {
      console.error('Failed to initiate connection:', err)
    }
  }

  const handleDisconnect = async (toolkit: string) => {
    if (!confirm(`Disconnect ${toolkit}?`)) return
    try {
      await deleteConnection(toolkit).unwrap()
    } catch (err) {
      console.error('Failed to disconnect:', err)
    }
  }

  const getConnection = (toolkit: string) => {
    return connections.find((c) => c.toolkit === toolkit)
  }

  return (
    <>
      <DataCard title="Composio — External Tools">
        <p className="mb-3 text-xs text-pdt-neutral/50">
          Connect external services (Gmail, Notion, Calendar, LinkedIn) to use them as AI agent tools.
          Get your API key from{' '}
          <a
            href="https://app.composio.dev/settings"
            target="_blank"
            rel="noopener noreferrer"
            className="text-pdt-accent underline"
          >
            app.composio.dev
          </a>
          .
        </p>

        <div className="flex items-center gap-2">
          <Input
            type="password"
            placeholder="Composio API Key"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            className="border-pdt-accent/20 bg-pdt-primary-light text-pdt-neutral placeholder:text-pdt-neutral/40"
          />
          <Button
            type="button"
            variant="pdt"
            size="sm"
            disabled={!apiKey || isSaving}
            onClick={handleSaveKey}
          >
            <Save className="mr-1 size-3" />
            {isSaving ? 'Saving...' : 'Save'}
          </Button>
        </div>

        <div className="mt-2 flex items-center gap-2 text-xs">
          {config?.configured ? (
            <>
              <StatusBadge variant="success">Configured</StatusBadge>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={handleRemoveKey}
              >
                <Trash2 className="size-3 text-red-400" />
              </Button>
            </>
          ) : (
            <StatusBadge variant="danger">Not configured</StatusBadge>
          )}
        </div>

        {message && (
          <p className={`mt-2 text-sm ${message.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
            {message.text}
          </p>
        )}
      </DataCard>

      {config?.configured && (
        <DataCard
          title="Connected Services"
        >
          <div className="mb-3 flex items-center justify-between">
            <p className="text-xs text-pdt-neutral/50">
              Connect services to let your AI agents use them. Get the Auth Config ID from{' '}
              <a
                href="https://app.composio.dev/auth_configs"
                target="_blank"
                rel="noopener noreferrer"
                className="text-pdt-accent underline"
              >
                Composio dashboard
              </a>
              .
            </p>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={isSyncing}
              onClick={() => syncConnections()}
            >
              <RefreshCw className={`size-3 ${isSyncing ? 'animate-spin' : ''}`} />
            </Button>
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            {TOOLKITS.map((tk) => {
              const conn = getConnection(tk.slug)
              const hasAuthConfig = !!conn?.auth_config_id
              const isActive = conn?.status === 'active'

              return (
                <div
                  key={tk.slug}
                  className="rounded-lg border border-pdt-neutral/10 p-3"
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-pdt-neutral">{tk.name}</p>
                      <p className="text-xs text-pdt-neutral/50">{tk.description}</p>
                    </div>
                    <StatusBadge variant={isActive ? 'success' : hasAuthConfig ? 'warning' : 'danger'}>
                      {isActive ? 'Connected' : hasAuthConfig ? 'Ready' : 'No Config'}
                    </StatusBadge>
                  </div>

                  <div className="mt-2 flex items-center gap-2">
                    <Input
                      type="text"
                      placeholder="Auth Config ID (e.g. ac_...)"
                      value={authConfigInputs[tk.slug] || ''}
                      onChange={(e) =>
                        setAuthConfigInputs((prev) => ({ ...prev, [tk.slug]: e.target.value }))
                      }
                      className="h-7 border-pdt-accent/20 bg-pdt-primary-light text-xs text-pdt-neutral placeholder:text-pdt-neutral/40"
                    />
                    <Button
                      type="button"
                      variant="pdt"
                      size="sm"
                      disabled={!authConfigInputs[tk.slug]}
                      onClick={() => handleSaveAuthConfig(tk.slug)}
                    >
                      <Save className="size-3" />
                    </Button>
                  </div>

                  {hasAuthConfig && (
                    <div className="mt-1 text-xs text-pdt-neutral/40">
                      ID: {conn.auth_config_id}
                    </div>
                  )}

                  <div className="mt-2 flex gap-2">
                    {hasAuthConfig && !isActive && (
                      <Button
                        type="button"
                        variant="pdtOutline"
                        size="sm"
                        onClick={() => handleConnect(tk.slug)}
                      >
                        <ExternalLink className="mr-1 size-3" />
                        Connect
                      </Button>
                    )}
                    {isActive && (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDisconnect(tk.slug)}
                      >
                        <Trash2 className="mr-1 size-3 text-red-400" />
                        <span className="text-red-400">Disconnect</span>
                      </Button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </DataCard>
      )}
    </>
  )
}
