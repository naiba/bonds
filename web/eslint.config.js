import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      '@typescript-eslint/ban-ts-comment': ['error', {
        'ts-ignore': true,
        'ts-expect-error': true,
        'ts-nocheck': true,
      }],
      // Forbid hardcoded color/background literals inside JSX style props.
      // A literal "#fff" doesn't switch with dark mode and ignores theme
      // overrides; use AntD tokens (e.g. token.colorTextLightSolid for
      // white-on-brand, token.colorError for danger, token.colorPrimary
      // for the brand accent). Third-party brand colors that must stay
      // constant (e.g. Telegram blue) can opt out with a narrow
      // `// eslint-disable-next-line no-restricted-syntax` comment.
      'no-restricted-syntax': ['error', {
        // Only flag hex and named colors — rgba()/rgb() with explicit
        // alpha is often intentional (overlays, semi-transparent
        // backgrounds), and forbidding those produces too many false
        // positives. Hex and named colors should always be tokens.
        selector: "JSXAttribute[name.name='style'] ObjectExpression Property[key.name=/^(color|background|backgroundColor|borderColor)$/] > Literal[value=/^(#[0-9a-fA-F]{3,8}|red|blue|green|black|white|grey|gray)$/]",
        message: 'Hardcoded color in style prop. Use theme tokens (token.colorXxx) so dark/light mode switching works.',
      }],
    },
  },
  {
    files: ['src/stores/**/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': 'off',
    },
  },
  {
    files: ['src/api/*.ts'],
    rules: {
      '@typescript-eslint/ban-ts-comment': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': 'off',
    },
  },
])
