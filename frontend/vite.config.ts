import {createHash} from 'node:crypto'
import {existsSync, readFileSync} from 'node:fs'
import {join} from 'node:path'
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
  let listed: string[] = []
  return {
    name: 'twr-pwa-shell',
    apply: 'build',
    // Run after Vite drops the empty script files of CSS-only chunks. Listing one of those made every
    // phone's worker install fail (one missing file fails the whole precache), so no update ever ran.
    enforce: 'post',
    transformIndexHtml: () => [{
      tag: 'meta', attrs: {name: 'theme-color', content: '#090a0c'}, injectTo: 'head',
    }, {
      tag: 'link', attrs: {rel: 'manifest', href: '/app/manifest.webmanifest'}, injectTo: 'head',
    }],
    generateBundle(_options, bundle) {
      const assets = Object.values(bundle).filter(output => output.fileName.startsWith('assets/')
        && /[.-][A-Za-z0-9_-]{8,}\.[^/]+$/.test(output.fileName))
        .map(output => `/app/${output.fileName}`);
      listed = assets;
      const shell = JSON.stringify(['/app/', ...assets]);
      const template = readFileSync(new URL('./pwa/sw.js', import.meta.url), 'utf8');
      // Version = hash of what the worker precaches plus the worker code itself. A release that
      // changes any shell file or the worker gets a new version; an identical build does not.
      const build = createHash('sha256').update(shell).update(template).digest('hex').slice(0, 12);
      const worker = template.replace('__APP_SHELL__', shell).replace('__TWR_BUILD__', build);
      const manifest = readFileSync(new URL('./pwa/manifest.webmanifest', import.meta.url), 'utf8');
      this.emitFile({type: 'asset', fileName: 'sw.js', source: worker});
      this.emitFile({type: 'asset', fileName: 'manifest.webmanifest', source: manifest});
    },
    // Guard: the worker must list only files this build wrote, or no phone can install it.
    writeBundle(options) {
      const dir = options.dir ?? ''
      const missing = listed.filter(file => !existsSync(join(dir, file.slice('/app/'.length))))
      if (missing.length) throw new Error(`sw.js lists files the build did not write: ${missing.join(', ')}`)
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
