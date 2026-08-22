import tailwindcss from '@tailwindcss/vite';
import { TanStackRouterVite } from '@tanstack/router-plugin/vite';
import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';

// The repo keeps one .env at the root; envDir points Vite at it so the web
// and api services stay configured from a single file.
const envDir = '../..';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, envDir, '');
  const port = Number(env.WEB_PORT ?? 39180);

  return {
    envDir,
    resolve: {
      tsconfigPaths: true,
    },
    server: { port, strictPort: true },
    preview: { port, strictPort: true },
    plugins: [
      TanStackRouterVite({
        routesDirectory: './src/routes',
        generatedRouteTree: './src/core/routeTree.gen.ts',
      }),
      react(),
      tailwindcss(),
    ],
    build: {
      target: 'es2022',
    },
  };
});
