import * as React from "react"
import { cn } from "../../lib/utils"

const Checkbox = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => (
  <input
    type="checkbox"
    ref={ref}
    className={cn(
      "peer size-4 shrink-0 rounded border border-input bg-transparent",
      "checked:bg-primary checked:text-primary-foreground",
      "focus-visible:outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
      "disabled:cursor-not-allowed disabled:opacity-50",
      "dark:bg-input/30",
      className,
    )}
    {...props}
  />
))
Checkbox.displayName = "Checkbox"

export { Checkbox }
