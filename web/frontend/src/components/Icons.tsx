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

export const Search = (props: SVGProps<SVGSVGElement>) => (
  <svg width="13" height="13" viewBox="0 0 14 14" fill="none" {...props}>
    <circle cx="6" cy="6" r="4" stroke="currentColor" strokeWidth="1.3" />
    <path
      d="M9 9l3 3"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
    />
  </svg>
);

export const Close = (props: SVGProps<SVGSVGElement>) => (
  <svg width="12" height="12" viewBox="0 0 14 14" fill="none" {...props}>
    <path
      d="M3 3l8 8M11 3l-8 8"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
    />
  </svg>
);

export const ChevronRight = (props: SVGProps<SVGSVGElement>) => (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...props}>
    <path
      d="M5 3l4 4-4 4"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export const ChevronDown = (props: SVGProps<SVGSVGElement>) => (
  <svg width="12" height="12" viewBox="0 0 14 14" fill="none" {...props}>
    <path
      d="M3 5l4 4 4-4"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export const Filter = (props: SVGProps<SVGSVGElement>) => (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...props}>
    <path
      d="M2 3h10M4 7h6M5.5 11h3"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
    />
  </svg>
);

export const Branch = (props: SVGProps<SVGSVGElement>) => (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...props}>
    <circle cx="3.5" cy="3" r="1.4" stroke="currentColor" strokeWidth="1.2" />
    <circle cx="3.5" cy="11" r="1.4" stroke="currentColor" strokeWidth="1.2" />
    <circle cx="10.5" cy="7" r="1.4" stroke="currentColor" strokeWidth="1.2" />
    <path
      d="M3.5 4.5v5M3.5 9.5C3.5 8 6 8.5 6 7c0-1.5 2.5-1 3-1"
      stroke="currentColor"
      strokeWidth="1.2"
      strokeLinecap="round"
    />
  </svg>
);

export const DiffIcon = (props: SVGProps<SVGSVGElement>) => (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" {...props}>
    <path
      d="M5 3v8M9 3v8M3 7h8"
      stroke="currentColor"
      strokeWidth="1.3"
      strokeLinecap="round"
    />
  </svg>
);
