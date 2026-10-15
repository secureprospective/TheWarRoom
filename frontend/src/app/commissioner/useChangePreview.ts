import { useEffect, useState } from 'react';
import { bodyOf, type Change } from './forms';
import type { Preview } from './preview';
import type { Safeguard } from './rules';
import { readPreview, refusalText } from './proxy';

type Review = { body: string; preview?: Preview; error?: string };

// useChangePreview asks Go to look over the exact change a form would send, again whenever it changes.
// A reading for an older draft, or for another league version than the one the form froze, is dropped, so
// the confirm only ever belongs to the draft whose preview was read. `wait` says why it cannot be
// confirmed yet: Go's blockers word for word, a failed reading, or that the reading is still on its way.
// `safeguard` is the last one Go named, kept while the next reading is on its way so the confirm control
// does not change under the person using it; the confirm stays off until that reading arrives.
export function useChangePreview(change: Change | null, request: string) {
  const body = change ? bodyOf(change, request) : '';
  const [review, setReview] = useState<Review>();
  const [safeguard, setSafeguard] = useState<Safeguard | null>(null);
  useEffect(() => {
    if (!body) return undefined;
    let active = true;
    void readPreview(body).then(preview => {
      if (preview.safeguard !== null &&
        (preview.target.league !== change?.league || preview.target.version !== change.expected))
        throw new Error('The preview does not match this league version. Reopen the change.');
      if (!active) return;
      setReview({ body, preview });
      setSafeguard(preview.safeguard);
    }).catch(cause => { if (active) setReview({ body, error: refusalText(cause) }); });
    return () => { active = false; };
  }, [body]);
  const current = review?.body === body ? review : undefined;
  const preview = current?.preview;
  const wait = current?.error || (!preview ? 'Reading the change preview…' : preview.blockers.join(' '));
  return { preview, wait, safeguard };
}
