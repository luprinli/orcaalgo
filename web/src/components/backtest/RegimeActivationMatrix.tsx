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
  calmKelly?: number
  trendingKelly?: number
  highVolKelly?: number
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
  { strategy: 'Grid Trading', calm: true, trending: false, highVol: false, crisis: false, calmKelly: 0.25 },
  { strategy: 'Vol-Adjusted Grid', calm: true, trending: false, highVol: false, crisis: false, calmKelly: 0.15 },
  { strategy: 'Trend Following', calm: false, trending: true, highVol: false, crisis: false, trendingKelly: 0.25 },
  { strategy: 'Session Scalp', calm: true, trending: true, highVol: true, crisis: false, calmKelly: 0.25, trendingKelly: 0.25, highVolKelly: 0.15 },
  { strategy: 'Intraday MR', calm: true, trending: false, highVol: false, crisis: false, calmKelly: 0.25 },
  { strategy: 'VWAP MR', calm: true, trending: false, highVol: false, crisis: false, calmKelly: 0.25 },
  { strategy: 'ORB (5m)', calm: true, trending: true, highVol: true, crisis: false, calmKelly: 0.10, trendingKelly: 0.25, highVolKelly: 0.15 },
  { strategy: 'ORB (15m)', calm: true, trending: true, highVol: true, crisis: false, calmKelly: 0.10, trendingKelly: 0.25, highVolKelly: 0.15 },
  { strategy: 'Pairs Trading', calm: true, trending: false, highVol: true, crisis: false, calmKelly: 0.25, highVolKelly: 0.15 },
  { strategy: 'Vol Harvesting', calm: false, trending: false, highVol: true, crisis: false, highVolKelly: 0.15 },
  { strategy: 'Dragon Trend', calm: false, trending: true, highVol: true, crisis: false, trendingKelly: 0.25, highVolKelly: 0.15 },
  { strategy: 'Volume Scalp', calm: true, trending: true, highVol: false, crisis: false, calmKelly: 0.25, trendingKelly: 0.25 },
  { strategy: 'VIX Futures Carry', calm: false, trending: false, highVol: true, crisis: false, highVolKelly: 0.15 },
  { strategy: 'MA Crossover', calm: true, trending: true, highVol: true, crisis: false },
  { strategy: 'RSI(2) Reversion', calm: true, trending: true, highVol: true, crisis: false },
  { strategy: 'Donchian Breakout', calm: true, trending: true, highVol: true, crisis: false },
  { strategy: 'Keltner MACD', calm: true, trending: true, highVol: true, crisis: false },
  { strategy: 'Ichimoku Cloud', calm: true, trending: true, highVol: true, crisis: false },
]

function kellyDisplay(entry: StrategyEntry, regime: 'calm' | 'trending' | 'highVol'): string {
  const map: Record<string, string | undefined> = {
    calm: entry.calmKelly !== undefined ? entry.calmKelly.toFixed(2) : undefined,
    trending: entry.trendingKelly !== undefined ? entry.trendingKelly.toFixed(2) : undefined,
    highVol: entry.highVolKelly !== undefined ? entry.highVolKelly.toFixed(2) : undefined,
  }
  return map[regime] || '-'
}

interface RegimeActivationMatrixProps {
  editable?: boolean
}

export default function RegimeActivationMatrix({ editable = false }: RegimeActivationMatrixProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Strategy vs Regime Activation Matrix</CardTitle>
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
                    <div className="flex flex-col items-center gap-0.5">
                      <Switch
                        checked={entry[regime]}
                        disabled={!editable}
                        aria-label={`${entry.strategy} in ${regime}`}
                      />
                      {regime !== 'crisis' && entry[regime] && (
                        <span className="text-[10px] text-muted-foreground">
                          k={kellyDisplay(entry, regime)}
                        </span>
                      )}
                    </div>
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
        {!editable && (
          <p className="text-xs text-muted-foreground mt-3">
            Read-only view. Grid Trading is disabled by default (HP agenda). Kelly values from regime activation matrix.
          </p>
        )}
      </CardContent>
    </Card>
  )
}
