import { useCallback, useState } from 'react';

/**
 * Writes a setting over HTTP.
 *
 * Nothing here reads anything back. The write is announced on the socket and
 * arrives through useRealtime, including in every other open tab. That round
 * trip is the point: this is what "no polling" looks like from the caller's
 * side.
 */
export function useSettingsWriter(apiUrl: string) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const write = useCallback(
    async (key: string, value: unknown) => {
      setBusy(true);
      setError(null);
      try {
        const response = await fetch(`${apiUrl}/settings/${key}`, {
          method: 'PUT',
          body: JSON.stringify(value),
        });
        if (!response.ok) {
          setError(`${response.status} ${response.statusText}`);
        }
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : 'request failed');
      } finally {
        setBusy(false);
      }
    },
    [apiUrl]
  );

  return { write, busy, error };
}
