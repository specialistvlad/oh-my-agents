import { useSplitter } from '@/hooks/useSplitter';
import { cn } from '@/lib/cn';

type Props = {
  width: number;
  side: 'left' | 'right';
  label: string;
  onResize: (width: number) => void;
};

/**
 * The handle between two columns.
 *
 * It carries the separator role and its value, and is focusable, so the width
 * is reachable without a pointer. The hit area is wider than the visible line,
 * because a one-pixel target is a target nobody hits.
 */
export function Splitter({ width, side, label, onResize }: Props) {
  const drag = useSplitter({ width, side, onResize });
  return (
    <div
      role="separator"
      tabIndex={0}
      aria-label={label}
      aria-orientation="vertical"
      aria-valuenow={width}
      className={cn(
        'group relative w-1 shrink-0 cursor-col-resize outline-none',
        'before:absolute before:inset-y-0 before:-left-1 before:-right-1 before:content-[""]'
      )}
      onPointerDown={drag.onPointerDown}
      onPointerMove={drag.onPointerMove}
      onPointerUp={drag.onPointerUp}
      onKeyDown={drag.onKeyDown}>
      <div
        className={cn(
          'h-full w-px bg-border transition-colors',
          'group-hover:bg-primary group-focus-visible:bg-primary',
          drag.dragging && 'bg-primary'
        )}
      />
    </div>
  );
}
