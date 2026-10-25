import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { backupState, NEEDS_SIGN_IN } from './backupState';
import { LocalSignIn } from './LocalSignIn';

// BackupNow sits in Control Room › App. Until a local sign-in exists it shows the sign-in form in
// place of the button.
export function BackupNow() {
  const s = backupState.use();
  useEffect(() => { void backupState.seat(); }, []);
  if (s.hasAccount === undefined) return s.error ? <p role="alert">{s.error}</p> : null;
  return (
    <section className="backup-now" aria-label="Back up now">
      <h4>Back up now</h4>
      {s.hasAccount ? <>
        <p>Saves your league, its history and your sign-in now. The app also backs up once a day.</p>
        {s.error && <p role="alert">{s.error}</p>}
        {s.message && <p role="status">{s.message}</p>}
        <Act variant="button" verb="backup.now" args={{}} disabled={s.pending}
          label="Back up now">{s.pending ? 'Backing up…' : 'Back up now'}</Act>
      </> : <>
        <p>{NEEDS_SIGN_IN}</p>
        <LocalSignIn onSeat={seat => backupState.signedIn(seat.hasAccount)} />
      </>}
    </section>
  );
}
