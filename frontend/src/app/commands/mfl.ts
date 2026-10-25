import { mflCommands } from './mfl.gen';
import { validateShape, type Value } from './shape';

export type MFLCommandId = keyof typeof mflCommands;
export type MFLArguments<K extends MFLCommandId> = Value<typeof mflCommands[K]['definition']['Arguments']>;
export type MFLRequest = { [K in MFLCommandId]: {
  command: K; mode: typeof mflCommands[K]['definition']['Modes'][number]; arguments: MFLArguments<K>;
} }[MFLCommandId];
export type MFLContext = {
  availability: 'desktop' | 'server'; role: 'gm' | 'commish' | 'admin';
  // Only a direct owner action supplies this; reading or previewing never implies consent.
  ownerAction?: boolean;
};
export function mflEnvelope(request: MFLRequest, context: MFLContext) {
  const definition = mflCommands[request.command]?.definition;
  if (!definition || context.availability !== 'desktop')
    throw new Error('MFL reference plans stay on your computer; they are not hosted league actions.');
  if (!definition.Role.split(',').includes(context.role))
    throw new Error('This action is not available to you.');
  if (!(definition.Modes as readonly string[]).includes(request.mode))
    throw new Error('Choose an available plan action.');
  if (definition.Gravity === 'G2' && !context.ownerAction)
    throw new Error('Choose this action on the saved bid plan first. Nothing has changed.');
  validateShape(request.arguments, definition.Arguments, 'MFL reference plan');
  return { ...request, version: definition.Version };
}

// This port is local Wails only. There is deliberately no HTTPS/agent-token fallback.
export function createMFLClient(port: (request: ReturnType<typeof mflEnvelope>) => Promise<unknown>) {
  return {
    invoke(request: MFLRequest, context: MFLContext) {
      return port(mflEnvelope(request, context));
    },
  };
}
