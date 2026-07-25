import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "../../lib/utils"

const badgeVariants = cva(
  "inline-flex items-center rounded-4xl border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 tabular-nums",
  {
    variants: {
      variant: {
        default: "border-transparent bg-primary text-primary-foreground",
        secondary: "border-transparent bg-secondary text-secondary-foreground",
        destructive: "border-transparent bg-destructive/10 text-destructive dark:bg-destructive/20",
        outline: "border-border text-foreground",
        ghost: "border-transparent hover:bg-muted dark:hover:bg-muted/50",
        link: "border-transparent text-primary underline-offset-4 hover:underline",
        /* --- Orca trading variants --- */
        success: "border-transparent bg-trading-success/10 text-trading-success",
        warning: "border-transparent bg-trading-warning/10 text-trading-warning",
        info: "border-transparent bg-trading-info/10 text-trading-info",
      },
      size: {
        default: "h-5",
        sm: "h-4 text-[10px] px-1.5",
      },
    },
    defaultVariants: { variant: "default", size: "default" },
  },
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, size, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant, size }), className)} {...props} />
  )
}

export { Badge, badgeVariants }
