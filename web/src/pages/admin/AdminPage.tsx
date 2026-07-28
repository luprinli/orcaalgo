import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { admin, models, reconciliation, dataValidate } from '../../api/client'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { Badge } from '../../components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../../components/ui/tabs'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogFooter, AlertDialogTitle, AlertDialogDescription, AlertDialogAction, AlertDialogCancel } from '../../components/ui/alert-dialog'
import MetricCard from '../../components/MetricCard'
import InfrastructureTab from './InfrastructureTab'
import AlertsTab from './AlertsTab'

export default function AdminPage() {
  const { t } = useTranslation()
  const [tab, setTab] = useState<'health' | 'users' | 'audit' | 'errors' | 'email' | 'seed' | 'models' | 'reconciliation' | 'dataValidate' | 'infrastructure' | 'alerts'>('health')
  /* eslint-disable @typescript-eslint/no-explicit-any */
  const [health, setHealth] = useState<any>(null)
  const [systemHealth, setSystemHealth] = useState<any>(null)
  const [users, setUsers] = useState<any[]>([])
  const [auditLogs, setAuditLogs] = useState<any[]>([])
  const [errorLogs, setErrorLogs] = useState<any[]>([])
  const [seedInfo, setSeedInfo] = useState<any>(null)
  /* eslint-enable @typescript-eslint/no-explicit-any */
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')
  const [emailForm, setEmailForm] = useState({ host: '', port: '587', username: '', password: '', from: '', from_name: '' })
  const [confirmSeed, setConfirmSeed] = useState(false)
  const [modelHash, setModelHash] = useState('')
  const [modelType, setModelType] = useState('')
  const [modelName, setModelName] = useState('')
  const [modelBrier, setModelBrier] = useState('')
  const [modelRoc, setModelRoc] = useState('')
  const [modelResult, setModelResult] = useState<Record<string, unknown> | null>(null)
  const [compareHash, setCompareHash] = useState('')
  const [compareResult, setCompareResult] = useState<Record<string, unknown> | null>(null)
  const [latestType, setLatestType] = useState('')
  const [latestResult, setLatestResult] = useState<Record<string, unknown> | null>(null)
  const [reconDate, setReconDate] = useState(new Date().toISOString().slice(0, 10))
  const [reconResult, setReconResult] = useState<Record<string, unknown> | null>(null)
  const [dataValResult, setDataValResult] = useState<any>(null)

  const fetchHealth = useCallback(async () => {
    setLoading(true)
    try {
      const [h, sh] = await Promise.all([admin.health(), admin.systemHealth()])
      setHealth(h)
      setSystemHealth(sh)
    } catch { setMsg(t('admin:failedToLoadHealth', 'Failed to load health')) }
    finally { setLoading(false) }
  }, [])

  const fetchUsers = useCallback(async () => {
    setLoading(true)
    try {
      const res = await admin.users()
      setUsers(res.users ?? [])
    } catch { setMsg(t('admin:failedToLoadUsers', 'Failed to load users')) }
    finally { setLoading(false) }
  }, [])

  const fetchAudit = useCallback(async (component?: string) => {
    setLoading(true)
    try {
      const res = await admin.auditLogs({ component, limit: 50 })
      setAuditLogs(res)
    } catch { setMsg(t('admin:failedToLoadAuditLogs', 'Failed to load audit logs')) }
    finally { setLoading(false) }
  }, [])

  const fetchErrors = useCallback(async (params?: { severity?: string; component?: string }) => {
    setLoading(true)
    try {
      const res = await admin.errorLogs({ ...params, limit: 50 })
      setErrorLogs(res)
    } catch { setMsg(t('admin:failedToLoadErrorLogs', 'Failed to load error logs')) }
    finally { setLoading(false) }
  }, [])

  const fetchSeedInfo = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/admin/info').then(r => r.json())
      setSeedInfo(res)
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  const handleSeed = () => {
    setConfirmSeed(true)
  }

  const confirmSeedDatabase = async () => {
    setConfirmSeed(false)
    try {
      const res = await admin.seed(true) as { seeded?: boolean }
      setMsg(res?.seeded ? t('admin:dbSeeded', 'Database seeded successfully') : t('admin:seedFailed', 'Seed failed'))
      fetchSeedInfo()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('admin:seedFailed', 'Seed failed'))
    }
  }

  const handleUserToggle = async (userId: string, enable: boolean) => {
    try {
      if (enable) {
        const res = await fetch(`/api/v1/admin/users/${userId}/enable`, { method: 'PUT' }).then(r => r.json())
        setMsg(res?.enabled ? t('admin:userEnabled', 'User enabled') : t('admin:failed', 'Failed'))
      } else {
        const res = await fetch(`/api/v1/admin/users/${userId}/disable`, { method: 'PUT' }).then(r => r.json())
        setMsg(res?.disabled ? t('admin:userDisabled', 'User disabled') : t('admin:failed', 'Failed'))
      }
      fetchUsers()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('admin:toggleFailed', 'Toggle failed'))
    }
  }

  const handleTestEmail = async () => {
    try {
      const res = await fetch('/api/v1/admin/email/test', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(emailForm) }).then(r => r.json())
      setMsg(res.ok ? t('admin:emailTestSuccess', 'Email test successful') : t('admin:emailTestFailed', 'Email test failed: {{error}}', { error: res.error }))
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('admin:emailTestFailed', 'Email test failed', { error: '' }))
    }
  }

  const handleSaveEmail = async () => {
    try {
      const res = await fetch('/api/v1/admin/email/config', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(emailForm) }).then(r => r.json())
      setMsg(res.ok ? t('admin:emailConfigSaved', 'Email config saved') : t('admin:failed', 'Failed'))
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('settings:updateFailed', 'Save failed'))
    }
  }

	const tabLabels: Record<string, string> = {
		health: t('admin:health', 'Health'),
		users: t('admin:users', 'Users'),
		audit: t('admin:audit', 'Audit'),
		errors: t('admin:errors', 'Errors'),
		email: t('admin:email', 'Email'),
		seed: t('admin:seed', 'Seed'),
		models: t('admin:models', 'ML Models'),
		reconciliation: t('admin:reconciliation', 'Reconciliation'),
		dataValidate: t('admin:dataValidate', 'Data Quality'),
		infrastructure: 'Infrastructure',
		alerts: 'Alerts',
	}

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="m-0">{t('admin:title', 'Admin')}</h1>
      </div>

      <Tabs value={tab} onValueChange={(v) => { setTab(v as typeof tab); setMsg('') }} className="w-full">
        <TabsList className="mb-4 flex-wrap">
          {(['health', 'users', 'audit', 'errors', 'email', 'seed', 'models', 'reconciliation', 'dataValidate', 'infrastructure', 'alerts'] as const).map(mt => (
            <TabsTrigger key={mt} value={mt}>{tabLabels[mt]}</TabsTrigger>
          ))}
        </TabsList>

        {msg && <p className={`text-sm mb-2 ${msg.includes('fail') || msg.includes('Fail') ? 'text-destructive' : 'text-emerald-400'}`}>{msg}</p>}

        <TabsContent value="health">
          <Button variant="outline" className="mb-4" onClick={fetchHealth}>{loading ? t('admin:loading', 'Loading...') : t('admin:refreshHealth', 'Refresh Health')}</Button>
          {health && (
            <Card className="mb-4">
              <CardHeader><CardTitle>{t('admin:health', 'Health')}</CardTitle></CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 gap-4">
                  <Card>
                    <CardContent className="p-4">
                      <p className="text-sm text-muted-foreground">{t('common:status', 'Status')}</p>
                      <p className="text-xl font-bold mt-1" style={{ color: health.healthy ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{health.healthy ? t('admin:healthy', 'Healthy') : t('admin:unhealthy', 'Unhealthy')}</p>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="p-4">
                      <p className="text-sm text-muted-foreground">{t('admin:database', 'Database')}</p>
                      <p className="text-xl font-bold mt-1">{health.components?.database ?? '--'}</p>
                    </CardContent>
                  </Card>
                </div>
                <p className="text-sm text-muted-foreground mt-2">{health.timestamp ? new Date(health.timestamp).toLocaleString() : ''}</p>
              </CardContent>
            </Card>
          )}
          {systemHealth && (
            <Card>
              <CardHeader><CardTitle>{t('admin:systemHealth', 'System Health')}</CardTitle></CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {Object.entries(systemHealth.checks ?? {}).map(([key, val]: [string, any]) => (
                    <Card key={key}>
                      <CardContent className="p-4">
                        <p className="text-sm text-muted-foreground">{key}</p>
                        <p className="text-xl font-bold mt-1" style={{ color: val.status === 'ok' ? 'var(--trading-success)' : 'var(--trading-danger)' }}>{val.status}</p>
                        <p className="text-xs text-muted-foreground">{val.message}</p>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
          {systemHealth && (
            <Card>
              <CardHeader><CardTitle>Runtime Metrics</CardTitle></CardHeader>
              <CardContent>
                <div className="grid grid-cols-3 sm:grid-cols-4 gap-3">
                  {Object.entries(systemHealth).filter(([k]) => !k.endsWith('checks') && !k.endsWith('_at') && !k.endsWith('_time')).map(([k, v]) => (
                    typeof v === 'number' || typeof v === 'boolean' ? (
                      <MetricCard key={k} label={k.replace(/_/g, ' ')} value={typeof v === 'boolean' ? (v ? 'Yes' : 'No') : String(v)} color={typeof v === 'boolean' ? (v ? 'positive' : 'negative') : undefined} />
                    ) : null
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="users">
          <Button variant="outline" className="mb-4" onClick={fetchUsers}>{loading ? t('admin:loading', 'Loading...') : t('admin:refreshUsers', 'Refresh Users')}</Button>
          <Card>
            {users.length === 0 ? <CardContent className="p-6"><p className="text-sm text-muted-foreground">{t('admin:noUsers', 'No users')}</p></CardContent> : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('admin:table.user', 'Username')}</TableHead>
                    <TableHead>{t('auth:email', 'Email')}</TableHead>
                    <TableHead>{t('admin:table.role', 'Roles')}</TableHead>
                    <TableHead>{t('admin:verified', 'Verified')}</TableHead>
                    <TableHead>2FA</TableHead>
                    <TableHead>{t('common:active', 'Active')}</TableHead>
                    <TableHead>{t('admin:table.actions', 'Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {users.map((u: any) => (
                    <TableRow key={u.id}>
                      <TableCell>{u.username}</TableCell>
                      <TableCell>{u.email}</TableCell>
                      <TableCell>{(u.roles ?? []).join(', ')}</TableCell>
                      <TableCell>
                        <Badge variant={u.is_verified ? 'outline' : 'destructive'} className={u.is_verified ? 'text-trading-success border-trading-success/50' : ''}>
                          {u.is_verified ? t('common:yes', 'Yes') : t('common:no', 'No')}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={u.totp_enabled ? 'outline' : 'destructive'} className={u.totp_enabled ? 'text-trading-success border-trading-success/50' : ''}>
                          {u.totp_enabled ? t('admin:on', 'On') : t('admin:off', 'Off')}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant={u.is_active ? 'outline' : 'destructive'} className={u.is_active ? 'text-trading-success border-trading-success/50' : ''}>
                          {u.is_active ? t('common:active', 'Active') : t('common:disabled', 'Disabled')}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Button variant="outline" size="sm" onClick={() => handleUserToggle(u.id, !u.is_active)}>
                          {u.is_active ? t('common:disable', 'Disable') : t('common:enable', 'Enable')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="audit">
          <Button variant="outline" className="mb-4" onClick={() => fetchAudit()}>{loading ? t('admin:loading', 'Loading...') : t('admin:refreshAudit', 'Refresh Audit')}</Button>
          <Card className="max-h-[600px] overflow-y-auto">
            {auditLogs.length === 0 ? <CardContent className="p-6"><p className="text-sm text-muted-foreground">{t('admin:noAuditLogs', 'No audit logs')}</p></CardContent> : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('admin:table.timestamp', 'Time')}</TableHead>
                    <TableHead>{t('admin:table.event', 'Action')}</TableHead>
                    <TableHead>{t('admin:resource', 'Resource')}</TableHead>
                    <TableHead>{t('admin:table.user', 'User')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {auditLogs.map((l: { id?: string; created_at?: string; action?: string; resource_type?: string; resource_id?: string; user_id?: string }, i: number) => (
                    <TableRow key={l.id ?? i}>
                      <TableCell className="text-xs">{l.created_at ? new Date(l.created_at).toLocaleString() : '--'}</TableCell>
                      <TableCell>{l.action ?? '--'}</TableCell>
                      <TableCell>{(l.resource_type ?? '') + (l.resource_id ? ': ' + l.resource_id : '')}</TableCell>
                      <TableCell>{l.user_id?.slice(0, 8) ?? '--'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="errors">
          <Button variant="outline" className="mb-4" onClick={() => fetchErrors()}>{loading ? t('admin:loading', 'Loading...') : t('admin:refreshErrors', 'Refresh Errors')}</Button>
          <Card className="max-h-[600px] overflow-y-auto">
            {errorLogs.length === 0 ? <CardContent className="p-6"><p className="text-sm text-muted-foreground">{t('admin:noErrorLogs', 'No error logs')}</p></CardContent> : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('admin:table.timestamp', 'Time')}</TableHead>
                    <TableHead>{t('admin:table.severity', 'Severity')}</TableHead>
                    <TableHead>{t('admin:table.component', 'Component')}</TableHead>
                    <TableHead>{t('admin:table.message', 'Message')}</TableHead>
                    <TableHead>{t('admin:resolved', 'Resolved')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {errorLogs.map((e: any, i: number) => (
                    <TableRow key={e.id ?? i}>
                      <TableCell className="text-xs">{e.timestamp ? new Date(e.timestamp).toLocaleString() : '--'}</TableCell>
                      <TableCell>
                        <Badge variant={e.severity === 'error' || e.severity === 'critical' ? 'destructive' : 'outline'} className={e.severity !== 'error' && e.severity !== 'critical' ? 'text-trading-warning border-yellow-500/50' : ''}>
                          {e.severity}
                        </Badge>
                      </TableCell>
                      <TableCell>{e.component}</TableCell>
                      <TableCell>{e.message}</TableCell>
                      <TableCell>{e.resolved != null ? (e.resolved ? t('common:yes', 'Yes') : t('common:no', 'No')) : '--'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
        </TabsContent>

        <TabsContent value="email">
          <Card className="max-w-[450px]">
            <CardHeader><CardTitle>{t('admin:emailConfig', 'Email Configuration')}</CardTitle></CardHeader>
            <CardContent className="flex flex-col gap-3">
              {(['host', 'port', 'username', 'password', 'from', 'from_name'] as const).map(f => (
                <div key={f}>
                  <Label className="capitalize">{f.replace('_', ' ')}</Label>
                  <Input type={f === 'password' ? 'password' : 'text'} value={emailForm[f]} onChange={e => setEmailForm(p => ({ ...p, [f]: e.target.value }))} />
                </div>
              ))}
              <div className="flex gap-2">
                <Button variant="outline" onClick={handleTestEmail}>{t('common:test', 'Test')}</Button>
                <Button onClick={handleSaveEmail}>{t('common:save', 'Save')}</Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="seed">
          <div className="flex gap-2 mb-4">
            <Button variant="outline" onClick={fetchSeedInfo}>{loading ? t('admin:loading', 'Loading...') : t('common:info', 'Info')}</Button>
            <Button variant="destructive" onClick={handleSeed}>{t('admin:seedDatabase', 'Seed Database')}</Button>
          </div>
          {seedInfo && (
            <Card>
              <CardHeader><CardTitle>{t('admin:databaseStatus', 'Database Status')}</CardTitle></CardHeader>
              <CardContent>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {Object.entries(seedInfo).filter(([k]) => k !== 'admin_credentials').map(([key, val]: [string, any]) => (
                    <Card key={key}>
                      <CardContent className="p-4">
                        <p className="text-sm text-muted-foreground capitalize">{key.replace(/_/g, ' ')}</p>
                        <p className="text-xl font-bold mt-1">{val}</p>
                      </CardContent>
                    </Card>
                  ))}
                </div>
                {seedInfo.admin_credentials && (
                  <div className="mt-4 p-3 rounded-md bg-amber-500/10 border border-amber-500/30">
                    <span className="text-sm text-muted-foreground">{t('admin:adminCredentials', 'Admin: {{user}} / {{pw}}', { user: seedInfo.admin_credentials.username, pw: seedInfo.admin_credentials.password })}</span>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </TabsContent>

      <TabsContent value="models">
        <Card className="mb-4">
          <CardHeader><CardTitle>Register Model</CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-3 max-w-[400px]">
            <Input placeholder="Model Hash" value={modelHash} onChange={e => setModelHash(e.target.value)} />
            <Input placeholder="Model Type" value={modelType} onChange={e => setModelType(e.target.value)} />
            <Input placeholder="Model Name" value={modelName} onChange={e => setModelName(e.target.value)} />
            <Input placeholder="Brier Score" type="number" step="0.01" value={modelBrier} onChange={e => setModelBrier(e.target.value)} />
            <Input placeholder="ROC AUC" type="number" step="0.01" value={modelRoc} onChange={e => setModelRoc(e.target.value)} />
            <Button onClick={async () => { try { const r = await models.register({ model_hash: modelHash, model_type: modelType, model_name: modelName, brier_score: parseFloat(modelBrier), roc_auc: parseFloat(modelRoc) }); setModelResult(r); setMsg('Model registered') } catch (e) { setMsg(String(e)) } }}>Register</Button>
            {modelResult && <pre className="text-xs bg-muted p-2 rounded">{JSON.stringify(modelResult, null, 2)}</pre>}
          </CardContent>
        </Card>
        <Card className="mb-4">
          <CardHeader><CardTitle>Compare Model</CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-3 max-w-[400px]">
            <Input placeholder="Model Hash" value={compareHash} onChange={e => setCompareHash(e.target.value)} />
            <Button variant="outline" onClick={async () => { try { const r = await models.compare(compareHash); setCompareResult(r); setMsg(r.exists ? 'Model exists in registry' : 'Model not found') } catch (e) { setMsg(String(e)) } }}>Lookup</Button>
            {compareResult && <pre className="text-xs bg-muted p-2 rounded">{JSON.stringify(compareResult, null, 2)}</pre>}
          </CardContent>
        </Card>
        <Card className="mb-4">
          <CardHeader><CardTitle>Latest Model by Type</CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-3 max-w-[400px]">
            <Input placeholder="Model Type" value={latestType} onChange={e => setLatestType(e.target.value)} />
            <Button variant="outline" onClick={async () => { try { const r = await models.latest(latestType); setLatestResult(r) } catch (e) { setMsg(String(e)) } }}>Get Latest</Button>
            {latestResult && <pre className="text-xs bg-muted p-2 rounded">{JSON.stringify(latestResult, null, 2)}</pre>}
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="reconciliation">
        <Card>
          <CardHeader><CardTitle>Daily Trade Reconciliation</CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-3 max-w-[400px]">
            <Input type="date" value={reconDate} onChange={e => setReconDate(e.target.value)} />
            <Button onClick={async () => { try { const r = await reconciliation.daily(reconDate); setReconResult(r) } catch (e) { setMsg(String(e)) } }}>Run Reconciliation</Button>
            {reconResult && (
              <div className="space-y-2">
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <span className="text-muted-foreground">Date:</span><span>{reconResult.date as string}</span>
                  <span className="text-muted-foreground">Matched:</span><span className="text-trading-success">{(reconResult.matched as number) ?? 0}</span>
                  <span className="text-muted-foreground">Missing:</span><span className="text-trading-warning">{(reconResult.missing as number) ?? 0}</span>
                  <span className="text-muted-foreground">Extra:</span><span className="text-trading-info">{(reconResult.extra as number) ?? 0}</span>
                  <span className="text-muted-foreground">Price Discrepancies:</span><span className="text-trading-danger">{(reconResult.price_discrepancies as number) ?? 0}</span>
                </div>
                {(reconResult.details as unknown[])?.length > 0 && (
                  <Table>
                    <TableHeader><TableRow><TableHead>Trade ID</TableHead><TableHead>Internal</TableHead><TableHead>Broker</TableHead><TableHead>Diff %</TableHead></TableRow></TableHeader>
                    <TableBody>{(reconResult.details as Array<Record<string,unknown>>).map((d: Record<string,unknown>, i: number) => (
                      <TableRow key={i}><TableCell className="text-xs">{(d.trade_id as string)?.slice(0, 12)}</TableCell><TableCell className="tabular-nums">{String(d.internal_price)}</TableCell><TableCell className="tabular-nums">{String(d.broker_price)}</TableCell><TableCell className={Number(d.diff_pct) > 0.5 ? 'text-trading-danger' : 'text-trading-success'}>{(Number(d.diff_pct)).toFixed(2)}%</TableCell></TableRow>
                    ))}</TableBody>
                  </Table>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="dataValidate">
        <Card>
          <CardHeader><CardTitle>Data Quality Validation</CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-3 max-w-[400px]">
            <Button onClick={async () => { try { const r = await dataValidate.run(); setDataValResult(r) } catch (e) { setMsg(String(e)) } }}>Run Data Quality Check</Button>
            {dataValResult && (
              <div className="space-y-2">
                <Badge variant={(dataValResult.status as string) === 'ok' ? 'success' : 'destructive'}>{(dataValResult.status as string) ?? 'Unknown'}</Badge>
                <pre className="text-xs bg-muted p-2 rounded max-h-[400px] overflow-auto">{JSON.stringify(dataValResult, null, 2)}</pre>
              </div>
            )}
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="infrastructure">
        <InfrastructureTab />
      </TabsContent>

      <TabsContent value="alerts">
        <AlertsTab />
      </TabsContent>

      <AlertDialog open={confirmSeed} onOpenChange={(open) => !open && setConfirmSeed(false)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('admin:seedDatabaseTitle', 'Seed Database')}</AlertDialogTitle>
            <AlertDialogDescription>{t('admin:seedDatabaseConfirm', 'This will reset the database to its initial state. All existing data will be replaced. Are you sure?')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setConfirmSeed(false)}>{t('common:cancel', 'Cancel')}</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive hover:bg-destructive/90" onClick={confirmSeedDatabase}>{t('admin:seedDatabase', 'Seed Database')}</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      </Tabs>
    </div>
  )
}
