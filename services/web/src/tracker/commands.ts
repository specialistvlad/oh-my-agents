import type { RealtimeClient } from '@/realtime/client';

/**
 * Tracker writes over the socket.
 *
 * A write goes out the same connection its resulting event comes back on, so
 * this client sees its own change exactly the way another tab does — one path
 * to be correct instead of two.
 */
export function createItem(
  client: RealtimeClient,
  project: string,
  type: string,
  title: string
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'item.create',
    id,
    project,
    idempotency,
    body: { type, title },
  }));
}

/**
 * Moves an item, stating the version it expects to replace.
 *
 * A stale version is refused rather than applied, which is what stops two
 * people moving the same item at once from silently losing one of the moves.
 */
export function moveItem(
  client: RealtimeClient,
  project: string,
  item: string,
  version: number,
  status: string
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'item.update',
    id,
    project,
    item,
    version,
    idempotency,
    body: { status },
  }));
}

export function deleteItem(
  client: RealtimeClient,
  project: string,
  item: string,
  version: number
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'item.delete',
    id,
    project,
    item,
    version,
    idempotency,
  }));
}
