export type Shape = {
  readonly Type: string; readonly Enum: readonly string[] | null; readonly Nullable: boolean;
  readonly Fields: readonly { readonly Name: string; readonly Optional: boolean;
    readonly Schema: Shape }[] | null;
  readonly Items: Shape | null;
};
type Field = NonNullable<Shape['Fields']>[number];
type ObjectValue<F extends readonly Field[]> = F extends readonly [] ? Record<string, never> :
  { [P in F[number] as P['Optional'] extends false ? P['Name'] : never]: Value<P['Schema']> } &
  { [P in F[number] as P['Optional'] extends true ? P['Name'] : never]?: Value<P['Schema']> };
export type Value<S extends Shape> = (S['Enum'] extends readonly string[] ? S['Enum'][number] :
  S['Type'] extends 'object' ? S['Fields'] extends readonly Field[] ? ObjectValue<S['Fields']> : never :
  S['Type'] extends 'array' ? S['Items'] extends Shape ? Value<S['Items']>[] : never :
  S['Type'] extends 'map' ? S['Items'] extends Shape ? Record<string, Value<S['Items']>> : never :
  S['Type'] extends 'string' ? string : S['Type'] extends 'boolean' ? boolean :
  S['Type'] extends 'integer' | 'number' ? number : unknown) |
  (S['Nullable'] extends true ? null : never);

export function validateShape(value: unknown, shape: Shape, path: string): void {
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

