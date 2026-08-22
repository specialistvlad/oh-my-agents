import { Fragment } from 'react';

import { Card } from '@/components/board/Card';
import { cn } from '@/lib/cn';
import type { Column as ColumnModel } from '@/tracker/board';
import type { Item } from '@/tracker/types';

type Props = {
  column: ColumnModel;
  dragging: Item | null;
  /** Whether the card being dragged may land here at all. */
  droppable: boolean;
  /** Where in this column the card would land, or null when not over it. */
  at: number | null;
  selected: string | null;
  onSelect: (item: Item) => void;
  onOpen: (item: Item) => void;
  onDragStart: (item: Item) => void;
  onDragEnd: () => void;
  onOver: (at: number) => void;
  onLeave: () => void;
  onDrop: () => void;
};

/**
 * One status and its cards.
 *
 * A column the dragged card cannot reach is visibly not a target and accepts
 * nothing: ADR-0017 chose never rendering an illegal drop over refusing one,
 * because a control that always fails is worse than no control.
 */
export function Column({
  column,
  dragging,
  droppable,
  at,
  selected,
  onSelect,
  onOpen,
  onDragStart,
  onDragEnd,
  onOver,
  onLeave,
  onDrop,
}: Props) {
  const barred = dragging !== null && !droppable;
  return (
    <section
      aria-label={column.status.name}
      className={cn(
        'flex w-64 shrink-0 flex-col rounded-app bg-muted/40 p-2',
        barred && 'opacity-40',
        droppable && at !== null && 'ring-2 ring-primary/50'
      )}
      onDragOver={(e) => {
        if (!droppable) return;
        e.preventDefault(); // without this the browser refuses the drop
        onOver(indexFrom(e.currentTarget, e.clientY));
      }}
      onDragLeave={onLeave}
      onDrop={(e) => {
        if (!droppable) return;
        e.preventDefault();
        onDrop();
      }}>
      <header className="flex items-center gap-2 px-1 pb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        {column.status.name}
        <span className="ml-auto">{column.items.length}</span>
      </header>
      <ul className="flex flex-1 flex-col gap-2">
        {column.items.map((item, i) => (
          <Fragment key={item.id}>
            {at === i ? <Gap /> : null}
            <Card
              item={item}
              selected={item.id === selected}
              dragging={dragging?.id === item.id}
              onSelect={onSelect}
              onOpen={onOpen}
              onDragStart={onDragStart}
              onDragEnd={onDragEnd}
            />
          </Fragment>
        ))}
        {at !== null && at >= column.items.length ? <Gap /> : null}
      </ul>
    </section>
  );
}

/** Where the card would land, drawn so the drop is predictable before it happens. */
function Gap() {
  return <li aria-hidden className="h-0.5 rounded bg-primary" />;
}

/**
 * Which slot a pointer at this height is over.
 *
 * Read from the rendered cards rather than tracked per card, because a card
 * being dragged is still in the list and its own hover events are unreliable
 * mid-drag.
 */
function indexFrom(column: HTMLElement, y: number): number {
  const cards = [...column.querySelectorAll('li:not([aria-hidden])')];
  for (const [i, card] of cards.entries()) {
    const box = card.getBoundingClientRect();
    if (y < box.top + box.height / 2) return i;
  }
  return cards.length;
}
