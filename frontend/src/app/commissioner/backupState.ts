import { createStore } from 'zustand/vanilla';
import { useStore } from 'zustand';
import { object, text } from '../data/parseValues';
import { ledgerBindings as bindings } from './api';
import { parseSeat } from './LocalSignIn';

type Backup = { hasAccount?: boolean; pending: boolean; message: string; error: string };
const store = createStore<Backup>(() => ({ pending: false, message: '', error: '' }));
const failure = (cause: unknown) => (cause instanceof Error ? cause.message : String(cause));
export const NEEDS_SIGN_IN = 'Before the first backup, create a sign-in. A backup restored on another '
  + 'computer gives your league roles back only to that sign-in.';

// backupState is Back up now (backup.now). Go refuses a backup without a local sign-in; the screen
// asks for one first, so a person never meets that refusal.
export const backupState = {
  use: () => useStore(store),
  peek: store.getState,
  signedIn(hasAccount: boolean) { store.setState({ hasAccount, error: '' }); },
  async seat() {
    try {
      store.setState({ hasAccount: parseSeat(await bindings().TargetCommissionerSeat()).hasAccount });
    } catch (cause) { store.setState({ error: failure(cause) }); }
  },
  async run() {
    if (store.getState().pending) return;
    if (store.getState().hasAccount === undefined) await this.seat();
    if (!store.getState().hasAccount) {
      store.setState({ message: '', error: NEEDS_SIGN_IN });
      return;
    }
    store.setState({ pending: true, message: '', error: '' });
    try {
      const saved = object(await bindings().TargetBackupNow(), 'Backup', ['set', 'message']);
      store.setState({ message: text(saved.message, 'Backup message') });
    } catch (cause) {
      store.setState({ error: failure(cause) });
    } finally { store.setState({ pending: false }); }
  },
};
