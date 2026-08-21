import { useEffect, useMemo, useState } from 'react';

import { RealtimeClient, socketUrl } from '@/realtime/client';
import type { RealtimeEvent, Status } from '@/realtime/types';

/** How many events the live view keeps. Enough to see, small enough to render. */
const KEEP = 50;

type Live = {
  status: Status;
  events: RealtimeEvent[];
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

  useEffect(() => {
    const stop = client.listen({
      onStatus: setStatus,
      onEvent: (event) => setEvents((seen) => [event, ...seen].slice(0, KEEP)),
      // A resync means messages were missed. There is nothing to re-read
      // from yet, so the honest thing is to show it rather than pretend.
      onResync: () =>
        setEvents((seen) =>
          [{ room: '', seq: 0, kind: 'resync', data: null }, ...seen].slice(
            0,
            KEEP
          )
        ),
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

  return { status, events };
}
