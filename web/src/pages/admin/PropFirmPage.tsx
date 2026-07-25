import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import i18next from 'i18next'
import { propfirm } from '../../api/client'
import type { PropFirmProfile, PropFirmState } from '../../types/api'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Label } from '../../components/ui/label'
import { Badge } from '../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogFooter, AlertDialogTitle, AlertDialogDescription, AlertDialogAction, AlertDialogCancel } from '../../components/ui/alert-dialog'
import MetricCard from '../../components/MetricCard'

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

  if (loading) return <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">Loading...</p></CardContent></Card>

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="m-0">{t('propfirm:title', 'Prop Firm')}</h1>
        <Button onClick={() => { setShowForm(true); reset() }}>{t('propfirm:newProfile', '+ New Profile')}</Button>
      </div>

      {msg && <p className={`text-sm mb-2 ${msg.includes('fail') ? 'text-destructive' : 'text-emerald-400'}`}>{msg}</p>}

      {status && (
        <Card className="mb-4">
          <CardHeader><CardTitle>{t('propfirm:status', 'Status')}</CardTitle></CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <MetricCard label={t('propfirm:phase', 'Phase')} value={status.current_phase} format="number" />
              <MetricCard label={t('propfirm:dailyPnl', 'Daily P&L')} value={status.daily_pnl_pct ?? 0} format="percent_raw" color="auto" />
              <MetricCard label={t('propfirm:cumulative', 'Cumulative')} value={status.cumulative_pnl ?? 0} format="currency" />
              <MetricCard label={t('propfirm:tradingDays', 'Trading Days')} value={status.trading_days} format="number" />
              <MetricCard
                label={t('propfirm:targetMet', 'Target Met')}
                value={status.phase_target_met ? t('common:yes', 'Yes') : t('common:no', 'No')}
                color={status.phase_target_met ? 'positive' : 'default'}
              />
              <MetricCard
                label={t('propfirm:violation', 'Violation')}
                value={status.violated ? status.violation_reason : t('common:none', 'None')}
                color={status.violated ? 'negative' : 'positive'}
              />
            </div>
          </CardContent>
        </Card>
      )}

      {showForm && (
        <Card className="mb-4 max-w-[500px]">
          <CardHeader><CardTitle>{t('propfirm:newProfileTitle', 'New Profile')}</CardTitle></CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit(handleCreate)} className="flex flex-col gap-3">
              <div>
                <Input placeholder={t('propfirm:profileId', 'Profile ID')} {...register('id')} />
                {errors.id && <p className="text-destructive text-xs mt-1">{errors.id.message}</p>}
              </div>
              <div>
                <Input placeholder={t('propfirm:name', 'Name')} {...register('name')} />
                {errors.name && <p className="text-destructive text-xs mt-1">{errors.name.message}</p>}
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>{t('propfirm:maxDailyLossPct', 'Max Daily Loss %')}</Label>
                  <Input type="number" step="0.1" {...register('max_daily_loss_pct')} />
                  {errors.max_daily_loss_pct && <p className="text-destructive text-xs mt-1">{errors.max_daily_loss_pct.message}</p>}
                </div>
                <div>
                  <Label>{t('propfirm:maxDrawdownPct', 'Max Drawdown %')}</Label>
                  <Input type="number" step="0.1" {...register('max_drawdown_pct')} />
                  {errors.max_drawdown_pct && <p className="text-destructive text-xs mt-1">{errors.max_drawdown_pct.message}</p>}
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>{t('propfirm:phase1TargetPct', 'Phase 1 Target %')}</Label>
                  <Input type="number" step="0.1" {...register('profit_target_pct_phase1')} />
                  {errors.profit_target_pct_phase1 && <p className="text-destructive text-xs mt-1">{errors.profit_target_pct_phase1.message}</p>}
                </div>
                <div>
                  <Label>{t('propfirm:phase2TargetPct', 'Phase 2 Target %')}</Label>
                  <Input type="number" step="0.1" {...register('profit_target_pct_phase2')} />
                  {errors.profit_target_pct_phase2 && <p className="text-destructive text-xs mt-1">{errors.profit_target_pct_phase2.message}</p>}
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label>{t('propfirm:maxPositions', 'Max Positions')}</Label>
                  <Input type="number" {...register('max_open_positions')} />
                  {errors.max_open_positions && <p className="text-destructive text-xs mt-1">{errors.max_open_positions.message}</p>}
                </div>
                <div>
                  <Label>{t('propfirm:minTradingDays', 'Min Trading Days')}</Label>
                  <Input type="number" {...register('min_trading_days')} />
                  {errors.min_trading_days && <p className="text-destructive text-xs mt-1">{errors.min_trading_days.message}</p>}
                </div>
              </div>
              <div className="flex gap-2">
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting ? t('propfirm:creating', 'Creating...') : t('propfirm:create', 'Create')}
                </Button>
                <Button variant="outline" type="button" onClick={() => setShowForm(false)}>
                  {t('propfirm:cancel', 'Cancel')}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><CardTitle>{t('propfirm:profiles', 'Profiles ({{n}})', { n: profiles.length })}</CardTitle></CardHeader>
        <CardContent>
          {profiles.length === 0 ? <p className="text-sm text-muted-foreground">{t('propfirm:noProfiles', 'No profiles')}</p> : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('propfirm:table.id', 'ID')}</TableHead>
                  <TableHead>{t('propfirm:table.name', 'Name')}</TableHead>
                  <TableHead>{t('propfirm:table.dailyLoss', 'Daily Loss')}</TableHead>
                  <TableHead>{t('propfirm:table.maxDd', 'Max DD')}</TableHead>
                  <TableHead>{t('propfirm:table.phase1', 'Phase 1')}</TableHead>
                  <TableHead>{t('propfirm:table.phase2', 'Phase 2')}</TableHead>
                  <TableHead>{t('propfirm:table.active', 'Active')}</TableHead>
                  <TableHead>{t('propfirm:table.actions', 'Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {profiles.map(p => (
                  <TableRow key={p.id}>
                    <TableCell>{p.id}</TableCell>
                    <TableCell>{p.name}</TableCell>
                    <TableCell>{p.max_daily_loss_pct}%</TableCell>
                    <TableCell>{p.max_drawdown_pct}%</TableCell>
                    <TableCell>{p.profit_target_pct_phase1}%</TableCell>
                    <TableCell>{p.profit_target_pct_phase2}%</TableCell>
                    <TableCell>
                      {activeId === p.id ? <Badge variant="outline" className="text-trading-success border-trading-success/50">{t('common:active', 'Active')}</Badge> : '\u2014'}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {activeId !== p.id && <Button variant="outline" size="sm" onClick={() => handleSetActive(p.id)}>{t('propfirm:activate', 'Activate')}</Button>}
                        <Button variant="outline" size="sm" className="text-destructive border-destructive/50 hover:bg-destructive/10" onClick={() => handleDelete(p.id)}>{t('propfirm:delete', 'Delete')}</Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={!!confirmDelete} onOpenChange={(open) => !open && setConfirmDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('propfirm:deleteTitle', 'Delete Profile')}</AlertDialogTitle>
            <AlertDialogDescription>{t('propfirm:deleteConfirm', 'Delete this prop firm profile? This action cannot be undone.')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setConfirmDelete(null)}>{t('common:cancel', 'Cancel')}</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive hover:bg-destructive/90" onClick={confirmDeleteProfile}>{t('propfirm:delete', 'Delete')}</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
