import { useState } from "react";

export interface MetricCardProps {
  label: string;
  value: string | number;
  format?: "number" | "percent" | "percent_raw" | "currency" | "decimal";
  color?: "default" | "positive" | "negative" | "auto";
  tooltip?: string;
  trend?: "up" | "down" | "neutral";
  onClick?: () => void;
  skeleton?: boolean;
}

function fmtUSD(n: number): string {
  return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 2 }).format(n);
}

function fmtNum(n: number, d: number): string {
  return n.toFixed(d);
}

export default function MetricCard({
  label,
  value,
  format = "decimal",
  color = "default",
  tooltip,
  trend,
  onClick,
  skeleton,
}: MetricCardProps) {
  const [showTooltip, setShowTooltip] = useState(false);

  if (skeleton) {
    return (
      <div className="rounded-lg bg-card ring-1 ring-foreground/10 p-3 flex flex-col gap-0.5 opacity-50">
        <div className="h-3 w-3/5 rounded bg-muted animate-pulse" />
        <div className="h-5 w-2/5 rounded bg-muted animate-pulse" />
      </div>
    );
  }

  const num = typeof value === "string" ? parseFloat(value) : value;

  const formatted = (() => {
    if (typeof value === "string" && isNaN(num)) return value;
    switch (format) {
      case "number":
        return fmtNum(num, 0);
      case "percent":
        return typeof value === "number" ? `${(num * 100).toFixed(1)}%` : String(value);
      case "percent_raw":
        return typeof value === "number" ? `${num.toFixed(1)}%` : String(value);
      case "currency":
        return fmtUSD(num);
      default:
        return fmtNum(num, 2);
    }
  })();

  const resolvedColor = (() => {
    if (color === "default") return undefined;
    if (color === "auto") {
      if (typeof value === "string") return undefined;
      return num >= 0 ? "var(--trading-success)" : "var(--trading-danger)";
    }
    return color === "positive" ? "var(--trading-success)" : color === "negative" ? "var(--trading-danger)" : undefined;
  })();

  const trendIcon = trend === "up" ? "↑" : trend === "down" ? "↓" : "";

  return (
    <div
      className="rounded-lg bg-card ring-1 ring-foreground/10 p-3 flex flex-col gap-0.5"
      onClick={onClick}
      style={{ cursor: onClick ? "pointer" : undefined, position: "relative" }}
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <div className="text-[10px] text-muted-foreground font-medium uppercase tracking-wider">
        {label}
        {trendIcon && <span className="ml-1 text-[10px]">{trendIcon}</span>}
      </div>
      <div className="text-lg font-bold tabular-nums" style={{ color: resolvedColor }}>
        {formatted}
      </div>
      {tooltip && showTooltip && (
        <div
          className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1 px-2 py-1 text-[11px] whitespace-nowrap rounded bg-popover border border-border shadow z-[100] pointer-events-none"
        >
          {tooltip}
        </div>
      )}
    </div>
  );
}
