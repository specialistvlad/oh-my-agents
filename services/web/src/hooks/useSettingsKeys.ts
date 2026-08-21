import { useEffect, useState } from 'react';

/**
 * The settings that currently exist.
 *
 * Fetched once per `generation` — that is, when the rooms are joined and
 * after any resync — and never again. Everything between those points comes
 * from the socket, which is what "no polling" means in practice: this hook
 * has no timer and no interval.
 */
export function useSettingsKeys(apiUrl: string, generation: number) {
  const [keys, setKeys] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (generation === 0) return; // not joined yet; fetching now could miss a change
    let live = true;
    const load = async () => {
      try {
        const response = await fetch(`${apiUrl}/settings/`);
        const body = (await response.json()) as { keys?: string[] };
        if (live) {
          setKeys(body.keys ?? []);
          setError(null);
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

  return { keys, error };
}
