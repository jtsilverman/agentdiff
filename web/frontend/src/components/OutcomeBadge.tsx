export type OutcomeKind = 'ok' | 'warn' | 'bad' | 'novel' | 'neutral' | 'accent';

export default function OutcomeBadge({
  kind,
  label,
}: {
  kind: OutcomeKind;
  label: string;
}) {
  return (
    <span className={`ad-badge ${kind}`}>
      <span className="dot" />
      {label}
    </span>
  );
}
