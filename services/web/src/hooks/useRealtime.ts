import { useEffect, useMemo, useState } from 'react';

import { RealtimeClient } from '@/realtime/client';
import type { RealtimeEvent, Status } from '@/realtime/types';
import { socketUrl } from '@/realtime/url';

/** How many events the live view keeps. Enough to see, small enough to render. */
const KEEP = 50;

type Live = {
  /** The connection itself, for sending commands over. */
  client: RealtimeClient;
  status: Status;
  events: RealtimeEvent[];
  /**
   * Increments whenever the caller should fetch current state: once the
   * rooms are joined, and again after any resync.
   *
   * This is the whole synchronisation model in one number (ADR-0008). There
   * is no replay and no resume: a consumer refetches what it is showing when
   * this changes, and otherwise waits. It rises only on join and on resync,
   * so using it as an effect dependency cannot become a poll.
   */
  generation: number;
};

/**
 * Subscribes to rooms for as long as the component is mounted.
 *
 * The client lives in a memo rather than state because replacing it would
 * drop the socket; rooms are joined in their own effect so changing the list
 * re-subscribes without reconnecting.
 */
export function useRealtime(apiUrl: string, rooms: string[]): Live {
  const client = useMemo(() => new RealtimeClient(socketUrl(apiUrl)), [apiUrl]);
  const [status, setStatus] = useState<Status>('closed');
  const [events, setEvents] = useState<RealtimeEvent[]>([]);
  const [generation, setGeneration] = useState(0);

  useEffect(() => {
    const bump = () => setGeneration((n) => n + 1);
    const stop = client.listen({
      onStatus: setStatus,
      onEvent: (event) => setEvents((seen) => [event, ...seen].slice(0, KEEP)),
      // Ready means every room is joined, so nothing that happens from here
      // can be missed by both the fetch and the socket.
      onReady: bump,
      onResync: bump,
    });
    client.start();
    return () => {
      stop();
      client.stop();
    };
  }, [client]);

  const key = rooms.join(',');
  useEffect(() => {
    const joined = key ? key.split(',') : [];
    joined.forEach((room) => client.join(room));
    return () => joined.forEach((room) => client.leave(room));
  }, [client, key]);

  return { client, status, events, generation };
}
