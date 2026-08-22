import { defineConfig } from 'vitest/config';

// The `node` environment is the default, because most tests here are pure
// logic. A test that needs a DOM opts in per-file with a
// `// @vitest-environment jsdom` pragma — component tests do.
// Path resolution mirrors vite.config.ts.
export default defineConfig({
  resolve: {
    tsconfigPaths: true,
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts', 'src/**/*.test.tsx'],
  },
});
