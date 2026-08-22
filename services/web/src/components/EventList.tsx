import type { RealtimeEvent } from '@/realtime/types';

/** The events this client has been told about, newest first. */
export function EventList({ events }: { events: RealtimeEvent[] }) {
  if (events.length === 0) {
    return (
      <p className="py-2 text-sm text-muted-foreground">
        Nothing yet. Write a setting — or open this page in a second tab and
        write one there.
      </p>
    );
  }
  return (
    <ul className="font-mono text-sm">
      {events.map((event, i) => (
        <li
          key={`${event.seq}-${i}`}
          className="flex gap-4 border-b border-border py-2 last:border-0">
          <span className="w-12 shrink-0 text-muted-foreground">
            {event.seq || '—'}
          </span>
          <span className="w-40 shrink-0 font-semibold">{event.kind}</span>
          <span className="truncate text-muted-foreground">
            {JSON.stringify(event.data)}
          </span>
        </li>
      ))}
    </ul>
  );
}
