interface ChartOverlayButtonsProps {
  isFullscreen: boolean
  onToggleFullscreen: () => void
  onExportPNG: () => void
  drawingMode: boolean
  onToggleDrawing: () => void
  lineCount: number
  onClearLines: () => void
}

export default function ChartOverlayButtons({
  isFullscreen, onToggleFullscreen, onExportPNG,
  drawingMode, onToggleDrawing, lineCount, onClearLines,
}: ChartOverlayButtonsProps) {
  return (
    <>
      <button
        onClick={onToggleFullscreen}
        title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
        aria-label={isFullscreen ? 'Exit fullscreen' : 'Enter fullscreen'}
        style={{
          position: 'absolute', top: 8, right: 8, zIndex: 10,
          background: 'transparent', border: 'none', color: '#ffffff',
          opacity: 0.7, cursor: 'pointer', fontSize: 18, padding: '2px 4px',
          lineHeight: 1, fontFamily: 'sans-serif',
        }}
        onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
        onMouseLeave={e => (e.currentTarget.style.opacity = '0.7')}
      >
        {isFullscreen ? '\u2715' : '\u26F6'}
      </button>
      <button
        onClick={onExportPNG}
        title="Export chart as PNG"
        aria-label="Export chart as PNG"
        style={{
          position: 'absolute', top: 8, right: 34, zIndex: 10,
          background: 'transparent', border: 'none', color: '#ffffff',
          opacity: 0.7, cursor: 'pointer', fontSize: 16, padding: '2px 4px',
          lineHeight: 1, fontFamily: 'sans-serif',
        }}
        onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
        onMouseLeave={e => (e.currentTarget.style.opacity = '0.7')}
      >
        {'\u{1F4F7}'}
      </button>
      <button
        onClick={onToggleDrawing}
        title={drawingMode ? 'Exit drawing mode' : 'Trendline tool (click start, click end)'}
        aria-label={drawingMode ? 'Exit drawing mode' : 'Enable trendline tool'}
        aria-pressed={drawingMode}
        style={{
          position: 'absolute', top: 8, right: 62, zIndex: 10,
          background: drawingMode ? 'var(--accent)' : 'transparent',
          border: '1px solid var(--border)', borderRadius: 4,
          color: drawingMode ? '#fff' : '#ffffff',
          opacity: 0.7, cursor: 'pointer', fontSize: 13, padding: '2px 6px',
          lineHeight: 1, fontFamily: 'sans-serif',
        }}
      >
        {'\u2194'}
      </button>
      {lineCount > 0 && (
        <button
          onClick={onClearLines}
          title="Clear trendlines"
          style={{
            position: 'absolute', top: 8, right: 92, zIndex: 10,
            background: 'transparent', border: '1px solid var(--border)', borderRadius: 4,
            color: '#ffffff', opacity: 0.7, cursor: 'pointer', fontSize: 11,
            padding: '2px 6px', lineHeight: 1, fontFamily: 'sans-serif',
          }}
        >
          Clear
        </button>
      )}
    </>
  )
}
