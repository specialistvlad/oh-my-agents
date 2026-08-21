import eslint from '@eslint/js';
import prettier from 'eslint-config-prettier';
import tseslint from 'typescript-eslint';

const bannedHooks = [
  'useState',
  'useEffect',
  'useRef',
  'useReducer',
  'useMemo',
  'useCallback',
  'useLayoutEffect',
  'useImperativeHandle',
];

export default tseslint.config(
  eslint.configs.recommended,
  ...tseslint.configs.recommended,
  prettier,
  {
    rules: {
      '@typescript-eslint/no-unused-vars': [
        'error',
        { varsIgnorePattern: '^_', ignoreRestSiblings: true },
      ],
      'max-lines': [
        'error',
        { max: 150, skipBlankLines: false, skipComments: false },
      ],
    },
  },
  {
    // Logic-in-hooks: component files hold markup, hooks hold state.
    files: ['src/components/**/*.tsx', 'src/routes/**/*.tsx'],
    ignores: ['src/components/**/use*.tsx'],
    rules: {
      'no-restricted-syntax': [
        'error',
        ...bannedHooks.map((hook) => ({
          selector: `CallExpression[callee.name="${hook}"]`,
          message: `${hook} is not allowed in component files. Extract into a custom hook (useXxx.ts).`,
        })),
      ],
    },
  },
  {
    // Build/codegen scripts run in Node, where console and process are globals.
    files: ['scripts/**/*.{mjs,js}'],
    languageOptions: {
      globals: { console: 'readonly', process: 'readonly', URL: 'readonly' },
    },
  },
  {
    // Spec files grow with coverage; the 150-line cap is a readability rule
    // for production code, not a meaningful bound here.
    files: ['**/*.test.ts', '**/*.test.tsx'],
    rules: {
      'max-lines': [
        'error',
        { max: 250, skipBlankLines: false, skipComments: false },
      ],
    },
  },
  {
    ignores: ['dist/', 'public/', '.vite/', 'src/core/routeTree.gen.ts'],
  }
);
