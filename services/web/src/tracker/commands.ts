import type { RealtimeClient } from '@/realtime/client';

import type { Actor } from './types';

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
  title: string,
  author: Actor
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'item.create',
    id,
    project,
    idempotency,
    body: { type, title, author },
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
  status: string,
  author: Actor
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'item.update',
    id,
    project,
    item,
    version,
    idempotency,
    body: { status, author },
  }));
}

/**
 * Moves an item between two neighbors.
 *
 * It states no version and expects no reply body: a drag is a position rather
 * than a change to one, so there is nothing to compare against (ADR-0013).
 */
export function reorderItem(
  client: RealtimeClient,
  project: string,
  item: string,
  after: string | undefined,
  before: string | undefined
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'item.reorder',
    id,
    project,
    item,
    version: 0,
    idempotency,
    body: { after, before },
  }));
}

export function deleteItem(
  client: RealtimeClient,
  project: string,
  item: string,
  version: number,
  author: Actor
): Promise<void> {
  return client.command((id, idempotency) => ({
    type: 'item.delete',
    id,
    project,
    item,
    version,
    idempotency,
    body: { author },
  }));
}
