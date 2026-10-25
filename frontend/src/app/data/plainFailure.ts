// Go's offline sentences, offline.Sentence in internal/offline/offline.go: "Can't reach <site>."
// then ADVICE. Go may put its own context in front ("site news: startup: …"); a person sees the
// sentence alone.
export const ADVICE = " Check your internet connection, then try again.";
export const OFFLINE = "Can't reach the league site." + ADVICE;
const SENTENCE = new RegExp("Can't reach [^.]+\\." + ADVICE.replace(/[.,]/g, '\\$&'));

export function plainFailure(text: string): string {
  return text.match(SENTENCE)?.[0] ?? text;
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
            const plain = plainFailure(text);
            throw plain === text ? cause : new Error(plain);
          });
      }
    }
  }
}
