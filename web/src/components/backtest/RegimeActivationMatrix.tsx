import { Switch } from '../ui/switch'
import { Card, CardContent, CardHeader, CardTitle } from '../ui/card'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '../ui/table'
import { Badge } from '../ui/badge'

interface StrategyEntry {
  strategy: string
  calm: boolean
  trending: boolean
  highVol: boolean
  crisis: boolean
}

const REGIME_LABELS: Record<number, string> = {
  0: 'Calm',
  1: 'Trending',
  2: 'High Vol',
  3: 'Crisis',
}
const REGIME_COLORS: Record<number, string> = {
  0: 'bg-green-500/20 text-green-400',
  1: 'bg-blue-500/20 text-blue-400',
  2: 'bg-amber-500/20 text-amber-400',
  3: 'bg-red-500/20 text-red-400',
}

const DEFAULT_MATRIX: StrategyEntry[] = [
  { strategy: 'Grid Trading', calm: false, trending: false, highVol: false, crisis: false },
  { strategy: 'Trend Following', calm: false, trending: true, highVol: false, crisis: false },
  { strategy: 'Session Scalp', calm: true, trending: true, highVol: true, crisis: false },
  { strategy: 'Mean Reversion', calm: true, trending: false, highVol: false, crisis: false },
  { strategy: 'ORB', calm: false, trending: true, highVol: true, crisis: false },
  { strategy: 'Pairs Trading', calm: true, trending: false, highVol: true, crisis: false },
  { strategy: 'Vol Harvesting', calm: false, trending: false, highVol: true, crisis: false },
]

interface RegimeActivationMatrixProps {
  editable?: boolean
}

export default function RegimeActivationMatrix({ editable = false }: RegimeActivationMatrixProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Strategy ↔ Regime Activation Matrix</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-40">Strategy</TableHead>
              {[0, 1, 2, 3].map((r) => (
                <TableHead key={r} className="text-center">
                  <Badge variant="outline" className={REGIME_COLORS[r]}>
                    {REGIME_LABELS[r]}
                  </Badge>
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {DEFAULT_MATRIX.map((entry) => (
              <TableRow key={entry.strategy}>
                <TableCell className="font-medium text-sm">{entry.strategy}</TableCell>
                {(['calm', 'trending', 'highVol', 'crisis'] as const).map((regime) => (
                  <TableCell key={regime} className="text-center">
                    <Switch
                      checked={entry[regime]}
                      disabled={!editable}
                      aria-label={`${entry.strategy} in ${regime}`}
                    />
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {!editable && (
          <p className="text-xs text-muted-foreground mt-3">
            Read-only view. Grid Trading is disabled by default. Configure via GKR configs.
          </p>
        )}
      </CardContent>
    </Card>
  )
}
