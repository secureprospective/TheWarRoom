import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { PlayerPhoto } from './PlayerPhoto';

const render = (view?: { src: string; alt: string }, team?: string) =>
  renderToStaticMarkup(createElement(PlayerPhoto, { view, team }));

describe('PlayerPhoto', () => {
  it('shows the photo Go addressed, lazily, with Go\'s alt text', () => {
    const html = render({ src: '/headshot/0501', alt: 'Photo of Kayden McDonald' }, 'HOU');
    expect(html).toContain('src="/headshot/0501"');
    expect(html).toContain('alt="Photo of Kayden McDonald"');
    expect(html).toContain('loading="lazy"');
  });
  it('shows the team, and makes no request, when Go gives no address', () => {
    for (const view of [undefined, { src: '', alt: 'Kayden McDonald' }]) {
      const html = render(view, 'HOU');
      expect(html).not.toContain('<img');
      expect(html).toContain('>HOU<');
      expect(html).toContain('player-photo-md');
    }
  });
});
