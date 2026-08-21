import type { Outbound, RealtimeEvent } from './types';

/** Parses a frame, or returns null for anything we cannot act on. */
export function decode(raw: unknown): Outbound | null {
  if (typeof raw !== 'string') return null;
  try {
    return JSON.parse(raw) as Outbound;
  } catch {
    return null;
  }
}

/** Reads an event out of a frame, or null if it is not a usable one. */
export function toEvent(frame: Outbound): RealtimeEvent | null {
  if (frame.type !== 'event' || !frame.room || !frame.kind) return null;
  return {
    room: frame.room,
    seq: frame.seq ?? 0,
    kind: frame.kind,
    data: frame.data,
  };
}
