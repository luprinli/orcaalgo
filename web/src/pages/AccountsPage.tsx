import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm, Controller } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { accounts, brokers } from '../api/client'
import ConfirmDialog from '../components/ConfirmDialog'
import { Card, CardHeader, CardTitle, CardContent } from '../components/ui/card'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'
import { Label } from '../components/ui/label'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../components/ui/select'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table'
import type { Account, CreateAccountRequest } from '../types/api'

const accountSchema = z.object({
  name: z.string().min(1, 'Account name is required').max(64),
  broker_type: z.string().min(1),
  is_default: z.boolean(),
})

type AccountFormData = z.infer<typeof accountSchema>

export default function AccountsPage() {
  const { t } = useTranslation()
  const [list, setList] = useState<Account[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [msg, setMsg] = useState('')
  const [brokerOptions, setBrokerOptions] = useState<{ id: string; label: string }[]>([])
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; name: string } | null>(null)

  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors, isSubmitting },
  } = useForm<AccountFormData>({
    resolver: zodResolver(accountSchema),
    defaultValues: { name: '', broker_type: 'paper', is_default: false },
  })

  const fetchAll = useCallback(async () => {
    try {
      setError(null)
      const res = await accounts.list()
      setList(Array.isArray(res) ? res : [])
    } catch (err) {
      setError(err instanceof Error ? err.message : t('accounts:failedToLoad', 'Failed to load accounts'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchAll()
    brokers.list().then((res) => {
      if (res.brokers?.length) {
        setBrokerOptions(res.brokers)
      }
    }).catch(() => {
      setBrokerOptions([
        { id: 'paper', label: t('accounts:paper', 'Paper') },
        { id: 'alpaca', label: t('accounts:alpacaLive', 'Alpaca Live') },
        { id: 'ibkr', label: t('accounts:ibkr', 'IBKR') },
      ])
    })
  }, [fetchAll])

  const onCreate = async (form: AccountFormData) => {
    try {
      const data: CreateAccountRequest = {
        name: form.name,
        broker_type: form.broker_type,
        is_default: form.is_default,
      }
      await accounts.create(data)
      setMsg(t('accounts:accountCreated', 'Account "{{name}}" created', { name: form.name }))
      setShowCreate(false)
      reset()
      fetchAll()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('accounts:createFailed', 'Create failed'))
    }
  }

  const confirmDeleteAccount = async () => {
    if (!confirmDelete) return
    try {
      await accounts.delete(confirmDelete.id)
      setMsg(t('accounts:accountDeleted', 'Account "{{name}}" deleted', { name: confirmDelete.name }))
      setConfirmDelete(null)
      fetchAll()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('accounts:deleteFailed', 'Delete failed'))
      setConfirmDelete(null)
    }
  }

  const handleSetDefault = async (id: string) => {
    try {
      await accounts.setDefault(id)
      setMsg(t('accounts:defaultUpdated', 'Default account updated'))
      fetchAll()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('accounts:failedToSetDefault', 'Failed to set default'))
    }
  }

  if (loading) {
    return (
      <Card>
        <CardContent className="pt-6 text-muted-foreground">
          {t('accounts:loading', 'Loading accounts...')}
        </CardContent>
      </Card>
    )
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-bold mb-0">{t('accounts:title', 'Accounts')}</h1>
        <Button onClick={() => setShowCreate(true)}>
          {t('accounts:newAccount', '+ New Account')}
        </Button>
      </div>

      {error && (
        <Card className="mb-4 border-destructive border-l-4">
          <CardContent className="text-destructive text-sm pt-4">{error}</CardContent>
        </Card>
      )}

      {showCreate && (
        <Card className="mb-4 max-w-[400px]">
          <CardHeader><CardTitle>{t('accounts:createAccount', 'Create Account')}</CardTitle></CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit(onCreate)} className="flex flex-col gap-2.5">
              <div>
                <Input placeholder={t('accounts:accountName', 'Account name')} {...register('name')} />
                {errors.name && <p className="text-destructive text-[11px] mt-1 mb-0">{t('accounts:validation:nameRequired', errors.name.message || 'Account name is required')}</p>}
              </div>
              <Controller
                name="broker_type"
                control={control}
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      {(brokerOptions.length > 0 ? brokerOptions : [{ id: 'paper', label: 'Paper' }]).map((b) => (
                        <SelectItem key={b.id} value={b.id}>{b.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
              <label className="flex items-center gap-2">
                <input type="checkbox" {...register('is_default')} />
                {t('accounts:setAsDefault', 'Set as default')}
              </label>
              <div className="flex gap-2">
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting ? t('accounts:creating', 'Creating...') : t('accounts:create', 'Create')}
                </Button>
                <Button variant="outline" type="button" onClick={() => setShowCreate(false)}>
                  {t('accounts:cancel', 'Cancel')}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {msg && (
        <p className="text-muted-foreground text-[13px] mb-4">{msg}</p>
      )}

      {list.length === 0 ? (
        <Card>
          <CardContent className="pt-6 text-muted-foreground">
            {t('accounts:noAccounts', 'No accounts configured. Create one to start trading.')}
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="pt-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('accounts:table:name', 'Name')}</TableHead>
                  <TableHead>{t('accounts:table:broker', 'Broker')}</TableHead>
                  <TableHead>{t('accounts:table:default', 'Default')}</TableHead>
                  <TableHead>{t('accounts:table:balance', 'Balance')}</TableHead>
                  <TableHead>{t('accounts:table:equity', 'Equity')}</TableHead>
                  <TableHead>{t('accounts:table:dailyPnl', 'Daily P&L')}</TableHead>
                  <TableHead>{t('accounts:table:buyingPower', 'Buying Power')}</TableHead>
                  <TableHead>{t('accounts:table:status', 'Status')}</TableHead>
                  <TableHead>{t('accounts:table:actions', 'Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {list.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell className="font-semibold">{a.label || a.id}</TableCell>
                    <TableCell>{a.broker_type}</TableCell>
                    <TableCell>{a.is_default ? <Badge>{t('accounts:defaultAccount', 'Default')}</Badge> : '—'}</TableCell>
                    <TableCell>${a.balance?.toFixed(2) ?? '--'}</TableCell>
                    <TableCell>${a.equity?.toFixed(2) ?? '--'}</TableCell>
                    <TableCell className={(a.daily_pnl_pct ?? 0) >= 0 ? 'text-trading-success' : 'text-trading-danger'}>
                      {a.daily_pnl_pct != null ? `${a.daily_pnl_pct.toFixed(2)}%` : '--'}
                    </TableCell>
                    <TableCell>${a.buying_power?.toFixed(2) ?? '--'}</TableCell>
                    <TableCell>
                      <Badge variant={a.halted ? 'destructive' : 'default'}>
                        {a.halted ? t('common:halted', 'HALTED') : t('common:active', 'Active')}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {!a.is_default && (
                          <Button variant="outline" size="sm" onClick={() => handleSetDefault(a.id)}>
                            {t('accounts:setDefault', 'Set Default')}
                          </Button>
                        )}
                        <Button variant="outline" size="sm" className="text-trading-danger" onClick={() => setConfirmDelete({ id: a.id, name: a.label || a.id })}>
                          {t('accounts:delete', 'Delete')}
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {confirmDelete && (
        <ConfirmDialog
          title={t('accounts:deleteTitle', 'Delete Account')}
          message={t('accounts:deleteConfirm', 'Delete account "{{name}}"? This action cannot be undone.', { name: confirmDelete.name })}
          confirmLabel={t('accounts:delete', 'Delete')}
          danger
          onConfirm={confirmDeleteAccount}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  )
}
