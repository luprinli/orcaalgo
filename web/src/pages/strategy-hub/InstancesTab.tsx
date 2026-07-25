import { useTranslation } from 'react-i18next'
import type { Strategy } from '../../types/api'
import { Button } from '../../components/ui/button'
import { Card, CardContent } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'

interface InstancesTabProps {
  dbList: Strategy[]
  onEdit: (id: string) => void
  onDelete: (id: string) => void
  onClone: (id: string) => void
  onToggle: (id: string, current: boolean) => void
}

export default function InstancesTab({
  dbList,
  onEdit,
  onDelete,
  onClone,
  onToggle,
}: InstancesTabProps) {
  const { t } = useTranslation()

  if (dbList.length === 0) {
    return (
      <Card>
        <CardContent className="p-6">
          <p className="text-sm text-muted-foreground">
            {t(
              'strategies:noInstances',
              'No strategy instances. Switch to Catalog view and click "Create" on a strategy type.',
            )}
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('strategies:table:name', 'Name')}</TableHead>
              <TableHead>{t('strategies:table:type', 'Type')}</TableHead>
              <TableHead>{t('strategies:table:parameters', 'Parameters')}</TableHead>
              <TableHead>{t('strategies:table:enabled', 'Enabled')}</TableHead>
              <TableHead>{t('strategies:table:created', 'Created')}</TableHead>
              <TableHead>{t('strategies:table:actions', 'Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {dbList.map((s) => (
              <TableRow key={s.id}>
                <TableCell
                  className="cursor-pointer font-bold"
                  onClick={() => onEdit(s.id)}
                >
                  {s.name}
                </TableCell>
                <TableCell className="font-mono text-xs">{s.type}</TableCell>
                <TableCell className="text-xs font-mono max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">
                  {s.parameters
                    ? Object.entries(s.parameters)
                        .map(([k, v]) => `${k}:${v}`)
                        .join(', ')
                    : '\u2014'}
                </TableCell>
                <TableCell>
                  <Badge variant={s.enabled ? 'success' : 'destructive'}>
                    {s.enabled ? t('common:enabled', 'Enabled') : t('common:disabled', 'Disabled')}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {s.created_at ? new Date(s.created_at).toLocaleDateString() : '\u2014'}
                </TableCell>
                <TableCell>
                  <div className="flex gap-1">
                    <Button variant="outline" size="sm" onClick={() => onToggle(s.id, s.enabled)}>
                      {t('common:toggle', 'Toggle')}
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => onClone(s.id)}>
                      {t('common:clone', 'Clone')}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-destructive border-destructive/50 hover:bg-destructive/10"
                      onClick={() => onDelete(s.id)}
                    >
                      {t('common:del', 'Del')}
                    </Button>
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
