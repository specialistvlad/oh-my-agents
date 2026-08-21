import Button from '@mui/material/Button';
import Container from '@mui/material/Container';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { createFileRoute } from '@tanstack/react-router';

import { ConnectionBadge } from '@/components/ConnectionBadge';
import { EventList } from '@/components/EventList';
import { configuration } from '@/core/configuration';
import { useRealtime } from '@/hooks/useRealtime';
import { useSettingsWriter } from '@/hooks/useSettingsWriter';

export const Route = createFileRoute('/')({ component: Index });

const ROOMS = ['settings'];

function Index() {
  const { status, events } = useRealtime(configuration.apiUrl, ROOMS);
  const { write, busy, error } = useSettingsWriter(configuration.apiUrl);

  return (
    <Container sx={{ py: 6 }}>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center', mb: 1 }}>
        <Typography variant="h4">oh-my-agents</Typography>
        <ConnectionBadge status={status} />
      </Stack>
      <Typography color="text.secondary" sx={{ mb: 3 }}>
        Events arrive over a WebSocket. Nothing on this page polls.
      </Typography>

      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
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
      </Stack>
      {error ? <Typography color="error">{error}</Typography> : null}

      <EventList events={events} />
    </Container>
  );
}
