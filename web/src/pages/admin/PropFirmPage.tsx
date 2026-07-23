import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import i18next from 'i18next'
import { propfirm } from '../../api/client'
import ConfirmDialog from '../../components/ConfirmDialog'
import type { PropFirmProfile, PropFirmState } from '../../types/api'

const profileSchema = z.object({
  id: z.string().min(1, i18next.t('propfirm:validation.profileIdRequired', 'Profile ID is required')),
  name: z.string().min(1, i18next.t('propfirm:validation.profileNameRequired', 'Profile name is required')),
  max_daily_loss_pct: z.number({ message: i18next.t('propfirm:validation.required', 'Required') }).min(0).max(100),
  max_drawdown_pct: z.number({ message: i18next.t('propfirm:validation.required', 'Required') }).min(0).max(100),
  profit_target_pct_phase1: z.number({ message: i18next.t('propfirm:validation.required', 'Required') }).min(0).max(1000),
  profit_target_pct_phase2: z.number({ message: i18next.t('propfirm:validation.required', 'Required') }).min(0).max(1000),
  max_open_positions: z.number({ message: i18next.t('propfirm:validation.required', 'Required') }).int().min(1),
  min_trading_days: z.number({ message: i18next.t('propfirm:validation.required', 'Required') }).int().min(1),
})

type ProfileFormData = z.infer<typeof profileSchema>

export default function PropFirmPage() {
  const { t } = useTranslation()
  const [profiles, setProfiles] = useState<PropFirmProfile[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [status, setStatus] = useState<PropFirmState | null>(null)
  const [loading, setLoading] = useState(true)
  const [msg, setMsg] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      id: '',
      name: '',
      max_daily_loss_pct: 5,
      max_drawdown_pct: 10,
      profit_target_pct_phase1: 10,
      profit_target_pct_phase2: 5,
      max_open_positions: 5,
      min_trading_days: 4,
    },
  })

  const fetchAll = useCallback(async () => {
    setLoading(true)
    try {
      const [profilesRes, activeRes, statusRes] = await Promise.all([
        propfirm.profiles.list(), propfirm.active.get(), propfirm.status(),
      ])
      setProfiles(Array.isArray(profilesRes) ? profilesRes : [])
      setActiveId(activeRes?.id ?? null)
      setStatus(statusRes)
    } catch {
      setMsg(t('propfirm:failedToLoad', 'Failed to load prop firm data'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])

  const handleCreate = async (form: ProfileFormData) => {
    try {
      const profile: PropFirmProfile = {
        id: form.id,
        name: form.name,
        max_daily_loss_pct: form.max_daily_loss_pct,
        max_drawdown_pct: form.max_drawdown_pct,
        drawdown_type: 'trailing',
        max_position_pct: 100,
        max_open_positions: form.max_open_positions,
        max_trades_per_day: 100,
        consistency_enabled: false,
        profit_target_pct_phase1: form.profit_target_pct_phase1,
        profit_target_pct_phase2: form.profit_target_pct_phase2,
        min_trading_days: form.min_trading_days,
      }
      await propfirm.profiles.create(profile)
      setMsg(t('propfirm:profileCreated', 'Profile created'))
      setShowForm(false)
      reset()
      fetchAll()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('propfirm:createFailed', 'Create failed'))
    }
  }

  const handleDelete = (id: string) => {
    setConfirmDelete(id)
  }

  const confirmDeleteProfile = async () => {
    if (!confirmDelete) return
    try {
      await propfirm.profiles.delete(confirmDelete)
      setMsg(t('propfirm:profileDeleted', 'Profile deleted'))
      setConfirmDelete(null)
      fetchAll()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('propfirm:deleteFailed', 'Delete failed'))
      setConfirmDelete(null)
    }
  }

  const handleSetActive = async (id: string) => {
    try {
      await propfirm.active.set(id)
      setMsg(t('propfirm:activeProfileUpdated', 'Active profile updated'))
      fetchAll()
    } catch (err) {
      setMsg(err instanceof Error ? err.message : t('propfirm:failed', 'Failed'))
    }
  }

  if (loading) return <div className="card"><p className="text-muted">Loading...</p></div>

  return (
    <div>
      <div className="flex-between mb-4">
        <h1 style={{ margin: 0 }}>{t('propfirm:title', 'Prop Firm')}</h1>
        <button className="btn btn-primary" onClick={() => { setShowForm(true); reset() }}>{t('propfirm:newProfile', '+ New Profile')}</button>
      </div>

      {msg && <p className="text-muted mb-2" style={{ color: msg.includes('fail') ? 'var(--danger)' : 'var(--success)' }}>{msg}</p>}

      {status && (
        <div className="card mb-4">
          <h2>{t('propfirm:status', 'Status')}</h2>
          <div className="metric-grid">
            <div className="metric-card"><div className="metric-label">{t('propfirm:phase', 'Phase')}</div><div className="metric-value">{status.current_phase}</div></div>
            <div className="metric-card"><div className="metric-label">{t('propfirm:dailyPnl', 'Daily P&L')}</div><div className="metric-value" style={{ color: (status.daily_pnl_pct ?? 0) >= 0 ? 'var(--success)' : 'var(--danger)' }}>{(status.daily_pnl_pct ?? 0).toFixed(2)}%</div></div>
            <div className="metric-card"><div className="metric-label">{t('propfirm:cumulative', 'Cumulative')}</div><div className="metric-value">${(status.cumulative_pnl ?? 0).toFixed(2)}</div></div>
            <div className="metric-card"><div className="metric-label">{t('propfirm:tradingDays', 'Trading Days')}</div><div className="metric-value">{status.trading_days}</div></div>
            <div className="metric-card"><div className="metric-label">{t('propfirm:targetMet', 'Target Met')}</div><div className="metric-value" style={{ color: status.phase_target_met ? 'var(--success)' : 'var(--warn)' }}>{status.phase_target_met ? t('common:yes', 'Yes') : t('common:no', 'No')}</div></div>
            <div className="metric-card"><div className="metric-label">{t('propfirm:violation', 'Violation')}</div><div className="metric-value" style={{ color: status.violated ? 'var(--danger)' : 'var(--success)' }}>{status.violated ? status.violation_reason : t('common:none', 'None')}</div></div>
          </div>
        </div>
      )}

      {showForm && (
        <div className="card mb-4" style={{ maxWidth: 500 }}>
          <h2>{t('propfirm:newProfileTitle', 'New Profile')}</h2>
          <form onSubmit={handleSubmit(handleCreate)} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div>
              <input className="input" placeholder={t('propfirm:profileId', 'Profile ID')} {...register('id')} />
              {errors.id && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.id.message}</p>}
            </div>
            <div>
              <input className="input" placeholder={t('propfirm:name', 'Name')} {...register('name')} />
              {errors.name && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.name.message}</p>}
            </div>
            <div className="grid-2">
              <div>
                <label className="text-muted">{t('propfirm:maxDailyLossPct', 'Max Daily Loss %')}</label>
                <input className="input" type="number" step="0.1" {...register('max_daily_loss_pct')} />
                {errors.max_daily_loss_pct && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.max_daily_loss_pct.message}</p>}
              </div>
              <div>
                <label className="text-muted">{t('propfirm:maxDrawdownPct', 'Max Drawdown %')}</label>
                <input className="input" type="number" step="0.1" {...register('max_drawdown_pct')} />
                {errors.max_drawdown_pct && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.max_drawdown_pct.message}</p>}
              </div>
            </div>
            <div className="grid-2">
              <div>
                <label className="text-muted">{t('propfirm:phase1TargetPct', 'Phase 1 Target %')}</label>
                <input className="input" type="number" step="0.1" {...register('profit_target_pct_phase1')} />
                {errors.profit_target_pct_phase1 && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.profit_target_pct_phase1.message}</p>}
              </div>
              <div>
                <label className="text-muted">{t('propfirm:phase2TargetPct', 'Phase 2 Target %')}</label>
                <input className="input" type="number" step="0.1" {...register('profit_target_pct_phase2')} />
                {errors.profit_target_pct_phase2 && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.profit_target_pct_phase2.message}</p>}
              </div>
            </div>
            <div className="grid-2">
              <div>
                <label className="text-muted">{t('propfirm:maxPositions', 'Max Positions')}</label>
                <input className="input" type="number" {...register('max_open_positions')} />
                {errors.max_open_positions && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.max_open_positions.message}</p>}
              </div>
              <div>
                <label className="text-muted">{t('propfirm:minTradingDays', 'Min Trading Days')}</label>
                <input className="input" type="number" {...register('min_trading_days')} />
                {errors.min_trading_days && <p style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>{errors.min_trading_days.message}</p>}
              </div>
            </div>
            <div className="flex gap-2">
              <button className="btn btn-primary" type="submit" disabled={isSubmitting}>
                {isSubmitting ? t('propfirm:creating', 'Creating...') : t('propfirm:create', 'Create')}
              </button>
              <button className="btn btn-outline" type="button" onClick={() => setShowForm(false)}>
                {t('propfirm:cancel', 'Cancel')}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="card">
        <h2>{t('propfirm:profiles', 'Profiles ({{n}})', { n: profiles.length })}</h2>
        {profiles.length === 0 ? <p className="text-muted">{t('propfirm:noProfiles', 'No profiles')}</p> : (
          <table className="data-table">
            <thead><tr><th>{t('propfirm:table.id', 'ID')}</th><th>{t('propfirm:table.name', 'Name')}</th><th>{t('propfirm:table.dailyLoss', 'Daily Loss')}</th><th>{t('propfirm:table.maxDd', 'Max DD')}</th><th>{t('propfirm:table.phase1', 'Phase 1')}</th><th>{t('propfirm:table.phase2', 'Phase 2')}</th><th>{t('propfirm:table.active', 'Active')}</th><th>{t('propfirm:table.actions', 'Actions')}</th></tr></thead>
            <tbody>
              {profiles.map(p => (
                <tr key={p.id}>
                  <td>{p.id}</td><td>{p.name}</td>
                  <td>{p.max_daily_loss_pct}%</td><td>{p.max_drawdown_pct}%</td>
                  <td>{p.profit_target_pct_phase1}%</td><td>{p.profit_target_pct_phase2}%</td>
                  <td>{activeId === p.id ? <span className="badge badge-ok">{t('common:active', 'Active')}</span> : '\u2014'}</td>
                  <td>
                    <div className="flex gap-1">
                      {activeId !== p.id && <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11 }} onClick={() => handleSetActive(p.id)}>{t('propfirm:activate', 'Activate')}</button>}
                      <button className="btn btn-outline" style={{ padding: '2px 8px', fontSize: 11, color: 'var(--danger)' }} onClick={() => handleDelete(p.id)}>{t('propfirm:delete', 'Delete')}</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {confirmDelete && (
        <ConfirmDialog
          title={t('propfirm:deleteTitle', 'Delete Profile')}
          message={t('propfirm:deleteConfirm', 'Delete this prop firm profile? This action cannot be undone.')}
          confirmLabel={t('propfirm:delete', 'Delete')}
          danger
          onConfirm={confirmDeleteProfile}
          onCancel={() => setConfirmDelete(null)}
        />
      )}
    </div>
  )
}
