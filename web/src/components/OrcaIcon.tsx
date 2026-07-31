import type { SVGProps } from 'react'

export default function OrcaIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" fill="none" {...props}>
      <defs>
        <linearGradient id="o-g" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#3b82f6"/>
          <stop offset="100%" stopColor="#1d4ed8"/>
        </linearGradient>
      </defs>
      <ellipse cx="28" cy="38" rx="18" ry="10" fill="url(#o-g)" stroke="#1e40af" strokeWidth="1"/>
      <ellipse cx="30" cy="43" rx="11" ry="5" fill="#f0f9ff" opacity="0.9"/>
      <ellipse cx="20" cy="32" rx="3.5" ry="2.5" fill="#f0f9ff"/>
      <circle cx="18.5" cy="32" r="1.3" fill="#0f172a"/>
      <path d="M34 30 L42 16 L38 27" fill="#2563eb" stroke="#1e40af" strokeWidth="0.8"/>
      <path d="M10 38 Q4 26 0 28" fill="url(#o-g)" stroke="#1e40af" strokeWidth="0.8"/>
      <path d="M10 38 Q4 50 0 48" fill="url(#o-g)" stroke="#1e40af" strokeWidth="0.8"/>
    </svg>
  )
}
