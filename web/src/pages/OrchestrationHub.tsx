"use client"

import { useEffect } from "react"

export default function OrchestrationHub() {
  useEffect(() => {
    window.location.replace("/backtest?mode=orchestrated")
  }, [])

  return null
}
