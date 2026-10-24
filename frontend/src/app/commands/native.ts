import { nativeCommands } from './native.gen';

export type NativeCommandId = keyof typeof nativeCommands;
export type NativeMode<K extends NativeCommandId = NativeCommandId> =
  typeof nativeCommands[K]['definition']['Modes'][number];

type Shape = {
  readonly Type: string; readonly Enum: readonly string[] | null; readonly Nullable: boolean;
  readonly Fields: readonly { readonly Name: string; readonly Optional: boolean;
    readonly Schema: Shape }[] | null;
  readonly Items: Shape | null;
};
type Field = NonNullable<Shape['Fields']>[number];
type ObjectValue<F extends readonly Field[]> = F extends readonly [] ? Record<string, never> :
  { [P in F[number] as P['Optional'] extends false ? P['Name'] : never]: Value<P['Schema']> } &
  { [P in F[number] as P['Optional'] extends true ? P['Name'] : never]?: Value<P['Schema']> };
type Value<S extends Shape> = (S['Enum'] extends readonly string[] ? S['Enum'][number] :
  S['Type'] extends 'object' ? S['Fields'] extends readonly Field[] ? ObjectValue<S['Fields']> : never :
  S['Type'] extends 'array' ? S['Items'] extends Shape ? Value<S['Items']>[] : never :
  S['Type'] extends 'map' ? S['Items'] extends Shape ? Record<string, Value<S['Items']>> : never :
  S['Type'] extends 'string' ? string : S['Type'] extends 'boolean' ? boolean :
  S['Type'] extends 'integer' | 'number' ? number : unknown) |
  (S['Nullable'] extends true ? null : never);
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

function validateShape(value: unknown, shape: Shape, path: string): void {
  if (value === null && shape.Nullable) return;
  if (shape.Enum && !shape.Enum.includes(String(value))) throw new Error(`${path}: unknown choice`);
  switch (shape.Type) {
    case 'json': return; // Its Raw validator remains the authority for this field.
    case 'string':
    case 'boolean':
      if (typeof value !== shape.Type) throw new Error(`${path}: expected ${shape.Type}`);
      return;
    case 'integer':
    case 'number':
      if (typeof value !== 'number' || !Number.isFinite(value) ||
        (shape.Type === 'integer' && !Number.isSafeInteger(value)))
        throw new Error(`${path}: expected ${shape.Type}`);
      return;
    case 'array':
      if (!Array.isArray(value) || !shape.Items) throw new Error(`${path}: expected list`);
      value.forEach((item, i) => validateShape(item, shape.Items!, `${path}.${i}`));
      return;
    case 'map':
    case 'object': {
      if (typeof value !== 'object' || value === null || Array.isArray(value))
        throw new Error(`${path}: expected object`);
      const fields = shape.Fields ?? [];
      for (const [name, item] of Object.entries(value)) {
        const field = fields.find(f => f.Name === name);
        const itemShape = shape.Type === 'map' ? shape.Items : field?.Schema;
        if (!itemShape) throw new Error(`${path}: unknown argument ${name}`);
        if (item !== undefined) validateShape(item, itemShape, `${path}.${name}`);
      }
      for (const field of fields) {
        if (!field.Optional && (!(field.Name in value) ||
          (value as Record<string, unknown>)[field.Name] === undefined))
          throw new Error(`${path}: missing ${field.Name}`);
      }
      return;
    }
    default: throw new Error(`${path}: unsupported argument shape`);
  }
}
