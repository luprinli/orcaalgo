import { type ReactNode } from 'react'
import { type UseFormRegister, type FieldError } from 'react-hook-form'

interface FormFieldProps {
  label: string
  name: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  register: UseFormRegister<any>
  error?: FieldError
  type?: string
  placeholder?: string
  children?: ReactNode
  className?: string
}

export function FormField({ label, name, register, error, type, placeholder, children, className }: FormFieldProps) {
  return (
    <div className={className}>
      <label className="text-muted" style={{ display: 'block', marginBottom: 4 }}>{label}</label>
      {children ?? (
        <input
          className="input"
          type={type ?? 'text'}
          placeholder={placeholder}
          aria-label={label}
          aria-invalid={!!error}
          {...register(name)}
        />
      )}
      {error && (
        <p role="alert" style={{ color: 'var(--danger)', fontSize: 11, margin: '4px 0 0' }}>
          {error.message}
        </p>
      )}
    </div>
  )
}
