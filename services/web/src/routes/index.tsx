import Button from '@mui/material/Button';
import Container from '@mui/material/Container';
import Divider from '@mui/material/Divider';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { createFileRoute } from '@tanstack/react-router';

import { ConnectionBadge } from '@/components/ConnectionBadge';
import { EventList } from '@/components/EventList';
import { KeyList } from '@/components/KeyList';
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
    <Container sx={{ py: 6 }}>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center', mb: 1 }}>
        <Typography variant="h4">oh-my-agents</Typography>
        <ConnectionBadge status={status} />
      </Stack>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        One socket carries writes out and events back. State is fetched once
        when the rooms are joined; after that nothing on this page polls or
        refetches.
      </Typography>

      <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
        <Button
          variant="contained"
          disabled={busy}
          onClick={() =>
            write('demo/clicked', { at: new Date().toISOString() })
          }>
          Write a setting
        </Button>
        <Button
          variant="outlined"
          disabled={busy}
          onClick={() => write('demo/counter', { n: events.length })}>
          Write another
        </Button>
        <Button
          variant="outlined"
          disabled={busy}
          onClick={() => remove('demo/clicked')}>
          Delete one
        </Button>
      </Stack>
      {error ? <Typography color="error">{error}</Typography> : null}

      <Typography variant="overline" color="text.secondary">
        Stored settings — fetched{' '}
        {generation === 0 ? 'not yet' : `${generation}×`}
      </Typography>
      <KeyList keys={keys} error={keysError} />

      <Divider sx={{ my: 3 }} />

      <Typography variant="overline" color="text.secondary">
        Live events
      </Typography>
      <EventList events={events} />
    </Container>
  );
}
