export type SurfaceAction = {
  screen: string; value: string;
  action: 'view' | 'find' | 'clear' | 'open' | 'group' | 'close' | 'tray' | 'pick' | 'check';
};
type Handler = (args: SurfaceAction) => void | Promise<void>;

const handlers = new Map<string, Handler>();
export const surfaceActions = {
  register(screen: string, handler: Handler) {
    handlers.set(screen, handler);
    return () => { if (handlers.get(screen) === handler) handlers.delete(screen); };
  },
  run(args: SurfaceAction) {
    const handler = handlers.get(args.screen);
    if (!handler) {
      const message = 'This league view is no longer open. Open it again and retry your action.';
      console.error(message);
      throw new Error(message);
    }
    return handler(args);
  },
};
