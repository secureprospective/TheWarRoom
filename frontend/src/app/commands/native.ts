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

// These are current, argument-scoped server answers, never roles inferred by this client.
// Display and preflight are conveniences; the server must recheck every request at admission.
export type NativeCapability = {
  command: NativeCommandId; mode: NativeMode; allowed: boolean; reason: string;
};
export type NativeContext = {
  availability: 'desktop' | 'server'; capabilities: readonly NativeCapability[];
};
export function nativeCapability(command: NativeCommandId, mode: NativeMode, context: NativeContext) {
  const definition = nativeCommands[command].definition;
  if (!(definition.Modes as readonly string[]).includes(mode))
    return { enabled: false, reason: 'This action is not available.' };
  if (definition.Availability !== 'both' && definition.Availability !== context.availability)
    return { enabled: false, reason: 'This action needs the desktop app.' };
  const answers = context.capabilities.filter(c => c.command === command && c.mode === mode);
  if (answers.length !== 1 || !answers[0].allowed)
    return { enabled: false, reason: answers[0]?.reason || 'This action is not available to you.' };
  return { enabled: true, reason: '' };
}

// Grammar is a list of named, typed arguments, not a shell or a route around form confirmation.
export function nativeGrammar(context: NativeContext, query = '') {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  return (Object.keys(nativeCommands) as NativeCommandId[]).flatMap(command => {
    const { definition, phrase } = nativeCommands[command];
    if (!words.every(word => phrase.includes(word))) return [];
    return definition.Modes.map(mode => ({
      command, mode, label: phrase[0].toUpperCase() + phrase.slice(1),
      arguments: definition.Arguments.Fields.map(f => ({ name: f.Name, required: !f.Optional })),
      ...nativeCapability(command, mode, context),
    }));
  });
}

export type NativePort = (request: {
  command: NativeCommandId; version: string; mode: NativeMode; arguments: object;
}) => Promise<unknown>;

// No bearer, role, or principal is accepted here. Existing forms own preview/reason/confirmation;
// this transport neither adds a mode nor supplies a default reason or confirmation.
export function createNativeClient(port: NativePort, current: (request: NativeRequest) => NativeContext) {
  return {
    async invoke(request: NativeRequest): Promise<unknown> {
      const definition = nativeCommands[request.command]?.definition;
      if (!definition) throw new Error('Unknown league action.');
      const capability = nativeCapability(request.command, request.mode, current(request));
      if (!capability.enabled) throw new Error(capability.reason);
      validateShape(request.arguments, definition.Arguments, 'League action');
      return port({ ...request, version: definition.Version });
    },
  };
}

// Hosted forms validate the generated wire contract without inventing capability grants.
// The server admits the real account and current arguments on every request.
export function nativeEnvelope(request: NativeRequest) {
  const definition = nativeCommands[request.command]?.definition;
  if (!definition || !(definition.Modes as readonly string[]).includes(request.mode)
    || definition.Availability !== 'both') throw new Error('This league action is not available here.');
  validateShape(request.arguments, definition.Arguments, 'League action');
  return { ...request, version: definition.Version };
}
