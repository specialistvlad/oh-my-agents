import { canMove } from './types';
import type { Item, ItemType, Status } from './types';

/** One column: a status, and the cards sitting in it, in rank order. */
export type Column = {
  status: Status;
  items: Item[];
};

/**
 * A board is a view of one type's workflow (ADR-0017).
 *
 * Columns come from the type's statuses in the order the type declares them,
 * including the ones nobody is in — an empty column is where work goes next,
 * and hiding it hides the workflow.
 *
 * Items are assumed already in rank order, which is what `?sort=rank` returns.
 * Grouping preserves that, so a column's order falls out of the project's
 * order rather than being maintained separately (ADR-0013).
 */
export function columnsOf(type: ItemType | undefined, items: Item[]): Column[] {
  if (!type) return [];
  const mine = items.filter((i) => i.type === type.id);
  return type.statuses.map((status) => ({
    status,
    items: mine.filter((i) => i.status === status.id),
  }));
}

/**
 * Whether a card may be dropped in a column.
 *
 * Legality is per **card**, not per type: what a card can reach depends on the
 * status it is in now. A board that offered every column would offer moves
 * that always fail, and a dead control teaches people not to trust the rest of
 * the interface (ADR-0017).
 *
 * A card's own column is always a legal target, because dropping there is a
 * reorder rather than a transition.
 */
export function canDrop(type: ItemType, item: Item, status: string): boolean {
  return status === item.status || canMove(type, item.status, status);
}

/**
 * What a drop should do: change the status, change the position, or both.
 *
 * Returned as a description rather than performed, so the decision is testable
 * without a socket and the component that renders a board does not also decide
 * what dropping means.
 */
export type Drop = {
  /** Absent when the card is already in this column. */
  status?: string;
  /** The card this one lands after, if any. */
  after?: string;
  /** The card this one lands before, if any. */
  before?: string;
};

/**
 * Reads a drop into the two neighbours it lands between.
 *
 * `at` is the index within the target column's cards that the card is dropped
 * at. The card being moved is excluded first, so dragging within a column does
 * not treat the card as its own neighbour — which would ask the store to rank
 * something between itself and something else.
 */
export function dropAt(column: Column, item: Item, at: number): Drop {
  const others = column.items.filter((i) => i.id !== item.id);
  const index = Math.max(0, Math.min(at, others.length));

  const drop: Drop = {};
  if (column.status.id !== item.status) drop.status = column.status.id;
  if (index > 0) drop.after = others[index - 1].id;
  if (index < others.length) drop.before = others[index].id;
  return drop;
}

/** Whether a drop would change anything at all. */
export function isNoop(drop: Drop, column: Column, item: Item): boolean {
  if (drop.status) return false;
  const at = column.items.findIndex((i) => i.id === item.id);
  if (at < 0) return false;
  const after = at > 0 ? column.items[at - 1].id : undefined;
  const before =
    at < column.items.length - 1 ? column.items[at + 1].id : undefined;
  return drop.after === after && drop.before === before;
}
