import { useCallback, useState } from 'react';

import type { RealtimeClient } from '@/realtime/client';
import { createItem, deleteItem, moveItem } from '@/tracker/commands';

/**
 * Tracker writes for the current project.
 *
 * Nothing here updates local state on success: the change comes back as an
 * event like everybody else's, so this client sees its own writes the way
 * another tab does.
 */
export function useTrackerWriter(
  client: RealtimeClient,
  project: string | null
) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = useCallback(
    async (work: (project: string) => Promise<void>) => {
      if (!project) return;
      setBusy(true);
      setError(null);
      try {
        await work(project);
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'command failed');
      } finally {
        setBusy(false);
      }
    },
    [project]
  );

  return {
    busy,
    error,
    create: useCallback(
      (type: string, title: string) =>
        run((p) => createItem(client, p, type, title)),
      [client, run]
    ),
    move: useCallback(
      (item: string, version: number, status: string) =>
        run((p) => moveItem(client, p, item, version, status)),
      [client, run]
    ),
    remove: useCallback(
      (item: string, version: number) =>
        run((p) => deleteItem(client, p, item, version)),
      [client, run]
    ),
  };
}
