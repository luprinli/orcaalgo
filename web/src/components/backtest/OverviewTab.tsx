import type { RegimeStat } from '../../types/api'

interface Props {
  regimeStats: RegimeStat[]
}

export default function OverviewTab({ regimeStats }: Props) {
  if (!regimeStats || regimeStats.length === 0) return null

  return (
    <div>
      <div className="mb-4">
        <h2>Regime Breakdown</h2>
        <div style={{ overflowX: 'auto' }}>
          <table className="data-table">
            <thead>
              <tr>
                <th>Regime</th>
                <th>Trades</th>
                <th>Win Rate</th>
                <th>Total Return</th>
                <th>Max DD</th>
                <th>Profit Factor</th>
              </tr>
            </thead>
            <tbody>
              {regimeStats.map((r) => (
                <tr key={r.regime}>
                  <td>{r.label}</td>
                  <td>{r.num_trades}</td>
                  <td>{(r.win_rate * 100).toFixed(1)}%</td>
                  <td style={{ color: r.total_return >= 0 ? 'var(--trading-success)' : 'var(--trading-danger)' }}>
                    {(r.total_return * 100).toFixed(1)}%
                  </td>
                  <td>{(r.max_drawdown * 100).toFixed(1)}%</td>
                  <td>{r.profit_factor?.toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
