import type { SVGProps } from 'react';

export const ArrowRight = (props: SVGProps<SVGSVGElement>) => (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...props}>
    <path
      d="M3 7h8M7.5 3.5L11 7l-3.5 3.5"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export const BrandMark = ({ size = 22 }: { size?: number }) => (
  <svg width={size} height={size} viewBox="0 0 22 22" fill="none">
    <circle cx="4" cy="6" r="2.2" stroke="#22d3ee" strokeWidth="1.4" />
    <circle cx="4" cy="16" r="2.2" stroke="#5b6068" strokeWidth="1.4" />
    <circle cx="17" cy="11" r="2.2" fill="#22d3ee" />
    <path d="M6 6.6 L15 10.4" stroke="#22d3ee" strokeWidth="1.2" />
    <path d="M6 15.4 L15 11.6" stroke="#5b6068" strokeWidth="1.2" />
  </svg>
);
