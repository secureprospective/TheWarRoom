import { useEffect } from 'react';
import { commands } from '../commands/registry';
import { connectRoutes, dispatchKey } from './bindings';
export function useShellBindings() {
  useEffect(() => {
    const routes = connectRoutes(commands, window.location, hash => window.history.replaceState(null, '', hash));
    const keydown = (event: KeyboardEvent) => {
      if (dispatchKey(event, commands)) event.preventDefault();
    };
    window.addEventListener('hashchange', routes.fromHash);
    window.addEventListener('keydown', keydown);
    return () => { routes.unsubscribe(); window.removeEventListener('hashchange', routes.fromHash); window.removeEventListener('keydown', keydown); };
  }, []);
}
