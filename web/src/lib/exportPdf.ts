import { lintMarkdown } from './reportBuilder'

const defaultOptions: Record<string, unknown> = {
  margin: [10, 10, 10, 10],
  filename: 'orca-report.pdf',
  image: { type: 'jpeg', quality: 0.98 },
  html2canvas: { scale: 2, useCORS: true },
  jsPDF: { unit: 'mm', format: 'a4', orientation: 'portrait' },
}

export async function exportPdf(
  element: HTMLElement,
  filename?: string,
  options?: Record<string, unknown>,
): Promise<void> {
  const { default: html2pdf } = await import('html2pdf.js') as { default: () => { set: (o: Record<string, unknown>) => { from: (el: HTMLElement) => { save: () => void } } } }
  const opts = {
    ...defaultOptions,
    ...options,
    filename: filename ?? 'orca-report.pdf',
  }
  html2pdf().set(opts).from(element).save()
}

export function renderMarkdownToHtml(markdown: string): string {
  const { marked } = window as unknown as { marked?: { parse: (md: string) => string } }
  if (marked) return marked.parse(markdown)
  return `<pre>${markdown}</pre>`
}

export async function exportMarkdownAsPdf(
  markdown: string,
  filename?: string,
): Promise<void> {
  const cleaned = await lintMarkdown(markdown)
  const container = document.createElement('div')
  container.style.cssText =
    'padding:20px;font-family:-apple-system,BlinkMacSystemFont,sans-serif;color:#1f2328;background:#fff;max-width:800px;margin:0 auto;'
  container.innerHTML = renderMarkdownToHtml(cleaned)
  document.body.appendChild(container)
  try {
    await exportPdf(container, filename)
  } finally {
    document.body.removeChild(container)
  }
}
