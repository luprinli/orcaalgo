import { useState, useRef, useEffect, type ReactNode } from 'react'

export interface MultiSelectItem {
  key: string
  name: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  [k: string]: any
}

export interface MultiSelectProps<T extends MultiSelectItem> {
  items: T[]
  selectedKeys: Set<string>
  onToggle: (key: string) => void
  onSelectAll?: () => void
  onDeselectAll?: () => void
  label: string
  placeholder?: string
  rowRender?: (item: T, selected: boolean) => ReactNode
  extraActions?: ReactNode
  searchPlaceholder?: string
  width?: number
}

export default function MultiSelect<T extends MultiSelectItem>({
  items, selectedKeys, onToggle, onSelectAll, onDeselectAll,
  label, placeholder = 'Select...', rowRender, extraActions,
  searchPlaceholder = 'Filter...', width,
}: MultiSelectProps<T>) {
  const [open, setOpen] = useState(false)
  const [filter, setFilter] = useState('')
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false) }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const selected = items.filter(i => selectedKeys.has(i.key))
  const filtered = filter ? items.filter(i => i.name.toUpperCase().includes(filter.toUpperCase())) : items

  const selCount = selectedKeys.size
  const total = items.length

  const chip = (key: string, name: string) => (
    <span key={key} className="badge" style={{ cursor: 'pointer', padding: '1px 5px', fontSize: 10, background: 'var(--accent)', color: '#fff', lineHeight: '18px', whiteSpace: 'nowrap' }}
          onClick={e => { e.stopPropagation(); onToggle(key) }}>{name}</span>
  )

  return (
    <div ref={ref} style={{ position: 'relative', width: width ? width : undefined, flex: width ? undefined : 1, minWidth: 0 }}>
      <div className="flex-between mb-1">
        <label className="text-muted" style={{ fontSize: 10 }}>{label}</label>
        <span className="text-muted" style={{ fontSize: 10 }}>{selCount}/{total}</span>
      </div>
      <div className="flex gap-1 flex-wrap" style={{ minHeight: 28, padding: '2px 6px', border: '1px solid var(--border)', borderRadius: 4, background: 'var(--input-bg)', cursor: 'pointer', fontSize: 10 }}
           onClick={() => setOpen(v => !v)}>
        {selCount === 0 && <span className="text-muted" style={{ fontSize: 10, lineHeight: '22px' }}>{placeholder}</span>}
        {selected.slice(0, 6).map(s => chip(s.key, s.name))}
        {selCount > 6 && <span className="text-muted" style={{ fontSize: 10, lineHeight: '22px' }}>+{selCount - 6}</span>}
      </div>
      {open && (
        <div className="card" style={{ position: 'absolute', zIndex: 200, minWidth: 240, maxHeight: 340, overflowY: 'auto', marginTop: 2, padding: 8, boxShadow: '0 8px 32px rgba(0,0,0,.3)' }}>
          {onSelectAll && (
            <div className="flex gap-1 mb-2 flex-wrap">
              <button className="btn btn-outline" style={{ fontSize: 10, padding: '2px 6px' }} onClick={onSelectAll}>All ({total})</button>
              {onDeselectAll && <button className="btn btn-outline" style={{ fontSize: 10, padding: '2px 6px' }} onClick={onDeselectAll}>None</button>}
              {extraActions}
            </div>
          )}
          <input className="input" style={{ fontSize: 10, padding: '3px 6px', marginBottom: 4, width: '100%', boxSizing: 'border-box' }}
                 placeholder={searchPlaceholder} value={filter} onChange={e => setFilter(e.target.value)} />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))', gap: '1px 4px' }}>
            {filtered.map(item => {
              const sel = selectedKeys.has(item.key)
              return rowRender ? rowRender(item, sel) : (
                <div key={item.key}
                     style={{ padding: '3px 6px', borderRadius: 3, cursor: 'pointer', fontSize: 10, background: sel ? 'var(--accent)' : 'transparent', color: sel ? '#fff' : 'inherit', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                     onClick={() => onToggle(item.key)}>{item.name}</div>
              )
            })}
          </div>
          {filtered.length === 0 && <p className="text-muted" style={{ fontSize: 10, padding: 8, textAlign: 'center' }}>No items found</p>}
        </div>
      )}
    </div>
  )
}
