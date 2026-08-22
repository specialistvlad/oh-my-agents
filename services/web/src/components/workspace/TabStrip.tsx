import { X } from 'lucide-react';

import { cn } from '@/lib/cn';
import type { Tab } from '@/workspace/tabs';

type Props = {
  tabs: Tab[];
  active: string | null;
  onFocus: (id: string) => void;
  onClose: (id: string) => void;
};

/**
 * The open tabs.
 *
 * A tab whose object is gone stays and says so, struck through rather than
 * removed: vanishing mid-sentence looks like a bug, loses what was on screen,
 * and slides every neighbour under the cursor (ADR-0011).
 */
export function TabStrip({ tabs, active, onFocus, onClose }: Props) {
  if (tabs.length === 0) return null;
  return (
    <div role="tablist" className="flex overflow-x-auto border-b border-border">
      {tabs.map((tab) => (
        <div
          key={tab.id}
          className={cn(
            'flex shrink-0 items-center gap-2 border-r border-border px-3 py-2 text-sm',
            tab.id === active
              ? 'bg-surface font-medium'
              : 'text-muted-foreground'
          )}>
          <button
            type="button"
            role="tab"
            aria-selected={tab.id === active}
            className={cn('outline-none', tab.gone && 'line-through')}
            title={tab.gone ? `${tab.title} — deleted` : tab.title}
            onClick={() => onFocus(tab.id)}>
            {tab.title}
          </button>
          <button
            type="button"
            aria-label={`Close ${tab.title}`}
            className="text-muted-foreground hover:text-foreground"
            onClick={() => onClose(tab.id)}>
            <X className="size-3.5" aria-hidden />
          </button>
        </div>
      ))}
    </div>
  );
}
