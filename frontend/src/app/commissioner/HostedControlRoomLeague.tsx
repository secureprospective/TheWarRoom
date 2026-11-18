import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { onHostedTopics } from '../data/hostedReads';
import { createReadQuery } from '../data/query';
import { HQAct } from '../shell/HQAct';
import type { NativeRequest } from '../commands/native';
import { createHostedLedgerQuery } from './hostedControlRoomData';
import { parseNotes, type LeagueNotes } from './notes';
import { newKey } from './forms';
import { when } from '../control/room/room';
import type { HostedLeagueRoom, HostedPort } from './hostedRoom';
import { hostedWorkflow } from './api';
import { Review } from './HostedReview';

type LeaguePageState = { notices?: LeagueNotes };
type LedgerChip = '' | 'rules' | 'corrections' | 'people';
const chips: [string, LedgerChip | null][] = [
  ['ALL', ''], ['RULES', 'rules'], ['FIXES', 'corrections'], ['TRADES', null],
  ['PEOPLE', 'people'], ['VIA AI', null], ['SYSTEM', null],
];
const unavailable = 'This part of the Control Room is not on the server app yet.';

function createLeagueNoticesQuery(league: string, port: HostedPort) {
  return createReadQuery('commissioner.TargetCommissionerNotices', {
    initial: (): LeaguePageState => ({}),
    read: async (scope: string): Promise<LeagueNotes> => {
      const response = await port({ command: 'commish.notices.read', mode: 'read',
        arguments: { league: scope } });
      const notices = parseNotes(response.data);
      if (scope !== league || notices.league !== league)
        throw new Error('League notices belong to another league.');
      return notices;
    },
    accept: (notices: LeagueNotes) => ({ notices }),
  });
}

export function HostedControlRoomLeague({ league, room, port = hostedWorkflow }: {
  league: string; room: HostedLeagueRoom; port?: HostedPort;
}) {
  const [chip, setChip] = useState<LedgerChip>('');
  const ledgerQuery = useMemo(() => createHostedLedgerQuery(league, port), [league, port]);
  const ledgerState = ledgerQuery.use();
  const noticesQuery = useMemo(() => createLeagueNoticesQuery(league, port), [league, port]);
  const noticesState = noticesQuery.use();
  const roomState = room.use();
  const [kind, setKind] = useState<'notice' | 'note' | 'newsletter'>('notice');
  const [text, setText] = useState('');
  const [pin, setPin] = useState(false);
  const [reason, setReason] = useState('');
  const awaitingReceipt = useRef(false);
  const ledger = ledgerState.ledger;
  const notices = noticesState.notices;
  const refresh = useCallback(() => {
    ledgerQuery.invalidate();
    noticesQuery.invalidate();
    void ledgerQuery.load([league, chip], { overlap: 'after' });
    void noticesQuery.load([league], { overlap: 'after' });
  }, [chip, league, ledgerQuery, noticesQuery]);
  useEffect(() => {
    ledgerQuery.set({ ledger: undefined, error: '' });
    void ledgerQuery.load([league, chip]);
  }, [chip, league, ledgerQuery]);
  useEffect(() => {
    void noticesQuery.load([league]);
  }, [league, noticesQuery]);
  useEffect(() => () => ledgerQuery.dispose(), [ledgerQuery]);
  useEffect(() => () => noticesQuery.dispose(), [noticesQuery]);
  useEffect(() => onHostedTopics(topics => {
    if (!topics.some(topic => topic.kind === 'league' && topic.league === league)) return;
    void ledgerQuery.load([league, chip], { overlap: 'join' });
    noticesQuery.invalidate();
    void noticesQuery.load([league], { overlap: 'after' });
  }), [chip, league, ledgerQuery, noticesQuery]);
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
  const revisionMatches = ledger !== undefined
    && roomState.reading?.rules.revision === ledger.revision;
  const preview = async () => {
    if (!ledger || !changeKind || !revisionMatches || !text.trim() || !reason.trim()) return;
    const request = { command, mode: 'preview', arguments: {
      league, command, args: { note: { text: text.trim(), pin } }, reason: reason.trim(),
      expected: ledger.revision, request: newKey(), effective: 'now', visibility: 'league',
    } } as NativeRequest;
    awaitingReceipt.current = true;
    await room.review(request, title);
    if (!room.peek().review) awaitingReceipt.current = false;
  };
  const disabled = !room.canChange() || roomState.pending || !ledger || !revisionMatches
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
          {ledger && roomState.reading && !revisionMatches
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
        {!notices && <p role={noticesState.error ? 'alert' : 'status'}>
          {noticesState.error || 'Reading league notices…'}
        </p>}
        {notices && <>
          {notices.notices.filter(notice => !notice.withdrawn).map(notice => <article key={notice.id}>
            <h4>{notice.pinned ? 'Pinned notice' : 'Notice'}</h4>
            <p>{notice.text}</p><small>{notice.who} · {when(notice.when)}</small>
          </article>)}
          {notices.notes.filter(note => !note.withdrawn).map(note => <article key={note.id}>
            <h4>{note.pinned ? 'Pinned league note' : 'League note'}</h4>
            <p>{note.text}</p><small>{note.who} · {when(note.when)}</small>
          </article>)}
          {notices.notices.filter(notice => !notice.withdrawn).length
            + notices.notes.filter(note => !note.withdrawn).length === 0
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
          <HQAct disabled={ledgerState.pending || noticesState.pending}
            onAction={refresh}>Refresh ledger</HQAct>
        </div>
        {(ledgerState.error || noticesState.error) && <p role="alert">
          {ledgerState.error || noticesState.error}
        </p>}
        {!ledger && !ledgerState.error && <p role="status">Reading league ledger…</p>}
        {ledger?.entries.map(entry => <div key={entry.id}>
          <span className="t">{when(entry.when)}</span><span className="a">{entry.who}</span>
          <span>{entry.what}{entry.why ? ` · ${entry.why}` : ''}</span>
        </div>)}
        {ledger && ledger.entries.length === 0 && <p>No ledger entries match this filter.</p>}
      </div>
    </div>

    <div className="cr-note hosted-league-unported">
      {unavailable} Feed and engine controls, league creation or migration, and house-rule editing
      are not exposed in this tab. Use the hosted Rules tab for supported rule changes.
    </div>
  </div>;
}
