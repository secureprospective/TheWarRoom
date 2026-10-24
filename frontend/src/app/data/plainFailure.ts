// Go's offline sentence, report.Offline in internal/app/report/held.go. Go puts its own
// context in front of it ("site news: startup: …"); a person sees the sentence alone.
export const OFFLINE = "Can't reach the league site. Check your internet connection, then try again.";

export function plainFailure(text: string): string {
  return text.includes(OFFLINE) ? OFFLINE : text;
}

type Bound = Record<string, Record<string, Record<string, unknown>>>;

// plainBindings wraps every Wails-bound Go method once, before any screen calls one, so every
// rejection reaches its screen through plainFailure. Wails rejects with an Error.
export function plainBindings(go: unknown): void {
  if (!go || typeof go !== 'object') return;
  for (const pkg of Object.values(go as Bound)) {
    for (const api of Object.values(pkg)) {
      for (const [name, method] of Object.entries(api)) {
        if (typeof method !== 'function') continue;
        api[name] = (...args: unknown[]) => (method as (...a: unknown[]) => Promise<unknown>)(...args)
          .catch((cause: unknown) => {
            const text = cause instanceof Error ? cause.message : String(cause);
            throw plainFailure(text) === text ? cause : new Error(OFFLINE);
          });
      }
    }
  }
}
