import { useEffect, useMemo, useState } from 'react';

import type { RealtimeEvent } from '@/realtime/types';
import { TrackerAPI } from '@/tracker/api';
import { ITEM_DELETED, itemOf } from '@/tracker/types';
import type { Item, Schema } from '@/tracker/types';

type Tracked = {
  schema: Schema;
  items: Item[];
  loaded: boolean;
  error: string | null;
};

/**
 * A project's tracker, kept current without polling.
 *
 * The schema and the item list are fetched once per `generation` — when the
 * rooms are joined, and after any resync — and thereafter maintained from
 * events.
 *
 * A tracker event names the item it concerns but does not carry it: the feed
 * is an activity record rather than a copy of the data (ADR-0008). So hearing
 * about an item means fetching that one item, which is a targeted read rather
 * than a poll — nothing happens while nothing changes. A deletion needs no
 * fetch, because the kind already says everything there is to know.
 */
export function useTracker(
  apiUrl: string,
  project: string | null,
  generation: number,
  events: RealtimeEvent[]
): Tracked {
  const api = useMemo(
    () => (project ? new TrackerAPI(apiUrl, project) : null),
    [apiUrl, project]
  );
  const [schema, setSchema] = useState<Schema>({ types: [] });
  const [items, setItems] = useState<Item[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoaded(false);
    setItems([]);
    if (!api || generation === 0) return; // not joined yet; a fetch now could miss a change

    const stop = new AbortController();
    const load = async () => {
      try {
        const [types, listed] = await Promise.all([
          api.schema(stop.signal),
          api.items(stop.signal),
        ]);
        setSchema(types);
        setItems(listed);
        setError(null);
        setLoaded(true);
      } catch (cause) {
        if (stop.signal.aborted) return;
        setError(cause instanceof Error ? cause.message : 'fetch failed');
      }
    };
    void load();
    return () => stop.abort();
  }, [api, generation]);

  // Only the newest event is applied. Re-reading the whole buffer on every
  // render would replay history, and history is what this design does not keep.
  const latest = events[0];
  useEffect(() => {
    if (!api || !latest) return;
    const id = itemOf(latest.kind, latest.data);
    if (!id) return;

    if (latest.kind === ITEM_DELETED) {
      setItems((current) => current.filter((i) => i.id !== id));
      return;
    }
    const stop = new AbortController();
    void api
      .item(id, stop.signal)
      .then((fetched) => {
        if (fetched) setItems((current) => upsert(current, fetched));
      })
      .catch(() => undefined); // a failed read leaves the list as it was
    return () => stop.abort();
  }, [api, latest]);

  return { schema, items, loaded, error };
}

/**
 * Replaces an item or appends it, keeping creation order.
 *
 * Applying the same item twice must be harmless: a fetch and an event will
 * describe the same state, and both arriving is ordinary (ADR-0008).
 */
function upsert(current: Item[], item: Item): Item[] {
  const at = current.findIndex((i) => i.id === item.id);
  if (at < 0) return [...current, item];
  const next = [...current];
  next[at] = item;
  return next;
}
