import { validateShape, type Value } from './shape';
import { nativeCommands } from './native.gen';

export type NativeCommandId = keyof typeof nativeCommands;
export type NativeMode<K extends NativeCommandId = NativeCommandId> =
  typeof nativeCommands[K]['definition']['Modes'][number];

export type NativeArguments<K extends NativeCommandId> =
  Value<typeof nativeCommands[K]['definition']['Arguments']>;
export type NativeRequest = { [K in NativeCommandId]: {
  command: K; mode: NativeMode<K>; arguments: NativeArguments<K>;
} }[NativeCommandId];

// Hosted forms validate the generated wire contract without inventing capability grants.
// The server admits the real account and current arguments on every request.
export function nativeEnvelope(request: NativeRequest) {
  const definition = nativeCommands[request.command]?.definition;
  if (!definition || !(definition.Modes as readonly string[]).includes(request.mode)
    || definition.Availability !== 'both') throw new Error('This league action is not available here.');
  validateShape(request.arguments, definition.Arguments, 'League action');
  return { ...request, version: definition.Version };
}
