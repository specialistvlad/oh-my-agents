/** Mirrors services/api/internal/tracker. */

/**
 * A field's value, mirroring value_json.go's envelope.
 *
 * Kind and value always agree, because the api will not build one where they
 * do not — a stored value whose payload contradicts its kind is refused on
 * load rather than becoming a value whose kind lies.
 */
export type Value = {
  kind: FieldKind;
  value: unknown;
};

export type FieldKind =
  | 'text'
  | 'markdown'
  | 'number'
  | 'bool'
  | 'date'
  | 'duration'
  | 'select'
  | 'multi_select'
  | 'actor'
  | 'item'
  | 'url';

export type Actor = {
  kind: 'human' | 'agent' | 'system';
  id: string;
};

export type Item = {
  id: string;
  type: string;
  title: string;
  body: string;
  status: string;
  parent: string | null;
  /** Values for the fields this item's type declares, keyed by field id. */
  fields: Record<string, Value>;
  /**
   * Where this item sits in the project's order (ADR-0013).
   *
   * Opaque: it sorts, and nothing may read anything else into it. The server
   * mints it, and a client names neighbors rather than keys.
   */
  rank: string;
  version: number;
  created_at: string;
  updated_at: string;
};

export type StatusCategory =
  'backlog' | 'active' | 'blocked' | 'done' | 'canceled';

export type Status = {
  id: string;
  name: string;
  category: StatusCategory;
};

export type Transition = { from: string; to: string };

export type Option = {
  id: string;
  name: string;
};

export type FieldDef = {
  id: string;
  name: string;
  description?: string;
  kind: FieldKind;
  required?: boolean;
  options?: Option[];
};

export type ItemType = {
  id: string;
  name: string;
  fields?: FieldDef[];
  statuses: Status[];
  initial: string;
  transitions: Transition[];
};

export type Schema = { types: ItemType[] };

/**
 * The event kinds the api publishes for tracker activity.
 *
 * Unlike a project event, these carry the item's **id** and not its contents:
 * the feed is an activity record, not a copy of the data (ADR-0008). So a
 * client hearing one fetches that item, except for a deletion, where the kind
 * already says everything there is to know.
 */
export const ITEM_DELETED = 'item_deleted';

const ITEM_EVENTS = new Set([
  'item_created',
  'item_updated',
  'status_changed',
  'parent_changed',
  ITEM_DELETED,
]);

/** The item an event concerns, or null if it concerns none. */
export function itemOf(kind: string, data: unknown): string | null {
  if (!ITEM_EVENTS.has(kind)) return null;
  const { item } = (data ?? {}) as { item?: string };
  return item ?? null;
}

/** Whether a transition is one the type declares. */
export function canMove(type: ItemType, from: string, to: string): boolean {
  return type.transitions.some((t) => t.from === from && t.to === to);
}

/** Where an item can go from where it is. */
export function movesFrom(type: ItemType, from: string): Status[] {
  return type.statuses.filter(
    (s) => s.id !== from && canMove(type, from, s.id)
  );
}

export function statusOf(
  type: ItemType | undefined,
  id: string
): Status | undefined {
  return type?.statuses.find((s) => s.id === id);
}

/** The fields a type declares, paired with whatever the item holds for each. */
export function fieldsOf(
  type: ItemType | undefined,
  item: Item
): Array<{ def: FieldDef; value: Value | undefined }> {
  return (type?.fields ?? []).map((def) => ({
    def,
    value: item.fields?.[def.id],
  }));
}
