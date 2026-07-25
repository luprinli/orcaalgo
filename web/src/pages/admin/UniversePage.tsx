import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { universe } from '../../api/client'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from '../../components/ui/select'
import { Label } from '../../components/ui/label'
import { Textarea } from '../../components/ui/textarea'
import { Badge } from '../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'

export default function UniversePage() {
  const { t } = useTranslation()
  /* eslint-disable @typescript-eslint/no-explicit-any */
  const [symbols, setSymbols] = useState<any[]>([])
  const [configs, setConfigs] = useState<any[]>([])
  /* eslint-enable @typescript-eslint/no-explicit-any */
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')
  const [overrideTicker, setOverrideTicker] = useState('')
  const [overrideAction, setOverrideAction] = useState<'add' | 'remove'>('add')
  const [showConfigForm, setShowConfigForm] = useState(false)
  const [configForm, setConfigForm] = useState({ name: '', profile_id: 'default', asset_class_filters: '{}', dynamic_triggers: '{}' })

  const fetchUniverse = useCallback(async () => {
    try {
      const [cur, cfg] = await Promise.all([
        universe.current(),
        universe.configs(),
      ])
      setSymbols(cur.symbols ?? [])
      setConfigs(cfg.configs ?? [])
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchUniverse() }, [fetchUniverse])

  const handleOverride = async () => {
    if (!overrideTicker) return
    try {
      await universe.override(overrideTicker.toUpperCase(), overrideAction)
      setMsg(overrideAction === 'add' ? t('universe:symbolAdded', 'Symbol {{ticker}} added', { ticker: overrideTicker.toUpperCase() }) : t('universe:symbolRemoved', 'Symbol {{ticker}} removed', { ticker: overrideTicker.toUpperCase() }))
      setOverrideTicker('')
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : t('universe:failed', 'Override failed')) }
  }

  const handleRefresh = async () => {
    try {
      const res = await universe.refresh()
      setMsg(t('universe:refreshed', 'Universe refreshed: {{total}} symbols', { total: res.total }))
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : t('universe:refreshedFailed', 'Refresh failed')) }
  }

  const createConfig = async () => {
    try {
      await universe.createConfig({
        name: configForm.name, profile_id: configForm.profile_id,
        asset_class_filters: JSON.parse(configForm.asset_class_filters),
        dynamic_triggers: JSON.parse(configForm.dynamic_triggers),
      })
      setMsg(t('universe:configCreated', 'Config created'))
      setShowConfigForm(false)
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : t('universe:failed', 'Failed')) }
  }

  const activateConfig = async (id: string) => {
    try {
      await universe.activateConfig(id)
      setMsg(t('universe:configActivated', 'Config activated'))
      fetchUniverse()
    } catch (err) { setMsg(err instanceof Error ? err.message : t('universe:failed', 'Failed')) }
  }

  if (loading) return <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">{t('universe:loading', 'Loading universe...')}</p></CardContent></Card>

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="m-0">{t('universe:title', 'Universe Management')}</h1>
        <Button variant="outline" onClick={handleRefresh}>{t('universe:refresh', 'Refresh')}</Button>
      </div>

      {msg && <p className={`text-sm mb-2 ${msg.includes('fail') ? 'text-destructive' : 'text-emerald-400'}`}>{msg}</p>}

      <div className="grid grid-cols-2 gap-6 mb-4">
        <Card>
          <CardHeader><CardTitle>{t('universe:currentUniverse', 'Current Universe ({{n}})', { n: symbols.length })}</CardTitle></CardHeader>
          <CardContent>
            <div className="flex gap-2 mb-4">
              <Input placeholder={t('universe:ticker', 'Ticker')} value={overrideTicker} onChange={e => setOverrideTicker(e.target.value.toUpperCase())} onKeyDown={e => e.key === 'Enter' && handleOverride()} />
              <Select value={overrideAction} onValueChange={v => setOverrideAction(v as 'add' | 'remove')}>
                <SelectTrigger className="w-[90px]"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="add">{t('universe:add', 'Add')}</SelectItem>
                  <SelectItem value="remove">{t('universe:remove', 'Remove')}</SelectItem>
                </SelectContent>
              </Select>
              <Button onClick={handleOverride} disabled={!overrideTicker}>{t('universe:go', 'Go')}</Button>
            </div>
            <div className="max-h-[400px] overflow-y-auto">
              {symbols.length === 0 ? <p className="text-sm text-muted-foreground">{t('universe:noSymbols', 'Empty universe')}</p> : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('universe:table.ticker', 'Ticker')}</TableHead>
                      <TableHead>{t('universe:table.exchange', 'Exchange')}</TableHead>
                      <TableHead>{t('universe:table.type', 'Type')}</TableHead>
                      <TableHead>{t('universe:table.active', 'Active')}</TableHead>
                      <TableHead>{t('common:price', 'Price')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                    {symbols.map((s: any) => (
                      <TableRow key={s.id}>
                        <TableCell className="font-bold">{s.ticker}</TableCell>
                        <TableCell>{s.exchange}</TableCell>
                        <TableCell>{s.asset_type}</TableCell>
                        <TableCell>
                          <Badge variant={s.is_active ? 'outline' : 'destructive'} className={s.is_active ? 'text-trading-success border-trading-success/50' : ''}>
                            {s.is_active ? t('common:active', 'Active') : t('common:inactive', 'Inactive')}
                          </Badge>
                        </TableCell>
                        <TableCell>{s.last_price ? `$${s.last_price}` : t('common:noData', '--')}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>{t('universe:configs', 'Configs')}</CardTitle>
              <Button onClick={() => setShowConfigForm(true)}>{t('universe:addConfig', '+ Config')}</Button>
            </div>
          </CardHeader>
          <CardContent>
            {showConfigForm && (
              <div className="flex flex-col gap-3 pb-4 mb-4 border-b">
                <Input placeholder={t('common:name', 'Name')} value={configForm.name} onChange={e => setConfigForm(p => ({ ...p, name: e.target.value }))} />
                <Input placeholder={t('universe:profileId', 'Profile ID')} value={configForm.profile_id} onChange={e => setConfigForm(p => ({ ...p, profile_id: e.target.value }))} />
                <div>
                  <Label>{t('universe:assetClassFilters', 'Asset Class Filters (JSON)')}</Label>
                  <Textarea className="font-mono text-xs min-h-[60px]" value={configForm.asset_class_filters} onChange={e => setConfigForm(p => ({ ...p, asset_class_filters: e.target.value }))} />
                </div>
                <div>
                  <Label>{t('universe:dynamicTriggers', 'Dynamic Triggers (JSON)')}</Label>
                  <Textarea className="font-mono text-xs min-h-[60px]" value={configForm.dynamic_triggers} onChange={e => setConfigForm(p => ({ ...p, dynamic_triggers: e.target.value }))} />
                </div>
                <div className="flex gap-2">
                  <Button onClick={createConfig}>{t('common:create', 'Create')}</Button>
                  <Button variant="outline" onClick={() => setShowConfigForm(false)}>{t('common:cancel', 'Cancel')}</Button>
                </div>
              </div>
            )}

            {configs.length === 0 ? <p className="text-sm text-muted-foreground">{t('universe:noConfigs', 'No configs')}</p> : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('universe:table.name', 'Name')}</TableHead>
                    <TableHead>{t('universe:table.profile', 'Profile')}</TableHead>
                    <TableHead>{t('common:active', 'Active')}</TableHead>
                    <TableHead>{t('common:actions', 'Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {/* eslint-disable-next-line @typescript-eslint/no-explicit-any */}
                  {configs.map((c: any) => (
                    <TableRow key={c.ID}>
                      <TableCell>{c.Name}</TableCell>
                      <TableCell>{c.ProfileID}</TableCell>
                      <TableCell>
                        {c.IsActive ? <Badge variant="outline" className="text-trading-success border-trading-success/50">{t('common:active', 'Active')}</Badge> : '\u2014'}
                      </TableCell>
                      <TableCell>
                        {!c.IsActive && <Button variant="outline" size="sm" onClick={() => activateConfig(c.ID)}>{t('universe:activate', 'Activate')}</Button>}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
