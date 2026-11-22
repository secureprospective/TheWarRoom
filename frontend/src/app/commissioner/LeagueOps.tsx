import { TrayLauncher } from './TrayLauncher';
import { DensityScreen } from './DensityScreen';
import { localTime } from '../clock/localTime';
import { GovernedGlyph } from '../governed/GovernedGlyph';
import { governedBy, type Note } from './notes';
import { leagueState } from './proxy';
import './commissioner.css';

const when = (iso: string) => localTime(iso, { year: 'numeric' });
// A release note carries a calendar date, not an instant, so it is read as the viewer's own day.
const day = (date: string) =>
  new Date(`${date}T12:00:00`).toLocaleDateString(undefined, { dateStyle: 'medium' });

// Withdrawing a note is an undo in the side tray, offered only where Go's ledger says this person may.
function Notes({ notes, empty, withdrawable }:
  { notes: Note[]; empty: string; withdrawable: ReadonlySet<string> }) {
  if (notes.length === 0) return <p>{empty}</p>;
  return (
    <ul className="league-notes">
      {notes.map(note => (
        <li key={note.id} data-pinned={note.pinned}>
          {note.pinned && <strong className="note-pin">Pinned </strong>}
          <span className="note-text">{note.text}</span>
          <GovernedGlyph governed={governedBy(note)} subject="this note" />
          <p className="note-by">Posted by {note.who} ·{' '}
            <time dateTime={note.when}>{when(note.when)}</time></p>
          {withdrawable.has(note.id) &&
            <TrayLauncher tray={`undo ${note.id}`} text="Withdraw" label={`Withdraw: ${note.text}`} />}
        </li>
      ))}
    </ul>
  );
}

// LeagueOps is what the league tells its owners: official notices, league notes, the app's own release
// notes, the mail log and the digest of changes. All of it is read from the ledger in Go; the app sends
// nothing outside itself.
export default function LeagueOps() {
  const { notes, digests, opsError, people, ledger, pending } = leagueState.use();
  const withdrawable = new Set(ledger?.entries.filter(e => e.undo !== '').map(e => e.id));
  const mayPost = people?.mayPostNotices === true;
  return (
    <DensityScreen screen="ops" label="League operations" pending={pending}>
      {opsError && <p role="alert">{opsError}</p>}
      {!notes && !opsError && <p role="status">Reading the league notices…</p>}
      {notes && (
        <>
          <section aria-label="League notices" data-endpoint="M-095">
            <h4>League notices</h4>
            {mayPost && <TrayLauncher tray="notice" label="Post an official notice" />}
            <Notes notes={notes.notices} empty="No official notices have been posted."
              withdrawable={withdrawable} />
          </section>
          <section aria-label="What’s New">
            <h4>What’s New</h4>
            {mayPost && <TrayLauncher tray="note" label="Post a league note" />}
            <h5>League notes</h5>
            <Notes notes={notes.notes} empty="No league notes have been posted."
              withdrawable={withdrawable} />
            <h5>What changed in the app</h5>
            <ul className="release-notes">
              {notes.releases.map(r => (
                <li key={r.date + r.title}>
                  <strong>{r.title}</strong> <time dateTime={r.date}>{day(r.date)}</time>
                  <p>{r.words}</p>
                </li>
              ))}
            </ul>
          </section>
          <section aria-label="Mail log" data-endpoint="M-093">
            <h4>Mail log</h4>
            <p>{notes.mail.words}</p>
            {notes.mail.recorded.length === 0 ? <p>No notices have been recorded.</p> : (
              <ol className="mail-log">
                {notes.mail.recorded.map(n => (
                  <li key={n.id}>
                    <span>{n.text}</span>
                    <p>{n.withdrawn ? 'Withdrawn' : 'Shown to owners'} · {n.who} ·{' '}
                      <time dateTime={n.when}>{when(n.when)}</time></p>
                  </li>
                ))}
              </ol>
            )}
          </section>
        </>
      )}
      {digests && (
        <section aria-label="Digest of changes">
          <h4>Digest of changes</h4>
          <p>One sentence for each change that took a hold or a typed reason, read from the ledger.</p>
          {digests.digests.length === 0 ? <p>No changes have been recorded that need a digest.</p> : (
            <ol className="league-digest">
              {digests.digests.map(d => (
                <li key={d.id}>{d.sentence}{' '}
                  <time dateTime={d.when}>{when(d.when)}</time></li>
              ))}
            </ol>
          )}
        </section>
      )}
    </DensityScreen>
  );
}
