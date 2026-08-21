import CssBaseline from '@mui/material/CssBaseline';
import { Outlet, createRootRoute } from '@tanstack/react-router';

export const Route = createRootRoute({
  component: () => (
    <>
      <CssBaseline />
      <Outlet />
    </>
  ),
});
