import { useTranslation } from 'react-i18next'
import type { Strategy } from '../../types/api'
import type { CatalogWithInstance } from '../../data/strategyCatalog'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'

interface CatalogTabProps {
  dbList: Strategy[]
  catalog: CatalogWithInstance[]
  onEdit: (id: string) => void
  onDelete: (id: string) => void
  onClone: (id: string) => void
  onToggle: (id: string, current: boolean) => void
  onNew: () => void
}

export default function CatalogTab({
  catalog,
  onEdit,
  onDelete,
  onClone,
  onToggle,
  onNew,
}: CatalogTabProps) {
  const { t } = useTranslation()

  return (
    <Card>
      <div className="flex items-center justify-between p-4 pb-0">
        <h2 className="text-sm font-medium text-foreground">
          {t('strategies:catalogTab', 'Strategy Catalog')}
        </h2>
        <Button size="sm" onClick={onNew}>
          {t('strategies:newStrategy', '+ New Strategy')}
        </Button>
      </div>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('strategies:table:strategy', 'Strategy')}</TableHead>
              <TableHead>{t('strategies:table:typeKey', 'Type Key')}</TableHead>
              <TableHead>{t('strategies:table:engine', 'Engine')}</TableHead>
              <TableHead>{t('strategies:table:gkr', 'GKR')}</TableHead>
              <TableHead>{t('strategies:table:dbInstance', 'DB Instance')}</TableHead>
              <TableHead>{t('strategies:table:parameters', 'Parameters')}</TableHead>
              <TableHead>{t('strategies:table:actions', 'Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {catalog.map((c) => (
              <TableRow key={c.typeKey} className={!c.inEngine && !c.dbInstance ? 'opacity-50' : ''}>
                <TableCell className="font-bold">{c.displayName}</TableCell>
                <TableCell className="font-mono text-xs">{c.typeKey}</TableCell>
                <TableCell>
                  {c.inEngine ? (
                    <Badge variant="success">{t('strategies:registered', 'Registered')}</Badge>
                  ) : (
                    <Badge variant="destructive">{'\u2014'}</Badge>
                  )}
                </TableCell>
                <TableCell>
                  {c.hasGkrFile ? (
                    <Badge variant="success">{t('strategies:yaml', 'YAML')}</Badge>
                  ) : (
                    <span className="text-sm text-muted-foreground">{'\u2014'}</span>
                  )}
                </TableCell>
                <TableCell>
                  {c.dbInstance ? (
                    <span>
                      <Badge variant={c.dbInstance.enabled ? 'success' : 'destructive'}>
                        {c.dbInstance.enabled ? t('common:active', 'Active') : t('common:disabled', 'Disabled')}
                      </Badge>
                      <span className="text-sm text-muted-foreground ml-1">{c.dbInstance.name}</span>
                    </span>
                  ) : (
                    <span className="text-sm text-muted-foreground">{t('common:none', 'None')}</span>
                  )}
                </TableCell>
                <TableCell className="text-xs text-muted-foreground max-w-[180px]">{c.paramDefs}</TableCell>
                <TableCell>
                  <div className="flex gap-1">
                    {c.dbInstance ? (
                      <>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => onEdit(c.dbInstance!.id)}
                        >
                          {t('common:edit', 'Edit')}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => onToggle(c.dbInstance!.id, c.dbInstance!.enabled)}
                        >
                          {c.dbInstance.enabled ? t('common:disable', 'Disable') : t('common:enable', 'Enable')}
                        </Button>
                        <Button variant="outline" size="sm" onClick={() => onClone(c.dbInstance!.id)}>
                          {t('common:clone', 'Clone')}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          className="text-destructive border-destructive/50 hover:bg-destructive/10"
                          onClick={() => onDelete(c.dbInstance!.id)}
                        >
                          {t('common:del', 'Del')}
                        </Button>
                      </>
                    ) : c.inEngine ? (
                      <Button variant="outline" size="sm" onClick={onNew}>
                        {t('common:create', 'Create')}
                      </Button>
                    ) : null}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </Card>
  )
}
