import { useTranslation } from 'react-i18next'
import { Card, CardHeader, CardTitle, CardContent } from '../../components/ui/card'
import { Badge } from '../../components/ui/badge'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../components/ui/table'

export interface SignalEntry {
  symbol: string
  side: string
  confidence: number
  reason: string
  timestamp: string
}

interface SignalsTabProps {
  signals: SignalEntry[]
}

export default function SignalsTab({ signals }: SignalsTabProps) {
  const { t } = useTranslation()

  return (
    <div>
      <Card>
        <CardHeader>
          <CardTitle>{t('monitor:signals.title', 'Signal History')}</CardTitle>
        </CardHeader>
        <CardContent>
          {signals.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">
              {t('monitor:signals.empty', 'No signals recorded yet. Signals will appear here as strategies generate them.')}
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('monitor:signals.time', 'Time')}</TableHead>
                  <TableHead>{t('monitor:signals.symbol', 'Symbol')}</TableHead>
                  <TableHead>{t('monitor:signals.side', 'Side')}</TableHead>
                  <TableHead>{t('monitor:signals.confidence', 'Confidence')}</TableHead>
                  <TableHead>{t('monitor:signals.reason', 'Reason')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {signals.map((s, i) => (
                  <TableRow key={i}>
                    <TableCell className="text-xs">{new Date(s.timestamp).toLocaleString()}</TableCell>
                    <TableCell className="font-medium">{s.symbol}</TableCell>
                    <TableCell>
                      <Badge variant={s.side === 'BUY' ? 'success' : s.side === 'SELL' ? 'destructive' : 'outline'} size="sm">
                        {s.side}
                      </Badge>
                    </TableCell>
                    <TableCell className="tabular-nums">{(s.confidence * 100).toFixed(1)}%</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{s.reason}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
