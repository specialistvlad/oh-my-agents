/**
 * Delay before reconnect attempt `attempt` (1-based), in milliseconds.
 *
 * Exponential with a ceiling, plus jitter. The jitter is the part that
 * matters: without it every client dropped by one server restart comes back
 * at the same instant and knocks it over again.
 */
export function backoffDelay(
  attempt: number,
  random: () => number = Math.random
): number {
  const base = Math.min(BASE_MS * 2 ** (attempt - 1), MAX_MS);
  return Math.round(base / 2 + random() * (base / 2));
}

const BASE_MS = 500;
const MAX_MS = 15_000;
