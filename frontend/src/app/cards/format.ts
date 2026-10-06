export function formatMoney(cents: number): string {
  const millions = (cents / 100_000_000).toFixed(1);
  return `$${millions === '-0.0' ? '0.0' : millions}M`;
}

export function ageAt(birthdate: number | undefined, asOf: Date): number | undefined {
  if (birthdate === undefined) return undefined;
  const born = new Date(birthdate * 1000);
  if (!Number.isFinite(born.getTime()) || !Number.isFinite(asOf.getTime()) || born > asOf) return undefined;
  const birthdayAhead = asOf.getUTCMonth() < born.getUTCMonth()
    || (asOf.getUTCMonth() === born.getUTCMonth() && asOf.getUTCDate() < born.getUTCDate());
  return asOf.getUTCFullYear() - born.getUTCFullYear() - Number(birthdayAhead);
}
