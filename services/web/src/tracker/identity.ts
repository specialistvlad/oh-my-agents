import type { Actor } from './types';

const STORED = 'oma.identity';

/**
 * Who this browser says it is.
 *
 * There is no authentication by design (ADR-0012): an actor is a claim, never
 * evidence, and anyone can say they are anyone. What that buys is that nothing
 * here has to verify, ask permission, or fail closed — but it also means the
 * activity feed is only worth reading if something actually declares a name.
 *
 * Per device, like the layout (ADR-0011), because who is at this keyboard is a
 * property of the screen rather than of the work.
 *
 * The fallback exists so the feed never records an empty actor: an unnamed
 * person is still better attribution than none, and it is visibly a default
 * rather than a claim about who someone is.
 */
export const UNKNOWN: Actor = { kind: 'human', id: 'someone' };

export function loadIdentity(): Actor {
  try {
    const held = localStorage.getItem(STORED);
    if (!held) return UNKNOWN;
    const name = held.trim();
    return name ? { kind: 'human', id: name } : UNKNOWN;
  } catch {
    return UNKNOWN;
  }
}

export function saveIdentity(name: string): Actor {
  const trimmed = name.trim();
  try {
    if (trimmed) localStorage.setItem(STORED, trimmed);
    else localStorage.removeItem(STORED);
  } catch {
    // A browser refusing storage is not a reason to stop working.
  }
  return trimmed ? { kind: 'human', id: trimmed } : UNKNOWN;
}
