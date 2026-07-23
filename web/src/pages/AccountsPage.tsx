import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { accounts, brokers } from '../api/client'
import ConfirmDialog from '../components/ConfirmDialog'
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
      <div className="card">
        <p className="text-muted">{t('accounts:loading', 'Loading accounts...')}</p>
      </div>
    )
  }

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('accounts:title', 'Accounts')}</h1>
        <button className="btn btn-primary" onClick={() => setShowCreate(true)}>
          {t('accounts:newAccount', '+ New Account')}
        </button>
      </div>

      {error && (
        <div className="card mb-4" style={{ background: 'rgba(218,54,51,.1)', border: '1px solid var(--danger)' }}>
          <span style={{ color: 'var(--danger)' }}>{error}</span>
        </div>
      )}

      {showCreate && (
        <div className="card mb-4" style={{ maxWidth: 400 }}>
          <h2>{t('accounts:createAccount', 'Create Account')}</h2>
          <form onSubmit={handleSubmit(onCreate)} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div>
              <input className="input" placeholder={t('accounts:accountName', 'Account name')} {...register('name')} />
              {errors.name && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{t('accounts:validation:nameRequired', errors.name.message || 'Account name is required')}</p>}
            </div>
            <select className="input" {...register('broker_type')}>
              {(brokerOptions.length > 0 ? brokerOptions : [{ id: 'paper', label: 'Paper' }]).map((b) => (
                <option key={b.id} value={b.id}>{b.label}</option>
              ))}
            </select>
            <label className="flex gap-2" style={{ alignItems: 'center' }}>
              <input type="checkbox" {...register('is_default')} />
              {t('accounts:setAsDefault', 'Set as default')}
            </label>
            <div className="flex gap-2">
              <button className="btn btn-primary" type="submit" disabled={isSubmitting}>
                {isSubmitting ? t('accounts:creating', 'Creating...') : t('accounts:create', 'Create')}
              </button>
              <button className="btn btn-outline" type="button" onClick={() => setShowCreate(false)}>
                {t('accounts:cancel', 'Cancel')}
              </button>
            </div>
          </form>
        </div>
      )}

      {msg && (
        <p className="text-muted mb-4" style={{ fontSize: 13, margin: '0 0 12px' }}>{msg}</p>
      )}

      {list.length === 0 ? (
        <div className="card">
          <p className="text-muted">{t('accounts:noAccounts', 'No accounts configured. Create one to start trading.')}</p>
        </div>
      ) : (
        <div className="card">
          <div style={{ overflowX: 'auto' }}>
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t('accounts:table:name', 'Name')}</th>
                  <th>{t('accounts:table:broker', 'Broker')}</th>
                  <th>{t('accounts:table:default', 'Default')}</th>
                  <th>{t('accounts:table:balance', 'Balance')}</th>
                  <th>{t('accounts:table:equity', 'Equity')}</th>
                  <th>{t('accounts:table:dailyPnl', 'Daily P&L')}</th>
                  <th>{t('accounts:table:buyingPower', 'Buying Power')}</th>
                  <th>{t('accounts:table:status', 'Status')}</th>
                  <th>{t('accounts:table:actions', 'Actions')}</th>
                </tr>
              </thead>
              <tbody>
                {list.map((a) => (
                  <tr key={a.id}>
                    <td><strong>{a.label || a.id}</strong></td>
                    <td>{a.broker_type}</td>
                    <td>{a.is_default ? <span className="badge badge-ok">{t('accounts:defaultAccount', 'Default')}</span> : '—'}</td>
                    <td>${a.balance?.toFixed(2) ?? '--'}</td>
                    <td>${a.equity?.toFixed(2) ?? '--'}</td>
                    <td style={{ color: (a.daily_pnl_pct ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                      {a.daily_pnl_pct != null ? `${a.daily_pnl_pct.toFixed(2)}%` : '--'}
                    </td>
                    <td>${a.buying_power?.toFixed(2) ?? '--'}</td>
                    <td>
                      <span className={`badge ${a.halted ? 'badge-err' : 'badge-ok'}`}>
                        {a.halted ? t('common:halted', 'HALTED') : t('common:active', 'Active')}
                      </span>
                    </td>
                    <td>
                      <div className="flex gap-1">
                        {!a.is_default && (
                          <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => handleSetDefault(a.id)}>
                            {t('accounts:setDefault', 'Set Default')}
                          </button>
                        )}
                        <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11, color: 'var(--danger)' }} onClick={() => setConfirmDelete({ id: a.id, name: a.label || a.id })}>
                          {t('accounts:delete', 'Delete')}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
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
