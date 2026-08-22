import { createFileRoute } from '@tanstack/react-router';
import { Trash2, Zap } from 'lucide-react';

import { ConnectionBadge } from '@/components/ConnectionBadge';
import { EventList } from '@/components/EventList';
import { KeyList } from '@/components/KeyList';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { configuration } from '@/core/configuration';
import { useRealtime } from '@/hooks/useRealtime';
import { useRealtimeWriter } from '@/hooks/useRealtimeWriter';
import { useSettingsKeys } from '@/hooks/useSettingsKeys';

export const Route = createFileRoute('/')({ component: Index });

const ROOMS = ['settings'];

function Index() {
  const { client, status, events, generation } = useRealtime(
    configuration.apiUrl,
    ROOMS
  );
  const { keys, error: keysError } = useSettingsKeys(
    configuration.apiUrl,
    generation
  );
  const { write, remove, busy, error } = useRealtimeWriter(client);

  return (
    <main className="mx-auto max-w-3xl px-6 py-12">
      <div className="mb-1 flex items-center gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">oh-my-agents</h1>
        <ConnectionBadge status={status} />
      </div>
      <p className="mb-6 text-sm text-muted-foreground">
        One socket carries writes out and events back. State is fetched once
        when the rooms are joined; after that nothing on this page polls or
        refetches.
      </p>

      <div className="mb-6 flex flex-wrap gap-3">
        <Button
          disabled={busy}
          onClick={() =>
            write('demo/clicked', { at: new Date().toISOString() })
          }>
          <Zap className="size-4" aria-hidden />
          Write a setting
        </Button>
        <Button
          variant="outline"
          disabled={busy}
          onClick={() => write('demo/counter', { n: events.length })}>
          Write another
        </Button>
        <Button
          variant="ghost"
          disabled={busy}
          onClick={() => remove('demo/clicked')}>
          <Trash2 className="size-4" aria-hidden />
          Delete one
        </Button>
      </div>
      {error ? <p className="mb-4 text-sm text-danger">{error}</p> : null}

      <h2 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
        Stored settings — fetched{' '}
        {generation === 0 ? 'not yet' : `${generation}×`}
      </h2>
      <KeyList keys={keys} error={keysError} />

      <Separator className="my-6" />

      <h2 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
        Live events
      </h2>
      <EventList events={events} />
    </main>
  );
}
