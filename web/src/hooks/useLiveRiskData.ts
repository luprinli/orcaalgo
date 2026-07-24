import { useState, useEffect, useCallback } from 'react'
import { useWebSocket } from './useWebSocket'
import { risk } from '../api/client'

interface RiskData {
  halted: boolean
  reason?: string
  balance: number
  equity: number
  daily_pnl_pct: number
  drawdown_used: number
  daily_loss_used: number
  daily_limit_pct: number
  max_dd_pct: number
  regime?: string
  confidence?: number
  vix?: number
  sentiment?: string
  consistency_multiplier?: number
}

interface UseLiveRiskDataReturn {
  riskData: RiskData | null
  connected: boolean
  isHalted: boolean
  error: string | null
  refetch: () => void
}

export function useLiveRiskData(): UseLiveRiskDataReturn {
  const { connected, lastMessage } = useWebSocket('risk', {
    maxReconnects: 30,
    reconnectInterval: 2000,
  })
  const [restData, setRestData] = useState<RiskData | null>(null)
  const [error, setError] = useState<string | null>(null)

  const fetchRest = useCallback(async () => {
    try {
      const data = await risk.status()
      setRestData(data as RiskData)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch risk status')
    }
  }, [])

  useEffect(() => {
    if (!connected) {
      fetchRest()
      const interval = setInterval(fetchRest, 10000)
      return () => clearInterval(interval)
    }
  }, [connected, fetchRest])

  const riskData = (lastMessage as RiskData) ?? restData

  return {
    riskData,
    connected,
    isHalted: riskData?.halted ?? false,
    error: connected ? null : error,
    refetch: fetchRest,
  }
}
