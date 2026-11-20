// Desktop v3 Control Room, League tab: post official notices or league notes, read posted
// notices and notes, and inspect the filtered league ledger. Shares all reads and state with
// HostedControlRoomLeague via useHostedLeague.
import type { ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { hostedLeagueUnavailable, ledgerChips, useHostedLeague }
  from '../commissioner/HostedControlRoomLeague';
import { Review } from '../commissioner/HostedReview';
import type { LeagueNotes, Note } from '../commissioner/notes';
import { memberDisplayName, type People } from '../commissioner/people';
import type { LedgerEntry } from '../commissioner/proxy';
import { when } from '../control/room/room';
import type { ControlBodyProps } from './ControlRoomV3';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;

function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

/** Active (non-withdrawn) notices followed by active league notes, preserving server order. */
export function activeNoticesAndNotes(reading: LeagueNotes | undefined): Note[] {
  if (!reading) return [];
  return [
    ...reading.notices.filter(notice => !notice.withdrawn),
    ...reading.notes.filter(note => !note.withdrawn),
  ];
}

/** Tag label for a posted notice or league note row. */
export function noticeKindTag(note: Pick<Note, 'kind' | 'pinned'>): string {
  if (note.kind === 'notice') return note.pinned ? 'PINNED NOTICE' : 'NOTICE';
  return note.pinned ? 'PINNED NOTE' : 'LEAGUE NOTE';
}

/** Actor display name and optional reason for a ledger entry row. */
export function ledgerEntryDetail(
  entry: Pick<LedgerEntry, 'who' | 'why'>,
  people?: Pick<People, 'seats'>,
): string {
  const actor = memberDisplayName(entry.who, people);
  return entry.why ? `${actor} · ${entry.why}` : actor;
}

export function CtrlLeagueV3({ c, league, port, badge, people }: ControlBodyProps) {
  const {
    chip, setChip, ledgerState, ledger, noticesState, notices, roomState,
    kind, setKind, text, setText, pin, setPin, reason, setReason, refresh,
    changeKind, title, fieldLabel, revisionMatches, preview, disabled,
  } = useHostedLeague(league, c.room, port);

  const items = activeNoticesAndNotes(notices);
  const rowWrap = { height: 'auto', minHeight: 30, padding: '6px 10px', whiteSpace: 'normal',
    alignItems: 'start' } as const;

  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1fr) minmax(0,1.25fr)',
    gridTemplateRows: 'minmax(0,1fr) minmax(0,1fr)' }}>
    <Panel title="Tell the league" badge={badge} style={{ gridRow: '1/2' }}
      foot="in-app only · never emailed">
      <div className="fbar" aria-label="League messages">
        <HQAct className="chip" selected={kind === 'notice'}
          onAction={() => setKind('notice')}>NOTICE</HQAct>
        <HQAct className="chip" selected={kind === 'note'}
          onAction={() => setKind('note')}>LEAGUE NOTE</HQAct>
        <HQAct className="chip" selected={kind === 'newsletter'}
          onAction={() => setKind('newsletter')}>NEWSLETTER</HQAct>
      </div>
      {changeKind ? <section className="league-form" aria-label={title}
        style={{ padding: '8px 10px', display: 'grid', gap: '8px' }}>
        <p className="d" style={{ margin: 0 }}>{changeKind === 'notice'
          ? 'Owners read official notices in the app. Nothing is emailed.'
          : 'Owners read league notes in What’s New. Nothing is emailed.'}</p>
        <label style={{ display: 'grid', gap: '4px' }}>{fieldLabel}
          <textarea rows={3} value={text} onChange={event => setText(event.target.value)} />
        </label>
        <label style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
          <input type="checkbox" checked={pin} onChange={event => setPin(event.target.checked)} />
          Pin to the top
        </label>
        <label style={{ display: 'grid', gap: '4px' }}>Reason for this change
          <textarea rows={2} value={reason} onChange={event => setReason(event.target.value)} />
        </label>
        {!c.room.canChange() && <St><span role="status">
          League writes are unavailable until the server reading is writable.</span></St>}
        {ledger && roomState.reading && !revisionMatches && <St><span role="status">
          League records changed. Refresh the Control Room before posting.</span></St>}
        <div><HQAct className="btn sm" disabled={disabled} onAction={() => void preview()}>
          Preview {title.toLowerCase()}</HQAct></div>
        {roomState.error && <St><span role="alert">{roomState.error}</span></St>}
      </section> : <St><span role="status">
        {hostedLeagueUnavailable} Newsletter publishing has no hosted command.</span></St>}
      {roomState.review && <Review room={c.room} />}
      <St><span role="status">{hostedLeagueUnavailable} Feed and engine controls, league creation or
        migration, and house-rule editing are not exposed in this tab. Use the hosted Rules tab for
        supported rule changes.</span></St>
    </Panel>

    <Panel title="League notices & notes" badge={badge} style={{ gridRow: '2/3' }}
      tag={notices ? <span className="tag o">{items.length}</span> : null}
      foot="in-app only · never emailed">
      {!notices && <St><span role={noticesState.error ? 'alert' : 'status'}>
        {noticesState.error || 'Reading league notices…'}</span></St>}
      {notices && items.length === 0 && <St>No league notices or notes have been posted.</St>}
      {items.map(item => <div key={item.id} className="row"
        style={{ ...rowWrap, gridTemplateColumns: 'auto minmax(0,1fr) auto' }}>
        <span className={item.pinned ? 'tag o' : 'tag'}>{noticeKindTag(item)}</span>
        <span>{item.text}</span>
        <span className="m">{memberDisplayName(item.who, people)} · {when(item.when)}</span>
      </div>)}
    </Panel>

    <Panel title="Ledger" badge={badge} style={{ gridRow: '1/3' }}
      tag={<>
        {ledger && <span className="tag o">{ledger.entries.length}</span>}
        <HQAct className="btn sm" disabled={ledgerState.pending || noticesState.pending}
          onAction={refresh}>Refresh ledger</HQAct>
      </>}
      foot="Owners see this, minus role grants and invites">
      <div className="fbar" aria-label="Ledger filters">
        {ledgerChips.map(([label, value]) => <HQAct key={label} className="chip"
          selected={chip === value} disabled={value === null}
          onAction={() => { if (value !== null) setChip(value); }}>{label}</HQAct>)}
      </div>
      {(ledgerState.error || noticesState.error) &&
        <St><span role="alert">{ledgerState.error || noticesState.error}</span></St>}
      {!ledger && !ledgerState.error && <St><span role="status">Reading league ledger…</span></St>}
      {ledger && ledger.entries.length === 0 && <St>No ledger entries match this filter.</St>}
      {ledger?.entries.map(entry => <div key={entry.id} className="row"
        style={{ ...rowWrap, gridTemplateColumns: 'auto minmax(0,1fr) auto' }}>
        <span className="m">{when(entry.when)}</span>
        <span><b>{entry.what}</b> <span className="d">{ledgerEntryDetail(entry, people)}</span></span>
        <span className="tag">{entry.gravity}</span>
      </div>)}
    </Panel>
  </div></div>;
}
