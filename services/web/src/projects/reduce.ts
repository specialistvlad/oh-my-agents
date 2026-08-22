import {
  PROJECT_CHANGED,
  PROJECT_CREATED,
  PROJECT_REMOVED,
  type Project,
} from './types';

/**
 * Applies one event to the list.
 *
 * Events carry the whole record (ADR-0010), so the list stays current without
 * a fetch after every change. Sorting here rather than trusting arrival order
 * keeps every client showing the same order, whatever order events reached it.
 *
 * Create and change are treated identically: a create for something already
 * present is a change, which is what makes applying an event twice harmless —
 * and applying one twice is exactly what happens when a fetch and an event
 * describe the same state.
 */
export function reduce(
  current: Project[],
  kind: string,
  data: unknown
): Project[] {
  if (kind === PROJECT_REMOVED) {
    const { id } = (data ?? {}) as { id?: string };
    return id ? current.filter((p) => p.id !== id) : current;
  }
  if (kind !== PROJECT_CREATED && kind !== PROJECT_CHANGED) return current;

  const incoming = data as Project | undefined;
  if (!incoming?.id) return current;

  const without = current.filter((p) => p.id !== incoming.id);
  return sort([...without, incoming]);
}

/** By name, then id, matching what the api returns. */
export function sort(all: Project[]): Project[] {
  return [...all].sort(
    (a, b) =>
      a.name.toLowerCase().localeCompare(b.name.toLowerCase()) ||
      a.id.localeCompare(b.id)
  );
}
