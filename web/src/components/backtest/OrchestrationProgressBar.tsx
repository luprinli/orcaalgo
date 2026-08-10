"use client"

import { useState, useEffect } from "react"
import { Clock, CheckCircle, XCircle, Loader2 } from "lucide-react"

interface OrchestrationProgressBarProps {
  status: string
  startTime?: number
}

export default function OrchestrationProgressBar({ status, startTime }: OrchestrationProgressBarProps) {
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    if (status !== "running" || !startTime) return
    const timer = setInterval(() => setElapsed(Math.floor((Date.now() - startTime) / 1000)), 1000)
    return () => clearInterval(timer)
  }, [status, startTime])

  if (status === "idle") return null

  const statusConfig: Record<string, { color: string; icon: React.ReactNode; label: string }> = {
    running: { color: "text-blue-600", icon: <Loader2 className="h-3.5 w-3.5 animate-spin" />, label: "Running" },
    completed: { color: "text-green-600", icon: <CheckCircle className="h-3.5 w-3.5" />, label: "Completed" },
    failed: { color: "text-red-600", icon: <XCircle className="h-3.5 w-3.5" />, label: "Failed" },
    cancelled: { color: "text-gray-500", icon: <XCircle className="h-3.5 w-3.5" />, label: "Cancelled" },
    queued: { color: "text-yellow-600", icon: <Clock className="h-3.5 w-3.5" />, label: "Queued" },
  }
  const cfg = statusConfig[status] || statusConfig.failed

  const fmtElapsed = (s: number) => {
    if (s < 60) return `${s}s`
    return `${Math.floor(s / 60)}m ${s % 60}s`
  }

  return (
    <div className={`flex items-center gap-2 px-3 py-1.5 rounded-md text-xs ${cfg.color} bg-background border`}>
      {cfg.icon}
      <span className="font-medium">{cfg.label}</span>
      {status === "running" && <span className="tabular-nums text-muted-foreground"><Clock className="h-3 w-3 inline mr-0.5" />{fmtElapsed(elapsed)}</span>}
    </div>
  )
}
