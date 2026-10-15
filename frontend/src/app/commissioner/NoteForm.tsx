import { useEffect, useState } from 'react';
import { ChangeBar } from './ChangeBar';
import { useRequestKey, type Change } from './forms';
import type { People } from './people';
import { PreviewReview } from './PreviewReview';
import { leagueState } from './proxy';
import { useChangePreview } from './useChangePreview';

// What each kind of note says on the screen and what it sends. Owners read both in the app, as written.
const kinds = {
  notice: { command: 'commish.notice.publish', label: 'Post an official notice', form: 'notice',
    field: 'Notice', confirm: 'Post this notice',
    note: 'Owners read it in the league notices. The app sends no email and no messages.' },
  note: { command: 'commish.whatsnew.post', label: 'Post a league note', form: 'note',
    field: 'League note', confirm: 'Post this note',
    note: 'Owners read it in What’s New. The app sends no email and no messages.' },
} as const;

// NoteForm posts an official notice or a league note, in the side tray. Go decides who may post and
// refuses anything that is not plain text; the person writes the words and may pin them to the top.
export function NoteForm({ people, kind }: { people: People; kind: keyof typeof kinds }) {
  const words = kinds[kind];
  const [text, setText] = useState('');
  const [pin, setPin] = useState(false);
  const [reason, setReason] = useState('');
  const outcome = leagueState.use().outcome;
  const landed = outcome.status === 'done' && outcome.form === words.form ? outcome.receipt.entry : '';
  useEffect(() => {
    if (!landed) return;
    setText(''); setPin(false); setReason('');
  }, [landed]);
  const written = text.trim();
  const change: Change | null = written ? {
    league: people.league, command: words.command, args: { note: { text: written, pin } },
    reason: reason.trim(), expected: people.revision, effective: 'now', visibility: 'league',
  } : null;
  const request = useRequestKey(change);
  const { preview, wait, safeguard } = useChangePreview(change, request);
  return (
    <section className="league-form" aria-label={words.label}>
      <PreviewReview preview={preview} confirm={
        <ChangeBar form={words.form} change={change} request={request} reason={reason} onReason={setReason}
          safeguard={safeguard} confirmLabel={words.confirm} note={words.note}
          blocked={written ? wait : 'Write what owners should read.'} />
      }>
        <label>{words.field}
          <textarea value={text} rows={4} onChange={e => setText(e.target.value)} />
        </label>
        <label>Pin to the top
          <input type="checkbox" checked={pin} onChange={e => setPin(e.target.checked)} />
        </label>
      </PreviewReview>
    </section>
  );
}
