import js from '@eslint/js'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'

export default [
  { ignores: ['node_modules/**', 'dist/**', 'dev-dist/**', 'build/**', 'coverage/**', '*.min.js'] },

  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/recommended'],

  {
    languageOptions: {
      globals: {
        // Browser globals
        window: 'readonly',
        document: 'readonly',
        navigator: 'readonly',
        console: 'readonly',
        fetch: 'readonly',
        URL: 'readonly',
        Blob: 'readonly',
        File: 'readonly',
        FileReader: 'readonly',
        FormData: 'readonly',
        HTMLElement: 'readonly',
        HTMLInputElement: 'readonly',
        HTMLTextAreaElement: 'readonly',
        HTMLCanvasElement: 'readonly',
        HTMLImageElement: 'readonly',
        HTMLVideoElement: 'readonly',
        Image: 'readonly',
        Event: 'readonly',
        KeyboardEvent: 'readonly',
        MouseEvent: 'readonly',
        TouchEvent: 'readonly',
        DragEvent: 'readonly',
        IntersectionObserver: 'readonly',
        MutationObserver: 'readonly',
        ResizeObserver: 'readonly',
        MediaStream: 'readonly',
        ReadableStream: 'readonly',
        TextDecoder: 'readonly',
        AbortController: 'readonly',
        Response: 'readonly',
        Request: 'readonly',
        Headers: 'readonly',
        localStorage: 'readonly',
        sessionStorage: 'readonly',
        location: 'readonly',
        history: 'readonly',
        requestAnimationFrame: 'readonly',
        cancelAnimationFrame: 'readonly',
        setTimeout: 'readonly',
        clearTimeout: 'readonly',
        setInterval: 'readonly',
        clearInterval: 'readonly',
        queueMicrotask: 'readonly',
        atob: 'readonly',
        btoa: 'readonly',
        crypto: 'readonly',
        performance: 'readonly',
        getComputedStyle: 'readonly',
        matchMedia: 'readonly',
        alert: 'readonly',
        confirm: 'readonly',
        Node: 'readonly',
        PointerEvent: 'readonly',
        Notification: 'readonly',
        PublicKeyCredentialCreationOptions: 'readonly',
        PublicKeyCredential: 'readonly',
        CredentialCreationOptions: 'readonly',
        // Vitest globals
        describe: 'readonly',
        it: 'readonly',
        expect: 'readonly',
        vi: 'readonly',
        beforeEach: 'readonly',
        afterEach: 'readonly',
        beforeAll: 'readonly',
        afterAll: 'readonly',
        test: 'readonly',
      },
    },
  },

  {
    files: ['**/*.vue'],
    languageOptions: {
      parserOptions: { parser: tseslint.parser },
    },
  },

  {
    rules: {
      // Align with project conventions
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'warn',

      // Vue conventions (script setup, Composition API)
      'vue/multi-word-component-names': 'off',
      'vue/require-default-prop': 'off',
      'vue/no-v-html': 'off',
      'vue/max-attributes-per-line': 'off',
      'vue/singleline-html-element-content-newline': 'off',
      'vue/html-self-closing': 'off',
      'vue/attributes-order': 'off',
      'vue/no-mutating-props': 'off',
      'vue/no-parsing-error': 'off',
    },
  },
]
