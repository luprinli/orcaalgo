import { describe, it, expect } from 'vitest'
import { lintMarkdown } from '../lib/reportBuilder'

describe('lintMarkdown', () => {
  it('passes clean markdown through unchanged', { timeout: 15000 }, async () => {
    const input = '# Title\n\n## Section\n\nContent here.\n'
    const output = await lintMarkdown(input)
    expect(output).toBe(input)
  })

  it('strips trailing spaces from lines', async () => {
    const input = '# Title   \n\n## Section\n'
    const output = await lintMarkdown(input)
    expect(output).not.toContain('   ')
    expect(output).toContain('# Title')
  })

  it('inserts blank line before heading when missing', async () => {
    const input = 'text\n## Heading\n'
    const output = await lintMarkdown(input)
    expect(output).toMatch(/text\n\n## Heading/)
  })

  it('collapses multiple consecutive blank lines', async () => {
    const input = '# Title\n\n\n\n## Section\n'
    const output = await lintMarkdown(input)
    const blankCount = (output.match(/\n\n/g) || []).length
    const tripleCount = (output.match(/\n\n\n/g) || []).length
    expect(tripleCount).toBe(0)
    expect(blankCount).toBeGreaterThan(0)
  })

  it('removes hard tabs', async () => {
    const input = '# Title\n\n\t- List item\n'
    const output = await lintMarkdown(input)
    expect(output).not.toContain('\t')
  })

  it('returns original markdown when there are no fixable errors', async () => {
    const input = '# Clean\n\nNo issues.\n'
    const output = await lintMarkdown(input)
    expect(output).toBe(input)
  })

  it('handles empty string', async () => {
    const output = await lintMarkdown('')
    expect(output).toBe('')
  })

  it('preserves intentional formatting', async () => {
    const input = '# Report\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n**bold** and *italic*\n'
    const output = await lintMarkdown(input)
    expect(output).toContain('| A | B |')
    expect(output).toContain('**bold**')
    expect(output).toContain('*italic*')
  })
})
