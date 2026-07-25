import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { strategies } from '../api/client'
import { useCacheStore } from '../stores/cacheStore'
import { STRATEGY_CATALOG, type CatalogWithInstance } from '../data/strategyCatalog'
import type { Strategy } from '../types/api'
import { Card, CardContent } from '../components/ui/card'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogAction,
  AlertDialogCancel,
} from '../components/ui/alert-dialog'
import CatalogTab from './strategy-hub/CatalogTab'
import InstancesTab from './strategy-hub/InstancesTab'
import EditorTab from './strategy-hub/EditorTab'

export default function StrategyHub() {
  const { t } = useTranslation()
  const cacheStore = useCacheStore()
  const [dbList, setDbList] = useState<Strategy[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [msg, setMsg] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState('catalog')
  const [editorId, setEditorId] = useState<string | null>(null)

  useEffect(() => {
    cacheStore
      .fetchStrategies(async () => {
        const res = await strategies.list()
        return res.strategies ?? []
      })
      .then((list) => setDbList(list as Strategy[]))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const refreshDbList = async () => {
    try {
      const refreshed = await strategies.list()
      setDbList(refreshed.strategies ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Refresh failed')
    }
  }

  const dbByType: Record<string, Strategy[]> = {}
  for (const s of dbList) {
    if (!dbByType[s.type]) dbByType[s.type] = []
    dbByType[s.type].push(s)
  }

  const catalog: CatalogWithInstance[] = STRATEGY_CATALOG.map((entry) => {
    const instances = dbByType[entry.typeKey] ?? []
    return { ...entry, dbInstance: instances.length > 0 ? instances[0] : null }
  })

  const toggleEnabled = async (id: string, current: boolean) => {
    try {
      await strategies.update(id, { enabled: !current })
      setDbList((prev) => prev.map((s) => (s.id === id ? { ...s, enabled: !current } : s)))
      setMsg(t('strategies:updated', 'Updated'))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('strategies:updateFailed', 'Update failed'))
    }
  }

  const handleClone = async (id: string) => {
    try {
      const clone = await strategies.clone(id)
      setMsg(t('strategies:clonedAs', 'Cloned as "{{name}}"', { name: clone.name }))
      await refreshDbList()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('strategies:cloneFailed', 'Clone failed'))
    }
  }

  const handleDelete = (id: string) => {
    setConfirmDelete(id)
  }

  const confirmDeleteStrategy = async () => {
    if (!confirmDelete) return
    try {
      await strategies.delete(confirmDelete)
      setDbList((prev) => prev.filter((s) => s.id !== confirmDelete))
      setMsg(t('strategies:deleted', 'Deleted'))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('strategies:deleteFailed', 'Delete failed'))
    } finally {
      setConfirmDelete(null)
    }
  }

  const handleEdit = (id: string) => {
    setEditorId(id)
    setActiveTab('editor')
  }

  const handleNew = () => {
    setEditorId(null)
    setActiveTab('editor')
  }

  const handleEditorCreated = async () => {
    await refreshDbList()
    setActiveTab('instances')
  }

  if (loading)
    return (
      <Card>
        <CardContent className="p-6">
          <p className="text-sm text-muted-foreground">{t('strategies:loading', 'Loading strategies...')}</p>
        </CardContent>
      </Card>
    )

  if (error)
    return (
      <Card>
        <CardContent className="p-6">
          <p className="text-destructive">{error}</p>
        </CardContent>
      </Card>
    )

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h1 className="m-0">{t('strategies:title', 'Strategy Hub')}</h1>
      </div>

      {msg && <p className="text-sm mb-2 text-trading-success">{msg}</p>}

      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList>
          <TabsTrigger value="catalog">
            {t('strategies:catalogTab', 'Catalog')} ({catalog.length})
          </TabsTrigger>
          <TabsTrigger value="instances">
            {t('strategies:instancesTab', 'Instances')} ({dbList.length})
          </TabsTrigger>
          <TabsTrigger value="editor">
            {editorId ? t('strategyEditor:editStrategy', 'Edit') : t('strategyEditor:newStrategy', 'Editor')}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="catalog">
          <CatalogTab
            dbList={dbList}
            catalog={catalog}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onClone={handleClone}
            onToggle={toggleEnabled}
            onNew={handleNew}
          />
        </TabsContent>

        <TabsContent value="instances">
          <InstancesTab
            dbList={dbList}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onClone={handleClone}
            onToggle={toggleEnabled}
          />
        </TabsContent>

        <TabsContent value="editor">
          <EditorTab
            id={editorId}
            onCreated={handleEditorCreated}
            onBack={() => setActiveTab('catalog')}
          />
        </TabsContent>
      </Tabs>

      <AlertDialog open={!!confirmDelete} onOpenChange={(open) => !open && setConfirmDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('strategies:deleteTitle', 'Delete Strategy')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('strategies:deleteConfirm', 'Delete this strategy instance? This action cannot be undone.')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setConfirmDelete(null)}>
              {t('common:cancel', 'Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction className="bg-destructive hover:bg-destructive/90" onClick={confirmDeleteStrategy}>
              {t('common:delete', 'Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
