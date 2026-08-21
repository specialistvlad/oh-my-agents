import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import Typography from '@mui/material/Typography';

/** The settings that exist, as of the last fetch. */
export function KeyList({ keys }: { keys: string[] }) {
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
