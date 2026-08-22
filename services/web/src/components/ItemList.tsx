import { ItemRow } from '@/components/ItemRow';
import type { Item, Schema } from '@/tracker/types';

type Props = {
  items: Item[];
  schema: Schema;
  loaded: boolean;
  busy: boolean;
  onMove: (item: Item, status: string) => void;
  onRemove: (item: Item) => void;
};

export function ItemList({
  items,
  schema,
  loaded,
  busy,
  onMove,
  onRemove,
}: Props) {
  if (!loaded) {
    return <p className="py-3 text-sm text-muted-foreground">Loading…</p>;
  }
  if (items.length === 0) {
    return (
      <p className="py-3 text-sm text-muted-foreground">
        Nothing tracked yet. Add something — then open a second tab and watch
        both stay in step.
      </p>
    );
  }
  return (
    <ul>
      {items.map((item) => (
        <ItemRow
          key={item.id}
          item={item}
          type={schema.types.find((t) => t.id === item.type)}
          busy={busy}
          onMove={onMove}
          onRemove={onRemove}
        />
      ))}
    </ul>
  );
}
