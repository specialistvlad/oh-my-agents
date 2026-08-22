import { useCallback, useState } from 'react';

import {
  createProject,
  removeProject,
  renameProject,
} from '@/projects/commands';
import type { RealtimeClient } from '@/realtime/client';

/**
 * Project changes over the socket.
 *
 * Nothing here updates local state on success. The change comes back as an
 * event like everybody else's, so this client sees its own writes exactly the
 * way another tab does — one path to be correct instead of two.
 */
export function useProjectWriter(client: RealtimeClient) {
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

  return {
    busy,
    error,
    create: useCallback(
      (name: string) => run(() => createProject(client, name)),
      [client, run]
    ),
    rename: useCallback(
      (id: string, name: string) => run(() => renameProject(client, id, name)),
      [client, run]
    ),
    remove: useCallback(
      (id: string) => run(() => removeProject(client, id)),
      [client, run]
    ),
  };
}
