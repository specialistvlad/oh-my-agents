import { Activity, ListTree } from 'lucide-react';

import { cn } from '@/lib/cn';
import type { PanelID } from '@/workspace/layout';

const panels = [
  { id: 'objects' as const, label: 'Objects', Icon: ListTree },
  { id: 'activity' as const, label: 'Activity', Icon: Activity },
];

type Props = {
  panel: PanelID;
  collapsed: boolean;
  onSelect: (panel: PanelID) => void;
};

/**
 * The icon rail: which panel the left column shows.
 *
 * It exists so later panels arrive without another decision, and it is what
 * brings a collapsed left column back — which is why collapsing to zero is
 * safe (ADR-0011). Clicking the panel already showing collapses it, the way
 * an IDE sidebar behaves.
 */
export function Rail({ panel, collapsed, onSelect }: Props) {
  return (
    <nav
      aria-label="Panels"
      className="flex w-12 shrink-0 flex-col items-center gap-1 border-r border-border py-2">
      {panels.map(({ id, label, Icon }) => {
        const showing = !collapsed && panel === id;
        return (
          <button
            key={id}
            type="button"
            title={label}
            aria-label={label}
            aria-pressed={showing}
            className={cn(
              'flex size-9 items-center justify-center rounded-app transition-colors',
              'outline-none focus-visible:ring-2 focus-visible:ring-primary/50',
              showing
                ? 'bg-muted text-foreground'
                : 'text-muted-foreground hover:bg-muted/60'
            )}
            onClick={() => onSelect(id)}>
            <Icon className="size-5" aria-hidden />
          </button>
        );
      })}
    </nav>
  );
}
