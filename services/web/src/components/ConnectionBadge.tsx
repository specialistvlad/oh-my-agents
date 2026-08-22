import { Badge } from '@/components/ui/badge';
import type { Status } from '@/realtime/types';

type Look = {
  label: string;
  tone: 'success' | 'warning' | 'neutral';
  dot: string;
};

const look: Record<Status, Look> = {
  open: { label: 'live', tone: 'success', dot: 'bg-success' },
  connecting: {
    label: 'connecting',
    tone: 'warning',
    dot: 'bg-warning animate-pulse',
  },
  reconnecting: {
    label: 'reconnecting',
    tone: 'warning',
    dot: 'bg-warning animate-pulse',
  },
  closed: { label: 'offline', tone: 'neutral', dot: 'bg-muted-foreground' },
};

/** Shows whether the socket is actually carrying anything. */
export function ConnectionBadge({ status }: { status: Status }) {
  const { label, tone, dot } = look[status];
  return (
    <Badge tone={tone}>
      {/* Decorative: the label beside it already says this. */}
      <span aria-hidden className={`size-1.5 rounded-full ${dot}`} />
      {label}
    </Badge>
  );
}
