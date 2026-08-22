import { useCallback, useState } from 'react';

import type { RealtimeClient } from '@/realtime/client';
import { deleteSetting, setSetting } from '@/realtime/commands';

/**
 * Writes a setting over the socket.
 *
 * One connection carries the write out and the resulting event back, so the
 * page needs no HTTP request at all once it is connected — the only fetch
 * left is the initial state. A failure here is a real answer from the
 * server, not a guess: the command resolves when its own reply arrives.
 */
export function useRealtimeWriter(client: RealtimeClient) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(async (work: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await work();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'command failed');
    } finally {
      setBusy(false);
    }
  }, []);

  const write = useCallback(
    (key: string, value: unknown) => run(() => setSetting(client, key, value)),
    [client, run]
  );
  const remove = useCallback(
    (key: string) => run(() => deleteSetting(client, key)),
    [client, run]
  );

  return { write, remove, busy, error };
}
