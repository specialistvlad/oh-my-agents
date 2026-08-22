import { Column } from '@/components/board/Column';
import type { Drag } from '@/hooks/useBoardDrag';
import { canDrop, columnsOf, dropAt, isNoop } from '@/tracker/board';
import type { Drop } from '@/tracker/board';
import type { Item, ItemType, Schema } from '@/tracker/types';

type Props = {
  schema: Schema;
  items: Item[];
  loaded: boolean;
  selected: string | null;
  drag: Drag;
  onSelect: (item: Item) => void;
  onOpen: (item: Item) => void;
  onDrop: (item: Item, drop: Drop) => void;
};

/**
 * A board is one type's workflow: the columns are its statuses and the moves
 * between them are the ones it declares (ADR-0017).
 *
 * The first type is shown. A picker arrives when a project has a second one —
 * a control with one option is not a control.
 */
export function Board({
  schema,
  items,
  loaded,
  selected,
  drag,
  onSelect,
  onOpen,
  onDrop,
}: Props) {
  const type: ItemType | undefined = schema.types[0];
  const columns = columnsOf(type, items);

  if (!loaded) {
    return <p className="text-sm text-muted-foreground">Loading…</p>;
  }
  if (!type) {
    return (
      <p className="text-sm text-muted-foreground">
        This project has no types.
      </p>
    );
  }

  return (
    <div className="flex h-full gap-3 overflow-x-auto pb-2">
      {columns.map((column) => {
        const droppable = drag.item
          ? canDrop(type, drag.item, column.status.id)
          : false;
        const at = drag.over?.status === column.status.id ? drag.over.at : null;
        return (
          <Column
            key={column.status.id}
            column={column}
            dragging={drag.item}
            droppable={droppable}
            at={at}
            selected={selected}
            onSelect={onSelect}
            onOpen={onOpen}
            onDragStart={drag.start}
            onDragEnd={drag.end}
            onOver={(index) => drag.hover(column.status.id, index)}
            onLeave={() => drag.leave(column.status.id)}
            onDrop={() => {
              const moved = drag.item;
              drag.end();
              if (!moved || at === null) return;
              const drop = dropAt(column, moved, at);
              // A card dropped where it already was is not a write.
              if (!isNoop(drop, column, moved)) onDrop(moved, drop);
            }}
          />
        );
      })}
    </div>
  );
}
