import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Typography from '@mui/material/Typography';

/**
 * The settings that exist, as of the last fetch.
 *
 * A failed fetch is shown as a failure. Rendering "none stored" when the
 * request never succeeded says something false about the server, and that is
 * exactly the state a broken API is in.
 */
export function KeyList({
  keys,
  error,
}: {
  keys: string[];
  error: string | null;
}) {
  if (error) {
    return (
      <Typography color="error" sx={{ py: 1 }}>
        Could not load settings: {error}
      </Typography>
    );
  }
  if (keys.length === 0) {
    return (
      <Typography color="text.secondary" sx={{ py: 1 }}>
        No settings stored.
      </Typography>
    );
  }
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, py: 1 }}>
      {keys.map((key) => (
        <Chip key={key} label={key} size="small" />
      ))}
    </Box>
  );
}
