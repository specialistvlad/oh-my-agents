import { defineConfig } from 'vitest/config';

// Pure-logic unit tests. The `node` environment is enough; a test that needs
// a DOM opts into jsdom per-file with a `// @vitest-environment` pragma.
// Path resolution mirrors vite.config.ts.
export default defineConfig({
  resolve: {
    tsconfigPaths: true,
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
});
