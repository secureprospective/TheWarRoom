/** @type {import('tailwindcss').Config} */
// Tailwind is a thin consumer of the CSS custom properties defined on :root in
// style.css (the Session-C token contract is the single source of truth). Every
// alias below resolves to a var() so components can style via utilities
// (bg-surface-tile, text-text-secondary, border-hairline) OR read the var
// directly in inline style — both stay in lockstep with the one token sheet.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        surface: {
          canvas: 'var(--surface-canvas)',
          sunken: 'var(--surface-sunken)',
          tile: 'var(--surface-tile)',
          raised: 'var(--surface-raised)',
          overlay: 'var(--surface-overlay)',
        },
        text: {
          primary: 'var(--text-primary)',
          secondary: 'var(--text-secondary)',
          tertiary: 'var(--text-tertiary)',
          disabled: 'var(--text-disabled)',
        },
        green: {
          muted: 'var(--green-muted)',
          base: 'var(--green-base)',
          loud: 'var(--green-loud)',
        },
        blue: {
          muted: 'var(--blue-muted)',
          base: 'var(--blue-base)',
          loud: 'var(--blue-loud)',
        },
        amber: {
          muted: 'var(--amber-muted)',
          base: 'var(--amber-base)',
          loud: 'var(--amber-loud)',
        },
        red: {
          muted: 'var(--red-muted)',
          base: 'var(--red-base)',
          loud: 'var(--red-loud)',
        },
        hairline: 'var(--hairline)',
        'bevel-hi': 'var(--bevel-hi)',
        'bevel-lo': 'var(--bevel-lo)',
      },
      fontFamily: {
        sans: 'var(--sans)',
        mono: 'var(--mono)',
      },
      boxShadow: {
        // Optical inset-bevel elevation — the ONLY elevation the design allows
        // (no dropshadows anywhere per the Session-A/C restraint doctrine).
        bevel: 'inset 1px 1px 0 var(--bevel-hi), inset -1px -1px 0 var(--bevel-lo)',
      },
    },
  },
  plugins: [],
};
