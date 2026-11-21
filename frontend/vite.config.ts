import {readFileSync} from 'node:fs'
import {defineConfig, type Plugin} from 'vitest/config'
import react from '@vitejs/plugin-react'

const webBuild = process.env.TWR_WEB_BUILD === '1'
const appVersion = (() => {
  try {
    const raw = JSON.parse(readFileSync(new URL('../wails.json', import.meta.url), 'utf8')) as {
      info?: { productVersion?: string }
    }
    return raw.info?.productVersion || '0.5.0'
  } catch {
    return '0.5.0'
  }
})()

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

function pwaShell(): Plugin {
  return {
    name: 'twr-pwa-shell',
    apply: 'build',
    transformIndexHtml: () => [{
      tag: 'meta', attrs: {name: 'theme-color', content: '#090a0c'}, injectTo: 'head',
    }, {
      tag: 'link', attrs: {rel: 'manifest', href: '/app/manifest.webmanifest'}, injectTo: 'head',
    }],
    generateBundle(_options, bundle) {
      const assets = Object.values(bundle).filter(output => output.fileName.startsWith('assets/')
        && /[.-][A-Za-z0-9_-]{8,}\.[^/]+$/.test(output.fileName))
        .map(output => `/app/${output.fileName}`);
      const shell = JSON.stringify(['/app/', ...assets]);
      const worker = readFileSync(new URL('./pwa/sw.js', import.meta.url), 'utf8')
        .replace('__APP_SHELL__', shell);
      const manifest = readFileSync(new URL('./pwa/manifest.webmanifest', import.meta.url), 'utf8');
      this.emitFile({type: 'asset', fileName: 'sw.js', source: worker});
      this.emitFile({type: 'asset', fileName: 'manifest.webmanifest', source: manifest});
    },
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  base: webBuild ? '/app/' : '/',
  publicDir: webBuild ? 'public' : false,
  define: { __TWR_APP_VERSION__: JSON.stringify(appVersion) },
  plugins: [react(), csp(), ...(webBuild ? [pwaShell()] : [])],
  test: {setupFiles: ['src/test/nativeWailsShim.ts', 'src/test/hqPrefetchStub.ts']},
  build: {
    outDir: webBuild ? 'dist-web' : 'dist',
    manifest: true,
    rolldownOptions: {
      output: {
        entryFileNames: 'assets/[name].[hash].js',
        chunkFileNames: 'assets/[name].[hash].js',
      },
    },
  },
})
