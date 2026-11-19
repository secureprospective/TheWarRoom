import { validateShape, type Value } from './shape';
import { nativeCommands, serverCommands } from './native.gen';

export type NativeCommandId = keyof typeof nativeCommands;
export type HostedWorkflowCommandId = keyof typeof serverCommands;
type NativeRequestCommandId = NativeCommandId | HostedWorkflowCommandId;
type CatalogRow = { definition: { Availability: string; Modes: readonly string[]; Permission: string } };
type NativeDefinition<K extends NativeRequestCommandId> =
  K extends keyof typeof nativeCommands ? typeof nativeCommands[K]['definition'] :
  K extends keyof typeof serverCommands ? typeof serverCommands[K]['definition'] : never;

export type NativeMode<K extends NativeRequestCommandId = NativeRequestCommandId> =
  NativeDefinition<K>['Modes'][number];

export type NativeArguments<K extends NativeRequestCommandId> =
  Value<NativeDefinition<K>['Arguments']>;
export type NativeRequest = { [K in NativeRequestCommandId]: {
  command: K; mode: NativeMode<K>; arguments: NativeArguments<K>;
} }[NativeRequestCommandId];

export function hostedCommandAvailable(command: string, mode: 'preview' | 'read' = 'preview'): boolean {
  const definition = (serverCommands as unknown as Record<string, CatalogRow>)[command]?.definition
    ?? (nativeCommands as unknown as Record<string, CatalogRow>)[command]?.definition;
  return Boolean(definition && definition.Availability !== 'desktop' && definition.Modes.includes(mode));
}

export function isAccountPreferenceWorkflow(command: string): boolean {
  return (serverCommands as unknown as Record<string, CatalogRow>)[command]?.definition.Permission
    === 'account-owner';
}

// Hosted forms validate the generated wire contract without inventing capability grants.
// The server admits the real account and current arguments on every request.
export function nativeEnvelope(request: NativeRequest) {
  const definition = request.command in serverCommands
    ? serverCommands[request.command as keyof typeof serverCommands]?.definition
    : nativeCommands[request.command as keyof typeof nativeCommands]?.definition;
  if (!definition || !(definition.Modes as readonly string[]).includes(request.mode)
    || definition.Availability === 'desktop') throw new Error('This league action is not available here.');
  validateShape(request.arguments, definition.Arguments, 'League action');
  return { ...request, version: definition.Version };
}
