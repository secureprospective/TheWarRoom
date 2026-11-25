// Go's offline sentences, offline.Sentence in internal/offline/offline.go: "Can't reach <site>."
// then ADVICE. Go may put its own context in front ("site news: startup: …"); a person sees the
// sentence alone.
export const ADVICE = " Check your internet connection, then try again.";
export const OFFLINE = "Can't reach the league site." + ADVICE;
const SENTENCE = new RegExp("Can't reach [^.]+\\." + ADVICE.replace(/[.,]/g, '\\$&'));

export function plainFailure(text: string): string {
  const cleaned = text.replace(/^Error:\s*/i, '');
  return cleaned.match(SENTENCE)?.[0] ?? cleaned;
}
