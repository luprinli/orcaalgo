export function formatNumber(v: number | undefined | null, d = 2): string {
  return v != null ? Number(v).toFixed(d) : '--'
}

export function formatUSD(v: number | undefined | null, d = 2): string {
  if (v == null) return '--'
  return `$${Number(v).toFixed(d)}`
}

export function formatPct(v: number | undefined | null, d = 2): string {
  if (v == null) return '--'
  return `${(v * 100).toFixed(d)}%`
}

export function formatPctRaw(v: number | undefined | null, d = 2): string {
  if (v == null) return '--'
  return `${Number(v).toFixed(d)}%`
}
