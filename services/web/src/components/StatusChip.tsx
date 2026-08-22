import { Badge } from '@/components/ui/badge';
import type { Status } from '@/tracker/types';

const tones = {
  backlog: 'neutral',
  active: 'warning',
  blocked: 'danger',
  done: 'success',
  canceled: 'neutral',
} as const;

/**
 * A status, coloured by its category.
 *
 * The colour comes from the category and never from the id, because statuses
 * are user-defined and a name can change while what it means does not
 * (ADR-0004).
 */
export function StatusChip({ status }: { status: Status | undefined }) {
  if (!status) {
    return <Badge tone="neutral">unknown</Badge>;
  }
  return <Badge tone={tones[status.category]}>{status.name}</Badge>;
}
