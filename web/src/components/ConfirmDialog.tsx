import { useTranslation } from 'react-i18next'

interface ConfirmDialogProps {
  title?: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export default function ConfirmDialog({
  title,
  message,
  confirmLabel,
  cancelLabel,
  danger = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const { t } = useTranslation()
  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 1100,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
      onClick={(e) => { if (e.target === e.currentTarget) onCancel() }}
      onKeyDown={(e) => { if (e.key === 'Escape') onCancel() }}
    >
      <div
        className="card"
        style={{ width: 400, maxWidth: '90vw' }}
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
      >
        <h2 id="confirm-title" style={{ margin: '0 0 8px' }}>{title ?? t('components:confirmDialog.confirm', 'Confirm')}</h2>
        <p className="text-muted" style={{ margin: '0 0 20px' }}>{message}</p>
        <div className="flex gap-2" style={{ justifyContent: 'flex-end' }}>
          <button className="btn btn-outline" onClick={onCancel}>
            {cancelLabel ?? t('common:cancel', 'Cancel')}
          </button>
          <button
            className={`btn ${danger ? 'btn-danger' : 'btn-primary'}`}
            onClick={onConfirm}
            autoFocus
          >
            {confirmLabel ?? t('common:confirm', 'Confirm')}
          </button>
        </div>
      </div>
    </div>
  )
}
