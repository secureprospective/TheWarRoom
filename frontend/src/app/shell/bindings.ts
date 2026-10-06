import { isTypingTarget } from '../../components/board/keys';
import type { createCommands } from '../commands/registry';
import { parseRoute, formatRoute } from './routes';
import { routeFor } from './nodes';
type Executor = ReturnType<typeof createCommands>;
export function dispatchKey(event: Pick<KeyboardEvent, 'key' | 'target' | 'ctrlKey' | 'metaKey' | 'altKey'>, executor: Executor): boolean {
  if (executor.read().harness || isTypingTarget(event.target) || event.ctrlKey || event.metaKey || event.altKey) return false;
  return executor.key(event.key);
}
export function connectRoutes(executor: Executor, location: { hash: string }, replace: (hash: string) => void) {
  const toHash = () => {
    const s = executor.read();
    const hash = formatRoute(routeFor(s.node, s.workspace[s.node])!);
    if (location.hash !== hash) replace(hash);
  };
  const fromHash = () => { executor.dispatch('nav.open', parseRoute(location.hash)); };
  const unsubscribe = executor.subscribe(toHash);
  fromHash();
  return { fromHash, unsubscribe };
}
