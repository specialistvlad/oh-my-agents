import Chip from '@mui/material/Chip';

import type { Status } from '@/realtime/types';

const look: Record<
  Status,
  { label: string; color: 'success' | 'warning' | 'default' }
> = {
  open: { label: 'live', color: 'success' },
  connecting: { label: 'connecting', color: 'warning' },
  reconnecting: { label: 'reconnecting', color: 'warning' },
  closed: { label: 'offline', color: 'default' },
};

/** Shows whether the socket is actually carrying anything. */
export function ConnectionBadge({ status }: { status: Status }) {
  const { label, color } = look[status];
  return <Chip size="small" color={color} label={label} variant="outlined" />;
}
