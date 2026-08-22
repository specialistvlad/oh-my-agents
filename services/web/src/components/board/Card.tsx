import { cn } from '@/lib/cn';
import type { Item } from '@/tracker/types';

type Props = {
  item: Item;
  selected: boolean;
  dragging: boolean;
  onSelect: (item: Item) => void;
  onOpen: (item: Item) => void;
  onDragStart: (item: Item) => void;
  onDragEnd: () => void;
};

/** One card. A click selects it, a double click opens it (ADR-0011). */
export function Card({
  item,
  selected,
  dragging,
  onSelect,
  onOpen,
  onDragStart,
  onDragEnd,
}: Props) {
  return (
    <li
      draggable
      className={cn(
        'cursor-grab rounded-app border border-border bg-surface px-3 py-2 text-sm',
        'hover:border-primary/40 active:cursor-grabbing',
        selected && 'ring-2 ring-primary/40',
        dragging && 'opacity-40'
      )}
      onDragStart={() => onDragStart(item)}
      onDragEnd={onDragEnd}
      onClick={() => onSelect(item)}
      onDoubleClick={() => onOpen(item)}>
      {item.title}
    </li>
  );
}
