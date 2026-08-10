import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { strategies, backtests } from '../api/client'
import { useCacheStore } from '../stores/cacheStore'
import { STRATEGY_DISPLAY } from '../data/constants'
import type { Strategy, BacktestHistoryEntry, EquityPoint } from '../types/api'
import { Card, CardContent } from '../components/ui/card'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import { Badge } from '../components/ui/badge'
import { Button } from '../components/ui/button'
import {
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogFooter,
  AlertDialogTitle, AlertDialogDescription,
  AlertDialogAction, AlertDialogCancel,
} from '../components/ui/alert-dialog'
import CatalogTab from './strategy-hub/CatalogTab'
import InstancesTab from './strategy-hub/InstancesTab'
import EditorTab from './strategy-hub/EditorTab'
import StatusTab from './strategy-hub/StatusTab'

export default function StrategyHub() {
  const { t } = useTranslation()
  const cacheStore = useCacheStore()
  const [dbList, setDbList] = useState<Strategy[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState('catalog')
  const [editorId, setEditorId] = useState<string | null>(null)

  const fetchDb = () => {
    cacheStore
      .fetchStrategies(async () => {
        const res = await strategies.list()
        return res.strategies ?? []
      })
      .then((list) => setDbList(list as Strategy[]))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }
  useEffect(fetchDb, [])

  const handleToggle = async (id: string, current: boolean) => {
    await strategies.update(id, { enabled: !current })
    fetchDb()
  }

  const handleClone = async (id: string) => {
    await strategies.clone(id)
    fetchDb()
  }

  const handleDelete = async (id: string) => {
    await strategies.delete(id)
    setConfirmDelete(null)
    fetchDb()
  }

  const handleEdit = (id: string) => {
    setEditorId(id)
    setActiveTab('editor')
  }

  const totalActive = dbList.filter((s) => s.enabled).length
  const strategyTypes = useMemo(() => [...new Set(dbList.map((s) => s.type))], [dbList])

  if (editorId) {
    return <EditorTab id={editorId} onCreated={fetchDb} onBack={() => { setEditorId(null); setActiveTab('catalog') }} />
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg m-0">{t('strategies:title', 'Strategy Hub')}</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            {t('strategies:subtitle', 'Manage and monitor strategy instances with performance metrics')}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-4 gap-3">
        <Card className="bg-gradient-to-br from-blue-500/10 to-transparent">
          <CardContent className="p-3">
            <div className="text-2xl font-bold tabular-nums">{strategyTypes.length}</div>
            <div className="text-[11px] text-muted-foreground">Strategy Types</div>
          </CardContent>
        </Card>
        <Card className="bg-gradient-to-br from-green-500/10 to-transparent">
          <CardContent className="p-3">
            <div className="text-2xl font-bold tabular-nums text-green-600">{dbList.length}</div>
            <div className="text-[11px] text-muted-foreground">Instances</div>
          </CardContent>
        </Card>
        <Card className="bg-gradient-to-br from-purple-500/10 to-transparent">
          <CardContent className="p-3">
            <div className="text-2xl font-bold tabular-nums text-purple-600">{totalActive}</div>
            <div className="text-[11px] text-muted-foreground">Active Instances</div>
          </CardContent>
        </Card>
        <Card className="bg-gradient-to-br from-amber-500/10 to-transparent">
          <CardContent className="p-3">
            <div className="text-2xl font-bold tabular-nums text-amber-600">{dbList.length - totalActive}</div>
            <div className="text-[11px] text-muted-foreground">Inactive</div>
          </CardContent>
        </Card>
      </div>

      {error && (
        <Card className="border-destructive/50 bg-destructive/5">
          <CardContent className="p-3 text-xs text-destructive">{error}</CardContent>
        </Card>
      )}

      <Tabs value={activeTab} onValueChange={(v) => { if (v === 'editor') setEditorId(null); setActiveTab(v) }}>
        <TabsList className="w-full justify-start gap-1 h-8">
          <TabsTrigger value="catalog" className="text-xs h-6 data-[state=active]:bg-card">
            Catalog ({strategyTypes.length})
          </TabsTrigger>
          <TabsTrigger value="instances" className="text-xs h-6 data-[state=active]:bg-card">
            Instances ({dbList.length})
          </TabsTrigger>
          <TabsTrigger value="editor" className="text-xs h-6 data-[state=active]:bg-card">
            Editor
          </TabsTrigger>
          <TabsTrigger value="status" className="text-xs h-6 data-[state=active]:bg-card">
            Status
          </TabsTrigger>
        </TabsList>
        <TabsContent value="catalog" className="mt-3">
          <CatalogTab
            dbList={dbList}
            onEdit={handleEdit}
            onDelete={(id) => setConfirmDelete(id)}
            onClone={handleClone}
            onToggle={handleToggle}
            onNew={() => { setEditorId(null); setActiveTab('editor') }}
          />
        </TabsContent>
        <TabsContent value="instances" className="mt-3">
          <InstancesTab
            dbList={dbList}
            loading={loading}
            onEdit={handleEdit}
            onDelete={(id) => setConfirmDelete(id)}
            onClone={handleClone}
            onToggle={handleToggle}
          />
        </TabsContent>
        <TabsContent value="editor" className="mt-3">
          <EditorTab id={null} onCreated={fetchDb} onBack={() => setActiveTab('catalog')} />
        </TabsContent>
        <TabsContent value="status" className="mt-3">
          <StatusTab />
        </TabsContent>
      </Tabs>

      <AlertDialog open={confirmDelete !== null} onOpenChange={(v) => { if (!v) setConfirmDelete(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Strategy Instance?</AlertDialogTitle>
            <AlertDialogDescription>
              This permanently removes the instance. Active deployments will be halted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => confirmDelete && handleDelete(confirmDelete)} className="bg-destructive hover:bg-destructive/90">
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
