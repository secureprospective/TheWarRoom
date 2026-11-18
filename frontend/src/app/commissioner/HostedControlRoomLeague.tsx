import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { hostedReads } from '../data/hostedReads';
import { createReadQuery } from '../data/query';
import type { TaggedReply } from '../data/session';
import { HQAct } from '../shell/HQAct';
import type { NativeRequest } from '../commands/native';
import { parseLedger, type Ledger } from './proxy';
import { parseNotes, type LeagueNotes } from './notes';
import { newKey } from './forms';
import { when } from '../control/room/room';
import type { HostedLeagueRoom, HostedPort } from './hostedRoom';
import { hostedWorkflow } from './api';
import { Review } from './HostedReview';

type LeaguePage = { ledger: Ledger; notices: LeagueNotes };
type LeaguePageState = { page?: LeaguePage };
type LedgerChip = '' | 'rules' | 'corrections' | 'people';
const chips: [string, LedgerChip | null][] = [
  ['ALL', ''], ['RULES', 'rules'], ['FIXES', 'corrections'], ['TRADES', null],
  ['PEOPLE', 'people'], ['VIA AI', null], ['SYSTEM', null],
];
const unavailable = 'This part of the Control Room is not on the server app yet.';

function createLeaguePageQuery(league: string, port: HostedPort) {
  return createReadQuery('commissioner.TargetCommissionerLedger', {
    initial: (): LeaguePageState => ({}),
    hosted: { session: hostedReads, schemaVersion: '1', dependencies: [{ kind: 'league' }],
      policy: 'memory' },
    read: async (args: [string, LedgerChip]): Promise<TaggedReply<LeaguePage>> => {
      const [scope, chip] = args;
      const ledgerRequest = chip
        ? port({ command: 'commish.ledger.filtered.read', mode: 'read', arguments: {
          league: scope, filter: { person: '', kind: chip, from: '', before: '' },
        } })
        : port({ command: 'commish.ledger.read', mode: 'read', arguments: { league: scope } });
      const [ledgerResponse, noticesResponse] = await Promise.all([
        ledgerRequest,
        port({ command: 'commish.notices.read', mode: 'read', arguments: { league: scope } }),
      ]);
      const ledger = parseLedger(ledgerResponse.data);
      const notices = parseNotes(noticesResponse.data);
      if (scope !== league || ledger.league !== league || notices.league !== league)
        throw new Error('League records belong to another league.');
      if (ledger.revision !== notices.revision || ledgerResponse.football !== noticesResponse.football)
        throw new Error('League records were read at different revisions. Refresh before relying on them.');
      const key = JSON.parse(hostedReads.key('commissioner.TargetCommissionerLedger', '1', args)) as unknown;
      if (!Array.isArray(key) || typeof key[1] !== 'string')
        throw new Error('The hosted league identity is unavailable. Refresh before reading league records.');
      return { data: { ledger, notices }, tags: {
        serverEpoch: key[1], league: { league: String(ledger.revision) },
        football: ledgerResponse.football, live: '', values: '',
      } };
    },
    accept: (page: LeaguePage) => ({ page }),
  });
}

export function HostedControlRoomLeague({ league, room, port = hostedWorkflow }: {
  league: string; room: HostedLeagueRoom; port?: HostedPort;
}) {
  const query = useMemo(() => createLeaguePageQuery(league, port), [league, port]);
  const state = query.use();
  const roomState = room.use();
  const [chip, setChip] = useState<LedgerChip>('');
  const [kind, setKind] = useState<'notice' | 'note' | 'newsletter'>('notice');
  const [text, setText] = useState('');
  const [pin, setPin] = useState(false);
  const [reason, setReason] = useState('');
  const awaitingReceipt = useRef(false);
  const page = state.page;

  const refresh = useCallback(() => {
    query.invalidate();
    void query.load([league, chip], { overlap: 'after' });
  }, [chip, league, query]);
  useEffect(() => {
    void query.load([league, chip]);
  }, [chip, league, query]);
  useEffect(() => () => query.dispose(), [query]);
  useEffect(() => {
    if (!awaitingReceipt.current || !roomState.message || roomState.pending || roomState.review) return;
    awaitingReceipt.current = false;
    setText('');
    setPin(false);
    setReason('');
    refresh();
  }, [roomState.message, roomState.pending, roomState.review, refresh]);

  const changeKind = kind === 'notice' || kind === 'note' ? kind : undefined;
  const title = changeKind === 'notice' ? 'Post an official notice' : 'Post a league note';
  const fieldLabel = changeKind === 'notice' ? 'Notice' : 'League note';
  const command = changeKind === 'notice' ? 'commish.notice.publish' : 'commish.whatsnew.post';
  const revisionMatches = page !== undefined
    && roomState.reading?.rules.revision === page.ledger.revision;
  const preview = async () => {
    if (!page || !changeKind || !revisionMatches || !text.trim() || !reason.trim()) return;
    const request = { command, mode: 'preview', arguments: {
      league, command, args: { note: { text: text.trim(), pin } }, reason: reason.trim(),
      expected: page.ledger.revision, request: newKey(), effective: 'now', visibility: 'league',
    } } as NativeRequest;
    awaitingReceipt.current = true;
    await room.review(request, title);
    if (!room.peek().review) awaitingReceipt.current = false;
  };
  const disabled = !room.canChange() || roomState.pending || !page || !revisionMatches
    || !text.trim() || !reason.trim();

  return <div className="cr-col hosted-control-league" data-testid="hosted-control-league">
    <div className="cr-panel">
      <div className="cr-ph">TELL THE LEAGUE</div>
      <div className="cr-pb">
        <div className="cr-seg" aria-label="League messages">
          <HQAct active={kind === 'notice'} onAction={() => setKind('notice')}>NOTICE</HQAct>
          <HQAct active={kind === 'note'} onAction={() => setKind('note')}>LEAGUE NOTE</HQAct>
          <HQAct active={kind === 'newsletter'} onAction={() => setKind('newsletter')}>NEWSLETTER</HQAct>
        </div>
        {changeKind ? <section className="league-form" aria-label={title}>
          <p>{changeKind === 'notice'
            ? 'Owners read official notices in the app. Nothing is emailed.'
            : 'Owners read league notes in What’s New. Nothing is emailed.'}</p>
          <label>{fieldLabel}
            <textarea rows={4} value={text} onChange={event => setText(event.target.value)} />
          </label>
          <label>Pin to the top
            <input type="checkbox" checked={pin} onChange={event => setPin(event.target.checked)} />
          </label>
          <label>Reason for this change
            <textarea value={reason} onChange={event => setReason(event.target.value)} />
          </label>
          {!room.canChange() && <p role="status">League writes are unavailable until the server
            reading is writable.</p>}
          {page && roomState.reading && !revisionMatches
            && <p role="status">League records changed. Refresh the Control Room before posting.</p>}
          <HQAct disabled={disabled} onAction={() => void preview()}>Preview {title.toLowerCase()}</HQAct>
          {roomState.error && <p role="alert">{roomState.error}</p>}
        </section> : <p role="status">{unavailable} Newsletter publishing has no hosted command.</p>}
        {roomState.review && <Review room={room} />}
      </div>
    </div>

    <div className="cr-panel">
      <div className="cr-ph">LEAGUE NOTICES & NOTES</div>
      <div className="cr-pb">
        {!page && <p role={state.error ? 'alert' : 'status'}>{state.error || 'Reading league notices…'}</p>}
        {page && <>
          {page.notices.notices.filter(notice => !notice.withdrawn).map(notice => <article key={notice.id}>
            <h4>{notice.pinned ? 'Pinned notice' : 'Notice'}</h4>
            <p>{notice.text}</p><small>{notice.who} · {when(notice.when)}</small>
          </article>)}
          {page.notices.notes.filter(note => !note.withdrawn).map(note => <article key={note.id}>
            <h4>{note.pinned ? 'Pinned league note' : 'League note'}</h4>
            <p>{note.text}</p><small>{note.who} · {when(note.when)}</small>
          </article>)}
          {page.notices.notices.filter(notice => !notice.withdrawn).length
            + page.notices.notes.filter(note => !note.withdrawn).length === 0
            && <p>No league notices or notes have been posted.</p>}
        </>}
      </div>
    </div>

    <div className="cr-panel">
      <div className="cr-ph">LEDGER · every change, who, why
        <span className="sp" /><span className="ct">Owners see this, minus role grants and invites</span>
      </div>
      <div className="cr-pb cr-led">
        <div className="cr-chips" aria-label="Ledger filters">
          {chips.map(([label, value]) => <HQAct key={label} className="cr-chip"
            selected={chip === value} disabled={value === null}
            onAction={() => { if (value !== null) setChip(value); }}>{label}</HQAct>)}
          <HQAct disabled={state.pending} onAction={refresh}>Refresh ledger</HQAct>
        </div>
        {state.error && <p role="alert">{state.error}</p>}
        {page?.ledger.entries.map(entry => <div key={entry.id}>
          <span className="t">{when(entry.when)}</span><span className="a">{entry.who}</span>
          <span>{entry.what}{entry.why ? ` · ${entry.why}` : ''}</span>
        </div>)}
        {page && page.ledger.entries.length === 0 && <p>No ledger entries match this filter.</p>}
      </div>
    </div>

    <div className="cr-note hosted-league-unported">
      {unavailable} Feed and engine controls, league creation or migration, and house-rule editing
      are not exposed in this tab. Use the hosted Rules tab for supported rule changes.
    </div>
  </div>;
}
