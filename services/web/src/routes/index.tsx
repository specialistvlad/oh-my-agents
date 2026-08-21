import Container from '@mui/material/Container';
import Typography from '@mui/material/Typography';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/')({
  component: Index,
});

function Index() {
  return (
    <Container sx={{ py: 6 }}>
      <Typography variant="h4">oh-my-agents</Typography>
    </Container>
  );
}
