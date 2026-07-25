import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import toast from 'react-hot-toast'
import { brokers, symbols as apiSymbols, providers as apiProviders, credentials as apiCredentials } from '../api/client'
import { PageHeader } from '../components/layout'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '../components/ui/card'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../components/ui/select'
import { Label } from '../components/ui/label'
import { Textarea } from '../components/ui/textarea'
import { Badge } from '../components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogFooter, AlertDialogTitle, AlertDialogDescription, AlertDialogAction, AlertDialogCancel } from '../components/ui/alert-dialog'

interface BrokerInfo {
  id: string
  label: string
  status?: string
  connected?: boolean
  last_seen?: string
}

interface Credential {
  id: string
  name: string
  provider_type: string
  api_key?: string
  secret_key?: string
  created_at?: string
  last_rotated?: string
}

type InnerTab = 'providers' | 'symbols'

const SUPPORTED_BROKERS = [
  { name: 'Paper Trading', desc: 'Simulated fills with configurable latency and slippage' },
  { name: 'Alpaca Markets', desc: 'Commission-free US equities via Alpaca API' },
  { name: 'Interactive Brokers', desc: 'Global multi-asset via IBKR TWS/Gateway' },
]

export default function IntegrationsPage() {
  const { t } = useTranslation()
  const [mainTab, setMainTab] = useState('brokers')
  const [innerTab, setInnerTab] = useState<InnerTab>('providers')

  const [brokerList, setBrokerList] = useState<BrokerInfo[]>([])
  const [brokerLoading, setBrokerLoading] = useState(true)
  const [brokerError, setBrokerError] = useState<string | null>(null)

  const [providersList, setProvidersList] = useState<unknown[]>([])
  const [symbolsList, setSymbolsList] = useState<unknown[]>([])
  const [paLoading, setPaLoading] = useState(true)
  const [msg, setMsg] = useState('')

  const [showSymbolForm, setShowSymbolForm] = useState(false)
  const [symbolForm, setSymbolForm] = useState({ ticker: '', exchange: 'NASDAQ', asset_type: 'equity', tick_size: 0.01, lot_size: 1 })
  const [showProviderForm, setShowProviderForm] = useState(false)
  const [providerForm, setProviderForm] = useState({ name: '', type: 'broker', driver: 'alpaca', config: '{}' })
  const [confirmDelete, setConfirmDelete] = useState<{ type: 'symbol' | 'provider'; id: number | string; label: string } | null>(null)

  const [credentialsList, setCredentialsList] = useState<Credential[]>([])
  const [credLoading, setCredLoading] = useState(true)
  const [credError, setCredError] = useState<string | null>(null)
  const [showCredForm, setShowCredForm] = useState(false)
  const [credName, setCredName] = useState('')
  const [credProviderType, setCredProviderType] = useState('alpaca')
  const [credApiKey, setCredApiKey] = useState('')
  const [credSecretKey, setCredSecretKey] = useState('')
  const [credSaving, setCredSaving] = useState(false)
  const [credMsg, setCredMsg] = useState('')

  const fetchBrokers = useCallback(async () => {
    try {
      setBrokerLoading(true)
      setBrokerError(null)
      const data = await brokers.list()
      setBrokerList((data as { brokers?: BrokerInfo[] }).brokers ?? [])
    } catch (err) {
      setBrokerError(err instanceof Error ? err.message : 'Failed to load brokers')
    } finally {
      setBrokerLoading(false)
    }
  }, [])

  const fetchProviders = useCallback(async () => {
    try {
      const res = await apiProviders.list() as unknown as { providers: unknown[] }
      setProvidersList(res.providers ?? [])
    } catch { /* ignore */ }
  }, [])

  const fetchSymbols = useCallback(async () => {
    try {
      const res = await apiSymbols.list() as unknown as { symbols: unknown[] }
      setSymbolsList(res.symbols ?? [])
    } catch { /* ignore */ }
  }, [])

  const fetchCredentials = useCallback(async () => {
    try {
      setCredLoading(true)
      setCredError(null)
      const data = await apiCredentials.list()
      setCredentialsList(data ?? [])
    } catch (err) {
      setCredError(err instanceof Error ? err.message : 'Failed to load credentials')
    } finally {
      setCredLoading(false)
    }
  }, [])

  useEffect(() => { fetchBrokers() }, [fetchBrokers])
  useEffect(() => { setPaLoading(true); Promise.all([fetchProviders(), fetchSymbols()]).finally(() => setPaLoading(false)) }, [fetchProviders, fetchSymbols])
  useEffect(() => { fetchCredentials() }, [fetchCredentials])

  useEffect(() => { if (mainTab === 'brokers') fetchBrokers() }, [mainTab, fetchBrokers])
  useEffect(() => { if (mainTab === 'credentials') fetchCredentials() }, [mainTab, fetchCredentials])

  const createSymbol = async () => {
    try {
      await apiSymbols.create(symbolForm as { ticker: string; name?: string; provider_id?: string })
      setMsg(`Symbol ${symbolForm.ticker} created`)
      setShowSymbolForm(false)
      setSymbolForm({ ticker: '', exchange: 'NASDAQ', asset_type: 'equity', tick_size: 0.01, lot_size: 1 })
      fetchSymbols()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Failed') }
  }

  const confirmDeleteAction = async () => {
    if (!confirmDelete) return
    try {
      if (confirmDelete.type === 'symbol') {
        await apiSymbols.delete(String(confirmDelete.id))
        fetchSymbols()
      } else {
        await apiProviders.delete(String(confirmDelete.id))
        fetchProviders()
      }
      setConfirmDelete(null)
      setMsg('')
    } catch { setMsg('Delete failed'); setConfirmDelete(null) }
  }

  const createProvider = async () => {
    try {
      const config = JSON.parse(providerForm.config)
      await apiProviders.create({ name: providerForm.name, type: providerForm.type, ...config } as { name: string; type: string })
      setMsg('Provider created')
      setShowProviderForm(false)
      fetchProviders()
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Failed') }
  }

  const testProvider = async (id: string) => {
    try {
      const res = await apiProviders.test(id)
      setMsg(res.success ? `Provider ${id} reachable (${res.latency_ms}ms)` : `Provider ${id} unreachable`)
    } catch (err) { setMsg(err instanceof Error ? err.message : 'Test failed') }
  }

  const createCredential = async () => {
    if (!credName || !credApiKey) { setCredMsg('Name and API Key are required'); return }
    setCredSaving(true)
    setCredMsg('')
    try {
      await apiCredentials.create({ name: credName, provider_type: credProviderType, api_key: credApiKey, secret_key: credSecretKey || undefined })
      setShowCredForm(false)
      setCredName(''); setCredApiKey(''); setCredSecretKey(''); setCredProviderType('alpaca')
      toast.success('Credential created')
      fetchCredentials()
    } catch (err) {
      setCredMsg(`Create failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
      toast.error('Failed to create credential')
    } finally {
      setCredSaving(false)
    }
  }

  const rotateCredential = async (id: string) => {
    try {
      await apiCredentials.rotate(id)
      toast.success('Credential rotated')
      fetchCredentials()
    } catch { toast.error('Rotation failed') }
  }

  return (
    <div>
      <PageHeader title="Integrations" subtitle="Manage brokers, data providers, symbols, and API credentials." />

      <Tabs value={mainTab} onValueChange={setMainTab} className="w-full">
        <TabsList className="mb-6">
          <TabsTrigger value="brokers">Brokers</TabsTrigger>
          <TabsTrigger value="providers-symbols">Providers &amp; Symbols</TabsTrigger>
          <TabsTrigger value="credentials">Credentials</TabsTrigger>
        </TabsList>

        <TabsContent value="brokers">
          {brokerError && (
            <Card className="border-l-4 border-l-destructive mb-4">
              <CardContent className="text-destructive text-sm pt-4">{brokerError}</CardContent>
              <CardContent className="pt-0"><Button variant="outline" size="sm" onClick={fetchBrokers}>Retry</Button></CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Configured Brokers</CardTitle>
            </CardHeader>
            <CardContent>
              {brokerLoading && !brokerList.length ? (
                <CardDescription>Loading brokers...</CardDescription>
              ) : brokerList.length === 0 ? (
                <CardDescription>No broker adapters configured. Supported: Alpaca, IBKR, Paper Trading.</CardDescription>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>ID</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Last Seen</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {brokerList.map(b => (
                      <TableRow key={b.id}>
                        <TableCell className="font-mono text-xs">{b.id}</TableCell>
                        <TableCell className="font-medium">{b.label}</TableCell>
                        <TableCell>
                          <Badge variant={b.connected ? 'default' : b.status === 'error' ? 'destructive' : 'secondary'}>
                            {b.connected ? 'Connected' : b.status || 'Unknown'}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground text-xs">{b.last_seen ? new Date(b.last_seen).toLocaleString() : '\u2014'}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>

          <Card className="mt-4">
            <CardHeader>
              <CardTitle>Supported Brokers</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-3 gap-4 mt-3">
                {SUPPORTED_BROKERS.map(b => (
                  <Card key={b.name} className="bg-secondary text-center">
                    <CardContent className="pt-6">
                      <div className="text-2xl mb-2">{String.fromCharCode(9723)}</div>
                      <div className="font-medium text-foreground">{b.name}</div>
                      <CardDescription className="text-xs mt-1">{b.desc}</CardDescription>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="providers-symbols">
          {msg && (
            <p className={`text-sm mb-3 ${msg.toLowerCase().includes('fail') || msg.toLowerCase().includes('unreachable') ? 'text-destructive' : 'text-emerald-400'}`}>
              {msg}
            </p>
          )}

          {paLoading ? (
            <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">Loading...</p></CardContent></Card>
          ) : (
            <Tabs value={innerTab} onValueChange={(v) => setInnerTab(v as InnerTab)} className="w-full">
              <TabsList className="mb-4">
                <TabsTrigger value="providers">Providers</TabsTrigger>
                <TabsTrigger value="symbols">Symbols</TabsTrigger>
              </TabsList>

              <TabsContent value="providers">
                <Button className="mb-4" onClick={() => setShowProviderForm(true)}>+ Provider</Button>
                {showProviderForm && (
                  <Card className="mb-4 max-w-[400px]">
                    <CardHeader><CardTitle>New Provider</CardTitle></CardHeader>
                    <CardContent className="flex flex-col gap-3">
                      <Input placeholder="Name" value={providerForm.name} onChange={e => setProviderForm(p => ({ ...p, name: e.target.value }))} />
                      <Select value={providerForm.type} onValueChange={v => setProviderForm(p => ({ ...p, type: v }))}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="broker">Broker</SelectItem>
                          <SelectItem value="data">Data</SelectItem>
                          <SelectItem value="llm">LLM</SelectItem>
                        </SelectContent>
                      </Select>
                      <Input placeholder="Driver" value={providerForm.driver} onChange={e => setProviderForm(p => ({ ...p, driver: e.target.value }))} />
                      <div>
                        <Label>Config (JSON)</Label>
                        <Textarea className="font-mono text-xs min-h-[80px]" value={providerForm.config} onChange={e => setProviderForm(p => ({ ...p, config: e.target.value }))} />
                      </div>
                      <div className="flex gap-2">
                        <Button onClick={createProvider}>Create</Button>
                        <Button variant="outline" onClick={() => setShowProviderForm(false)}>Cancel</Button>
                      </div>
                    </CardContent>
                  </Card>
                )}
                <Card>
                  {providersList.length === 0 ? (
                    <CardContent className="p-6"><p className="text-sm text-muted-foreground">No providers</p></CardContent>
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>ID</TableHead>
                          <TableHead>Name</TableHead>
                          <TableHead>Type</TableHead>
                          <TableHead>Driver</TableHead>
                          <TableHead>Enabled</TableHead>
                          <TableHead>Actions</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(providersList as Record<string, unknown>[]).map((p: Record<string, unknown>) => (
                          <TableRow key={String(p.id)}>
                            <TableCell className="font-mono text-xs">{String(p.id)}</TableCell>
                            <TableCell>{String(p.name)}</TableCell>
                            <TableCell>{String(p.type)}</TableCell>
                            <TableCell>{String(p.driver)}</TableCell>
                            <TableCell>
                              <Badge variant={p.is_enabled ? 'outline' : 'destructive'} className={p.is_enabled ? 'text-trading-success border-trading-success/50' : ''}>
                                {p.is_enabled ? 'Yes' : 'No'}
                              </Badge>
                            </TableCell>
                            <TableCell>
                              <div className="flex gap-1">
                                <Button variant="outline" size="sm" onClick={() => testProvider(String(p.id))}>Test</Button>
                                <Button variant="outline" size="sm" className="text-destructive border-destructive/50 hover:bg-destructive/10" onClick={() => setConfirmDelete({ type: 'provider', id: String(p.id), label: `provider ${String(p.id)}` })}>Delete</Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </Card>
              </TabsContent>

              <TabsContent value="symbols">
                <Button className="mb-4" onClick={() => setShowSymbolForm(true)}>+ Symbol</Button>
                {showSymbolForm && (
                  <Card className="mb-4 max-w-[400px]">
                    <CardHeader><CardTitle>New Symbol</CardTitle></CardHeader>
                    <CardContent className="flex flex-col gap-3">
                      <Input placeholder="Ticker" value={symbolForm.ticker} onChange={e => setSymbolForm(p => ({ ...p, ticker: e.target.value.toUpperCase() }))} />
                      <Input placeholder="Exchange" value={symbolForm.exchange} onChange={e => setSymbolForm(p => ({ ...p, exchange: e.target.value }))} />
                      <Select value={symbolForm.asset_type} onValueChange={v => setSymbolForm(p => ({ ...p, asset_type: v }))}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="equity">Equity</SelectItem>
                          <SelectItem value="forex">Forex</SelectItem>
                          <SelectItem value="crypto">Crypto</SelectItem>
                          <SelectItem value="futures">Futures</SelectItem>
                        </SelectContent>
                      </Select>
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <Label>Tick Size</Label>
                          <Input type="number" step="0.001" value={symbolForm.tick_size} onChange={e => setSymbolForm(p => ({ ...p, tick_size: parseFloat(e.target.value) }))} />
                        </div>
                        <div>
                          <Label>Lot Size</Label>
                          <Input type="number" value={symbolForm.lot_size} onChange={e => setSymbolForm(p => ({ ...p, lot_size: parseFloat(e.target.value) }))} />
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <Button onClick={createSymbol}>Create</Button>
                        <Button variant="outline" onClick={() => setShowSymbolForm(false)}>Cancel</Button>
                      </div>
                    </CardContent>
                  </Card>
                )}
                <Card>
                  {symbolsList.length === 0 ? (
                    <CardContent className="p-6"><p className="text-sm text-muted-foreground">No symbols</p></CardContent>
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Ticker</TableHead>
                          <TableHead>Exchange</TableHead>
                          <TableHead>Type</TableHead>
                          <TableHead>Active</TableHead>
                          <TableHead>Last Price</TableHead>
                          <TableHead>Actions</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(symbolsList as Record<string, unknown>[]).map((s: Record<string, unknown>) => (
                          <TableRow key={String(s.id)}>
                            <TableCell className="font-bold">{String(s.ticker)}</TableCell>
                            <TableCell>{String(s.exchange)}</TableCell>
                            <TableCell>{String(s.asset_type)}</TableCell>
                            <TableCell>
                              <Badge variant={s.is_active ? 'outline' : 'destructive'} className={s.is_active ? 'text-trading-success border-trading-success/50' : ''}>
                                {s.is_active ? 'Active' : 'Inactive'}
                              </Badge>
                            </TableCell>
                            <TableCell>{s.last_price ? `$${s.last_price}` : '\u2014'}</TableCell>
                            <TableCell>
                              <Button variant="outline" size="sm" className="text-destructive border-destructive/50 hover:bg-destructive/10" onClick={() => setConfirmDelete({ type: 'symbol', id: Number(s.id), label: `symbol #${String(s.id)}` })}>Delete</Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </Card>
              </TabsContent>
            </Tabs>
          )}
        </TabsContent>

        <TabsContent value="credentials">
          <div className="flex items-center justify-between mb-4">
            <Button onClick={() => setShowCredForm(!showCredForm)}>
              {showCredForm ? 'Cancel' : '+ New Credential'}
            </Button>
          </div>

          {credError && (
            <Card className="border-l-4 border-l-destructive mb-4">
              <CardContent className="text-destructive text-sm pt-4">{credError}</CardContent>
            </Card>
          )}

          {showCredForm && (
            <Card className="mb-4 max-w-[400px]">
              <CardHeader><CardTitle>New Credential</CardTitle></CardHeader>
              <CardContent className="flex flex-col gap-3">
                <Input placeholder="Credential name" value={credName} onChange={e => setCredName(e.target.value)} />
                <Select value={credProviderType} onValueChange={setCredProviderType}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="alpaca">Alpaca</SelectItem>
                    <SelectItem value="ibkr">Interactive Brokers</SelectItem>
                    <SelectItem value="tiingo">Tiingo</SelectItem>
                    <SelectItem value="polygon">Polygon</SelectItem>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="custom">Custom</SelectItem>
                  </SelectContent>
                </Select>
                <Input type="password" placeholder="API Key" value={credApiKey} onChange={e => setCredApiKey(e.target.value)} />
                <Input type="password" placeholder="Secret Key (optional)" value={credSecretKey} onChange={e => setCredSecretKey(e.target.value)} />
                {credMsg && <p className="text-sm text-destructive">{credMsg}</p>}
                <Button onClick={createCredential} disabled={credSaving}>
                  {credSaving ? 'Creating...' : 'Create Credential'}
                </Button>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Stored Credentials</CardTitle>
            </CardHeader>
            <CardContent>
              {credLoading ? (
                <CardDescription>Loading credentials...</CardDescription>
              ) : credentialsList.length === 0 ? (
                <CardDescription>No credentials stored. Add your first API key above.</CardDescription>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Provider</TableHead>
                      <TableHead>API Key</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead>Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {credentialsList.map(c => (
                      <TableRow key={c.id}>
                        <TableCell className="font-medium">{c.name}</TableCell>
                        <TableCell><Badge variant="secondary">{c.provider_type}</Badge></TableCell>
                        <TableCell className="text-muted-foreground font-mono text-xs">{c.api_key ? `${c.api_key.slice(0, 8)}...` : '\u2014'}</TableCell>
                        <TableCell className="text-muted-foreground text-xs">{c.created_at ? new Date(c.created_at).toLocaleDateString() : '\u2014'}</TableCell>
                        <TableCell>
                          <Button variant="outline" size="sm" onClick={() => rotateCredential(c.id)}>Rotate</Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <AlertDialog open={!!confirmDelete} onOpenChange={(open) => !open && setConfirmDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {confirmDelete?.type}</AlertDialogTitle>
            <AlertDialogDescription>Delete {confirmDelete?.label}? This action cannot be undone.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setConfirmDelete(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive hover:bg-destructive/90" onClick={confirmDeleteAction}>Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
