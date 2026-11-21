import { vi } from 'vitest';

/** stubFrames routes animation-frame requests to a test callback. It lives in a .ts file because the
 *  performance rule bans the frame API by name in .tsx views. */
export function stubFrames(take: (cb: FrameRequestCallback) => number) {
  return vi.spyOn(globalThis, 'requestAnimationFrame').mockImplementation(take);
}
