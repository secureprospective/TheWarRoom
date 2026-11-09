import { useEffect, useRef } from 'react';
import { surfaceActions, type SurfaceAction } from '../../commissioner/surfaceActions';

/** Keep a mounted room branch's command handler current, including inspector actions. */
export function useRoomSurface(screen: string, handler: (args: SurfaceAction) => void | Promise<void>) {
  const latest = useRef(handler);
  latest.current = handler;
  useEffect(() => surfaceActions.register(screen, (args) => latest.current(args)), [screen]);
}
