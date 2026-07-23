import { type ReactNode } from 'react'

export interface BacktestConfigBarProps {
  start: string
  end: string
  capital: string
  profile: string
  mode: 'single' | 'matrix'
  dataSource: string
  sizingPercent: number
  propFirmEnabled: boolean
  optimizeEnabled: boolean
  twoStageEnabled: boolean
  onStartChange: (v: string) => void
  onEndChange: (v: string) => void
  onCapitalChange: (v: string) => void
  onProfileChange: (v: string) => void
  onModeChange: (v: 'single' | 'matrix') => void
  onDataSourceChange: (v: string) => void
  onSizingPercentChange: (v: number) => void
  onPropFirmEnabledChange: (v: boolean) => void
  onOptimizeEnabledChange: (v: boolean) => void
  onTwoStageEnabledChange: (v: boolean) => void
  children?: ReactNode
}

export default function BacktestConfigBar(props: BacktestConfigBarProps) {
  const {
    start, end, capital, profile, mode, dataSource, sizingPercent, propFirmEnabled, optimizeEnabled, twoStageEnabled,
    onStartChange, onEndChange, onCapitalChange, onProfileChange, onModeChange, onDataSourceChange,
    onSizingPercentChange, onPropFirmEnabledChange, onOptimizeEnabledChange, onTwoStageEnabledChange,
    children,
  } = props

  return (
    <>
      <div className="flex gap-2 mb-2" style={{ flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <div style={{ flex: '0 0 140px' }}><label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>Start</label><input className="input" type="date" value={start} onChange={e => onStartChange(e.target.value)} style={{ fontSize: 12, padding: '4px 6px' }} /></div>
        <div style={{ flex: '0 0 140px' }}><label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>End</label><input className="input" type="date" value={end} onChange={e => onEndChange(e.target.value)} style={{ fontSize: 12, padding: '4px 6px' }} /></div>
        <div style={{ flex: '0 0 100px' }}><label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>Capital</label><input className="input" type="number" value={capital} onChange={e => onCapitalChange(e.target.value)} style={{ fontSize: 12, padding: '4px 6px' }} /></div>
        <div style={{ flex: '0 0 120px' }}><label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>Gate</label><select className="input" value={profile} onChange={e => onProfileChange(e.target.value)} style={{ fontSize: 11, padding: '4px 6px' }}><option value="research">Research</option><option value="paper">Paper</option><option value="pretrade">Pre-Trade</option><option value="production_guarded">Prod</option></select></div>
        <div style={{ flex: '0 0 120px' }}><label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>Mode</label><select className="input" value={mode} onChange={e => onModeChange(e.target.value as 'single' | 'matrix')} style={{ fontSize: 11, padding: '4px 6px' }}><option value="single">Single</option><option value="matrix">Matrix</option></select></div>
        <div style={{ flex: '0 0 110px' }}><label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>Data</label><select className="input" value={dataSource} onChange={e => onDataSourceChange(e.target.value)} style={{ fontSize: 11, padding: '4px 6px' }}><option value="synthetic">Synthetic</option><option value="stooq">Stooq</option></select></div>
        <div style={{ flex: '0 0 110px', display: 'flex', flexDirection: 'column', justifyContent: 'flex-end' }}>
          <label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>Size %</label>
          <select className="input" style={{ fontSize: 11, padding: '4px 6px' }} value={sizingPercent} onChange={e => onSizingPercentChange(parseFloat(e.target.value))}>
            <option value={0.005}>0.5%</option>
            <option value={0.01}>1.0%</option>
            <option value={0.02}>2.0%</option>
            <option value={0.03}>3.0%</option>
            <option value={0.05}>5.0%</option>
            <option value={0.10}>10.0%</option>
          </select>
        </div>
        <div style={{ flex: '0 0 90px', display: 'flex', flexDirection: 'column', justifyContent: 'flex-end' }}>
          <label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>PropFirm</label>
          <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', fontSize: 11, padding: '4px 0' }}>
            <input type="checkbox" checked={propFirmEnabled} onChange={e => onPropFirmEnabledChange(e.target.checked)} style={{ marginRight: 4 }} />{propFirmEnabled ? 'On' : 'Off'}
          </label>
        </div>
        <div style={{ flex: '0 0 90px', display: 'flex', flexDirection: 'column', justifyContent: 'flex-end' }}>
          <label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }}>Optimize</label>
          <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', fontSize: 11, padding: '4px 0' }}>
            <input type="checkbox" checked={optimizeEnabled} onChange={e => onOptimizeEnabledChange(e.target.checked)} style={{ marginRight: 4 }} />{optimizeEnabled ? 'On' : 'Off'}
          </label>
        </div>
        {mode === 'matrix' && (
          <div style={{ flex: '0 0 100px', display: 'flex', flexDirection: 'column', justifyContent: 'flex-end' }}>
            <label className="text-muted" style={{ fontSize: 10, marginBottom: 2, display: 'block' }} title="Broad 1d screen → skip non-viable pairs before intraday runs">Two-Stage</label>
            <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer', fontSize: 11, padding: '4px 0' }}>
              <input type="checkbox" checked={twoStageEnabled} onChange={e => onTwoStageEnabledChange(e.target.checked)} style={{ marginRight: 4 }} />{twoStageEnabled ? 'On' : 'Off'}
            </label>
          </div>
        )}
      </div>
      {children}
    </>
  )
}
