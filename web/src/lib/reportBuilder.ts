interface TableColumn {
  key: string
  label: string
  format?: 'number' | 'percent' | 'currency' | 'string'
}

function formatValue(value: unknown, format?: TableColumn['format']): string {
  if (value == null) return '-'
  if (format === 'percent') return `${(Number(value)).toFixed(2)}%`
  if (format === 'currency') return `$${(Number(value)).toFixed(2)}`
  if (format === 'number') return Number(value).toLocaleString()
  return String(value)
}

export function buildTable(columns: TableColumn[], rows: Record<string, unknown>[]): string {
  const header = `| ${columns.map((c) => c.label).join(' | ')} |`
  const separator = `|${columns.map(() => '---').join('|')}|`
  const body = rows
    .map((row) => `| ${columns.map((c) => formatValue(row[c.key], c.format)).join(' | ')} |`)
    .join('\n')
  return `${header}\n${separator}\n${body}`
}

export function buildReport(title: string, sections: { heading: string; content: string }[]): string {
  const lines = [`# ${title}`, '', `Generated: ${new Date().toISOString()}`, '']
  for (const section of sections) {
    lines.push(`## ${section.heading}`, '', section.content, '')
  }
  return lines.join('\n')
}

export function buildBacktestReport(metrics: Record<string, unknown>, title = 'Backtest Report'): string {
  const metricsSection = buildTable(
    [
      { key: 'metric', label: 'Metric' },
      { key: 'value', label: 'Value' },
    ],
    Object.entries(metrics).map(([metric, value]) => ({ metric, value })),
  )
  return buildReport(title, [{ heading: 'Metrics', content: metricsSection }])
}

export function buildAttributionReport(slices: { label: string; stats: Record<string, unknown> }[]): string {
  const sections = slices.map((slice) => {
    const table = buildTable(
      [
        { key: 'metric', label: 'Metric' },
        { key: 'value', label: 'Value' },
      ],
      Object.entries(slice.stats).map(([metric, value]) => ({ metric, value })),
    )
    return { heading: slice.label, content: table }
  })
  return buildReport('Attribution Report', sections)
}

export async function lintMarkdown(markdown: string): Promise<string> {
  const [{ lint }, { applyFixes }] = await Promise.all([
    import('markdownlint/promise'),
    import('markdownlint'),
  ])
  const results = await lint({
    strings: { report: markdown },
    config: { default: true },
  })
  const errors = results.report
  if (!errors || errors.length === 0) return markdown
  return applyFixes(markdown, errors)
}
