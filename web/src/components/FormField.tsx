import { type ReactNode } from 'react'
import { type UseFormRegister, type FieldError } from 'react-hook-form'
import { Input } from './ui/input'
import { Label } from './ui/label'

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
      <Label htmlFor={name} className="mb-1 block">{label}</Label>
      {children ?? (
        <Input
          id={name}
          type={type ?? 'text'}
          placeholder={placeholder}
          aria-label={label}
          aria-invalid={!!error}
          {...register(name)}
        />
      )}
      {error && (
        <p role="alert" className="text-destructive text-[11px] mt-1 mb-0">
          {error.message}
        </p>
      )}
    </div>
  )
}
