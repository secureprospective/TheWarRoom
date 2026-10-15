export type SurfaceAction = {
  screen: string; action: 'view' | 'find' | 'clear' | 'open' | 'group' | 'close' | 'tray'; value: string;
};
type Handler = (args: SurfaceAction) => void;

const handlers = new Map<string, Handler>();
export const surfaceActions = {
  register(screen: string, handler: Handler) {
    handlers.set(screen, handler);
    return () => { if (handlers.get(screen) === handler) handlers.delete(screen); };
  },
  run(args: SurfaceAction) { handlers.get(args.screen)?.(args); },
};
