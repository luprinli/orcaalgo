import { useMemo } from 'react'
import type { MonthlyReturn } from '../types/api'

interface YearlySummaryTableProps {
  data: MonthlyReturn[]
}

interface YearSummary {
  year: number
  return_pct: number
  num_months: number
  positive_months: number
  best_month: { month: number; ret: number } | null
  worst_month: { month: number; ret: number } | null
}

const MONTH_LABELS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export default function YearlySummaryTable({ data }: YearlySummaryTableProps) {
  const years = useMemo(() => {
    const byYear: Record<number, { month: number; ret: number }[]> = {}
    for (const d of data) {
      if (!byYear[d.year]) byYear[d.year] = []
      byYear[d.year].push({ month: d.month, ret: d.return_pct })
    }

    const result: YearSummary[] = []
    for (const [y, months] of Object.entries(byYear)) {
      const year = Number(y)
      const total = months.reduce((s, m) => s + m.ret, 0)
      const pos = months.filter(m => m.ret > 0).length
      const best = months.reduce((b, m) => m.ret > (b?.ret ?? -Infinity) ? m : b, null as typeof months[0] | null)
      const worst = months.reduce((w, m) => m.ret < (w?.ret ?? Infinity) ? m : w, null as typeof months[0] | null)
      result.push({ year, return_pct: total, num_months: months.length, positive_months: pos, best_month: best, worst_month: worst })
    }
    return result.sort((a, b) => a.year - b.year)
  }, [data])

  if (years.length === 0) return null

  return (
    <div className="card">
      <div className="card-header"><h3>Yearly Performance</h3></div>
      <div style={{ overflowX: 'auto' }}>
        <table className="data-table" style={{ fontSize: 12 }}>
          <thead>
            <tr>
              <th>Year</th>
              <th>Return</th>
              <th>Active Mo.</th>
              <th>Positive</th>
              <th>Best Month</th>
              <th>Worst Month</th>
            </tr>
          </thead>
          <tbody>
            {years.map((y) => (
              <tr key={y.year}>
                <td><strong>{y.year}</strong></td>
                <td style={{ color: y.return_pct >= 0 ? 'var(--success)' : 'var(--danger)' }}>
                  {y.return_pct >= 0 ? '+' : ''}{y.return_pct.toFixed(2)}%
                </td>
                <td>{y.num_months}</td>
                <td>{y.positive_months}/{y.num_months}</td>
                <td style={{ color: 'var(--success)' }}>
                  {y.best_month ? `${MONTH_LABELS[y.best_month.month - 1]} ${y.best_month.ret >= 0 ? '+' : ''}${y.best_month.ret.toFixed(2)}%` : '--'}
                </td>
                <td style={{ color: 'var(--danger)' }}>
                  {y.worst_month ? `${MONTH_LABELS[y.worst_month.month - 1]} ${y.worst_month.ret >= 0 ? '+' : ''}${y.worst_month.ret.toFixed(2)}%` : '--'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
