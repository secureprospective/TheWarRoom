// Beyond this bound, render only visible rows (§18), not a larger DOM.
export function assertShortList(n: number, max = 80) {
  if (n > max) throw new Error(`List has ${n} rows (limit ${max}); virtualization required`);
}
