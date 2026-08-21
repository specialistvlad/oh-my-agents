import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';

import type { RealtimeEvent } from '@/realtime/types';

/** The events this client has been told about, newest first. */
export function EventList({ events }: { events: RealtimeEvent[] }) {
  if (events.length === 0) {
    return (
      <Typography color="text.secondary" sx={{ py: 2 }}>
        Nothing yet. Write a setting — or open this page in a second tab and
        write one there.
      </Typography>
    );
  }
  return (
    <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
      {events.map((event, i) => (
        <Box
          component="li"
          key={`${event.seq}-${i}`}
          sx={{
            display: 'flex',
            gap: 2,
            py: 1,
            borderBottom: 1,
            borderColor: 'divider',
            fontFamily: 'monospace',
            fontSize: 14,
          }}>
          <Box sx={{ color: 'text.secondary', minWidth: 48 }}>
            {event.seq || '—'}
          </Box>
          <Box sx={{ fontWeight: 600, minWidth: 160 }}>{event.kind}</Box>
          <Box sx={{ color: 'text.secondary' }}>
            {JSON.stringify(event.data)}
          </Box>
        </Box>
      ))}
    </Box>
  );
}
