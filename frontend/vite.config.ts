import {defineConfig, type Plugin} from 'vitest/config'
import react from '@vitejs/plugin-react'

// Content-Security-Policy for the built app: same-origin scripts only, no eval, nothing remote.
// Photos come from the asset handler (/headshot/…); Vite may inline small images and fonts as
// data: URIs. React writes style attributes, so styles allow inline. Build only: the dev server
// injects an inline React-refresh preamble and an HMR socket this policy would block.
export const contentSecurityPolicy = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data:",
  "font-src 'self' data:",
  "connect-src 'self'",
  "object-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
].join('; ')

function csp(): Plugin {
  return {
    name: 'twr-csp',
    apply: 'build',
    transformIndexHtml: () => [{
      tag: 'meta',
      attrs: {'http-equiv': 'Content-Security-Policy', content: contentSecurityPolicy},
      injectTo: 'head-prepend',
    }],
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), csp()],
  test: {setupFiles: ['src/test/hqPrefetchStub.ts']},
  build: {
    manifest: true,
    rolldownOptions: {
      output: {
        entryFileNames: 'assets/[name].[hash].js',
        chunkFileNames: 'assets/[name].[hash].js',
      },
    },
  },
})
