import { describe, expect, it } from 'vitest';
import { reportMeta } from './CtrlFeedbackV3';

describe('reportMeta', () => {
  it('formats app release, viewport width, and user agent', () => {
    expect(reportMeta({ appRelease: 42, viewportWidth: 1440, userAgent: 'Test browser' }))
      .toBe('App release 42 · 1440px · User agent: Test browser');
  });

  it('falls back to Unavailable when user agent is empty', () => {
    expect(reportMeta({ appRelease: 7, viewportWidth: 390, userAgent: '' }))
      .toBe('App release 7 · 390px · User agent: Unavailable');
  });
});
