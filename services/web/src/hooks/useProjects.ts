import { useCallback, useEffect, useState } from 'react';

import { reduce, sort } from '@/projects/reduce';
import type { Project } from '@/projects/types';
import type { RealtimeEvent } from '@/realtime/types';

/**
 * The projects that exist, kept current without polling.
 *
 * Fetched once per `generation` — when the rooms are joined, and after any
 * resync — and thereafter maintained from events. That is the whole model
 * (ADR-0008): fetch what you need on connect, then wait.
 */
export function useProjects(
  apiUrl: string,
  generation: number,
  events: RealtimeEvent[]
) {
  const [projects, setProjects] = useState<Project[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (generation === 0) return; // not joined yet; fetching now could miss a change
    let live = true;
    const load = async () => {
      try {
        const response = await fetch(`${apiUrl}/projects/`);
        if (!response.ok)
          throw new Error(`${response.status} ${response.statusText}`);
        const body = (await response.json()) as { projects?: Project[] };
        if (live) {
          setProjects(sort(body.projects ?? []));
          setError(null);
          setLoaded(true);
        }
      } catch (cause) {
        if (live)
          setError(cause instanceof Error ? cause.message : 'fetch failed');
      }
    };
    void load();
    return () => {
      live = false;
    };
  }, [apiUrl, generation]);

  // Only the newest event is applied. Re-reducing the whole buffer on every
  // render would replay history, and history is exactly what this design does
  // not keep.
  const latest = events[0];
  useEffect(() => {
    if (!latest) return;
    setProjects((current) => reduce(current, latest.kind, latest.data));
  }, [latest]);

  const clearError = useCallback(() => setError(null), []);
  return { projects, error, loaded, clearError };
}
