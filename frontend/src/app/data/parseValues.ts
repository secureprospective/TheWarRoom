type Row = Record<string, unknown>;

export function object(value: unknown, path: string, keys: string[]): Row {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error(`${path}: expected object`);
  }
  const row = value as Row;
  for (const key of Object.keys(row)) {
    if (!keys.includes(key)) throw new Error(`${path}.${key}: unexpected field`);
  }
  return row;
}

export function text(value: unknown, path: string): string {
  if (typeof value !== 'string') throw new Error(`${path}: expected string`);
  return value;
}

export function integer(value: unknown, path: string, minimum = 0): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < minimum) {
    throw new Error(`${path}: expected safe integer >= ${minimum}`);
  }
  return value;
}

export function choice<T extends string>(
  value: unknown,
  path: string,
  allowed: readonly T[],
): T {
  const parsed = text(value, path);
  if (!allowed.includes(parsed as T))
    throw new Error(`${path}: expected ${allowed.join(' | ')}`);
  return parsed as T;
}

export function timestamp(value: unknown, path: string): string {
  const parsed = text(value, path);
  if (
    parsed !== '' &&
    (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(
      parsed,
    ) ||
      Number.isNaN(Date.parse(parsed)))
  ) {
    throw new Error(`${path}: expected RFC3339 timestamp or empty string`);
  }
  return parsed;
}

export function id(value: unknown, path: string, player = false): string {
  const parsed = text(value, path);
  if (!(player ? /^(?:\d{4}|[1-9]\d{4,})$/ : /^\d+$/).test(parsed)) {
    throw new Error(
      `${path}: expected ${player ? 'canonical player' : 'numeric string'} id`,
    );
  }
  return parsed;
}

export function array<T>(
  value: unknown,
  path: string,
  parse: (v: unknown, p: string) => T,
): T[] {
  if (!Array.isArray(value)) throw new Error(`${path}: expected array`);
  return value.map((entry, index) => parse(entry, `${path}[${index}]`));
}

export function requiredText(value: unknown, path: string): string {
  const parsed = text(value, path);
  if (!parsed.trim()) throw new Error(`${path}: expected nonempty string`);
  return parsed;
}
export function utc(value: unknown, path: string): string {
  const parsed = timestamp(value, path);
  if (!parsed.endsWith('Z')) throw new Error(`${path}: expected UTC timestamp`);
  return parsed;
}
