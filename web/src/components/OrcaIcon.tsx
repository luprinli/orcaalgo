import type { SVGProps } from 'react'

export default function OrcaIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" fill="none" {...props}>
      {/* Body — inherits color from parent (text-sidebar-primary-foreground) */}
      <ellipse cx="28" cy="38" rx="18" ry="10" fill="currentColor" stroke="currentColor" strokeWidth={1} strokeOpacity={0.5} />
      {/* Belly — subtle opacity for anatomical depth */}
      <ellipse cx="30" cy="43" rx="11" ry="5" fill="currentColor" opacity={0.35} />
      {/* Eye — white patch */}
      <ellipse cx="20" cy="32" rx="3.5" ry="2.5" fill="currentColor" opacity={0.7} />
      {/* Pupil — solid dark dot for visual anchor */}
      <circle cx="18.5" cy="32" r="1.3" fill="#0f172a" opacity={0.85} />
      {/* Dorsal fin */}
      <path d="M34 30 L42 16 L38 27" fill="currentColor" stroke="currentColor" strokeWidth={0.8} strokeOpacity={0.5} />
      {/* Tail flukes */}
      <path d="M10 38 Q4 26 0 28" fill="currentColor" stroke="currentColor" strokeWidth={0.8} strokeOpacity={0.5} />
      <path d="M10 38 Q4 50 0 48" fill="currentColor" stroke="currentColor" strokeWidth={0.8} strokeOpacity={0.5} />
    </svg>
  )
}
