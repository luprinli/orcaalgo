import { useToastStore, type ToastType } from '../stores/toastStore'

const ICONS: Record<ToastType, string> = {
  success: '\u2713',
  error: '\u2717',
  warn: '\u25B3',
  info: '\u2139',
}

const COLORS: Record<ToastType, string> = {
  success: 'var(--success)',
  error: 'var(--danger)',
  warn: 'var(--warn)',
  info: 'var(--accent)',
}

export default function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts)
  const removeToast = useToastStore((s) => s.removeToast)

  if (toasts.length === 0) return null

  return (
    <div
      style={{
        position: 'fixed',
        top: 16,
        right: 16,
        zIndex: 2000,
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        maxWidth: 360,
        pointerEvents: 'none',
      }}
    >
      {toasts.map((t) => (
        <div
          key={t.id}
          role="alert"
          style={{
            background: 'var(--bg-card)',
            border: `1px solid ${COLORS[t.type]}`,
            borderLeft: `4px solid ${COLORS[t.type]}`,
            borderRadius: 'var(--radius-sm)',
            padding: '10px 14px',
            display: 'flex',
            alignItems: 'flex-start',
            gap: 10,
            fontSize: 13,
            color: 'var(--text-primary)',
            boxShadow: '0 4px 12px rgba(0,0,0,0.4)',
            pointerEvents: 'auto',
            animation: 'toast-in 0.2s ease-out',
          }}
        >
          <span style={{ color: COLORS[t.type], fontWeight: 700, fontSize: 14, flexShrink: 0 }}>
            {ICONS[t.type]}
          </span>
          <span style={{ flex: 1 }}>{t.message}</span>
          <button
            onClick={() => removeToast(t.id)}
            style={{
              background: 'none', border: 'none', color: 'var(--text-secondary)',
              cursor: 'pointer', fontSize: 14, padding: 0, lineHeight: 1,
              flexShrink: 0,
            }}
            aria-label="Dismiss"
          >
            \u2715
          </button>
        </div>
      ))}
    </div>
  )
}
