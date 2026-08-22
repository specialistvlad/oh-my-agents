import { Trash2 } from 'lucide-react';

import { StatusChip } from '@/components/StatusChip';
import { Button } from '@/components/ui/button';
import { movesFrom, statusOf } from '@/tracker/types';
import type { Item, ItemType } from '@/tracker/types';

type Props = {
  item: Item;
  type: ItemType | undefined;
  busy: boolean;
  onMove: (item: Item, status: string) => void;
  onRemove: (item: Item) => void;
};

/**
 * One item, with the moves its workflow actually allows.
 *
 * Offering every status would produce buttons that always fail, so the type
 * decides what is on screen.
 */
export function ItemRow({ item, type, busy, onMove, onRemove }: Props) {
  const moves = type ? movesFrom(type, item.status) : [];
  return (
    <li className="flex items-center gap-3 border-b border-border py-3 last:border-0">
      <StatusChip status={statusOf(type, item.status)} />
      <span className="min-w-0 flex-1 truncate">{item.title}</span>
      {moves.map((status) => (
        <Button
          key={status.id}
          size="sm"
          variant="outline"
          disabled={busy}
          onClick={() => onMove(item, status.id)}>
          {status.name}
        </Button>
      ))}
      <Button
        size="sm"
        variant="ghost"
        disabled={busy}
        aria-label={`Delete ${item.title}`}
        onClick={() => onRemove(item)}>
        <Trash2 className="size-4 text-danger" aria-hidden />
      </Button>
    </li>
  );
}
