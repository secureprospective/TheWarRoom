// Desktop v3 Control Room, People tab, after TheWarRoom-mock-v3-FINAL.html (screens.js control.people):
// seats with search and filter, league health from the seats, and deputies with succession.
// Reads come from commish.people.read (the old People body's own read); every action is the old tool form.
import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { hostedCommandAvailable } from '../commands/native';
import { hostedSnapshot, onHostedChange, onHostedTopics } from '../data/hostedReads';
import { HostedCommissionerActTray } from '../commissioner/CommissionerActTray';
import { actTable, type ActDefinition } from '../commissioner/actTable';
import { createHostedPeopleQuery } from '../commissioner/HostedControlRoomPeople';
import { memberDisplayName, type People, type PeopleSeat } from '../commissioner/people';
import type { ControlBodyProps } from './ControlRoomV3';

export type SeatFilter = 'all' | 'vacant';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

/** Who sits in a seat: the owner, else the caretaker, else nobody. */
export function seatHolder(seat: PeopleSeat): string {
  return seat.owner || seat.caretaker || '';
}

export function seatState(seat: PeopleSeat): string {
  if (seat.owner) return 'Owner seated';
  if (seat.caretaker) return 'Caretaker seated';
  return 'Vacant';
}

/** Seats with no owner and no caretaker. */
export function vacantSeats(seats: PeopleSeat[]): PeopleSeat[] {
  return seats.filter(seat => !seat.owner && !seat.caretaker);
}

/** Seats passing the filter chip and the search words (team, abbreviation, owner, caretaker). */
export function filterSeats(seats: PeopleSeat[], filter: SeatFilter, search: string): PeopleSeat[] {
  const needle = search.trim().toLocaleLowerCase();
  return seats.filter(seat => {
    if (filter === 'vacant' && (seat.owner || seat.caretaker)) return false;
    return !needle || [seat.name, seat.abbreviation, seat.owner, seat.caretaker]
      .some(value => value?.toLocaleLowerCase().includes(needle));
  });
}

/** The roles a seat's owner holds, joined; an em dash when none. */
export function seatRoles(people: Pick<People, 'roles'>, seat: PeopleSeat): string {
  return people.roles.filter(role => seat.owner && role.person === seat.owner)
    .map(role => role.role).join(', ') || '—';
}

export type DeputyRow = { person: string; team: string; roles: string; permissions: string };
export type RoleRow = { person: string; name: string; role: string };

/** People with effective deputy permissions, named by their seat when they hold one. */
export function deputyRows(people: Pick<People, 'actors' | 'seats' | 'roles'>): DeputyRow[] {
  return people.actors.map(actor => ({
    person: actor.person,
    team: memberDisplayName(actor.person, people),
    roles: people.roles.filter(role => role.person === actor.person).map(role => role.role)
      .join(', ') || '—',
    permissions: actor.effectivePermissions.join(' · ') || 'No deputy permissions',
  }));
}

/** League roles paired with the holder's seat name, falling back to their id when unseated. */
export function roleRows(people: Pick<People, 'roles' | 'seats'>): RoleRow[] {
  return people.roles.map(role => ({
    person: role.person,
    name: memberDisplayName(role.person, people),
    role: role.role,
  }));
}

const definition = (command: string): ActDefinition | undefined =>
  actTable.find(action => action.command === command);

function ToolButtons({ commands, onTool }: { commands: string[]; onTool(d: ActDefinition): void }) {
  return <>{commands.map(command => {
    const action = definition(command);
    if (!action) return <span key={command} className="m" role="status">{command} has no hosted form.</span>;
    return <HQAct key={command} className="btn sm" disabled={!hostedCommandAvailable(command)}
      label={action.label} attributes={{ 'data-people-command': command }}
      onAction={() => onTool(action)}>{action.label}</HQAct>;
  })}</>;
}

export function CtrlPeopleV3({ c, league, port, badge }: ControlBodyProps) {
  const room = c.room;
  const query = useMemo(() => createHostedPeopleQuery(league, port), [league, port]);
  const reading = query.use();
  const [filter, setFilter] = useState<SeatFilter>('all');
  const [search, setSearch] = useState('');
  const [action, setAction] = useState<ActDefinition>();
  const people = !reading.pending && !reading.error ? reading.people : undefined;
  const seats = people?.seats ?? [];
  const vacant = vacantSeats(seats);
  const visible = filterSeats(seats, filter, search);

  useEffect(() => {
    const refresh = () => { query.invalidate(); void query.load([league], { overlap: 'after' }); };
    refresh();
    let identity = hostedSnapshot();
    const unsubscribeIdentity = onHostedChange(() => {
      const next = hostedSnapshot();
      if (next.blocked) query.reset();
      else if (identity.blocked || next.key !== identity.key) { query.reset(); refresh(); }
      identity = next;
    });
    const unsubscribeTopics = onHostedTopics(topics => {
      if (topics.some(topic => topic.league === league)) refresh();
    });
    return () => { unsubscribeIdentity(); unsubscribeTopics(); query.dispose(); room.cancel(); };
  }, [league, query, room]);

  const pending = !people && !reading.error;
  const waiting = <St>{reading.pending ? 'Reading people from the league server…'
    : 'People have not been read yet.'}</St>;
  const deputies = people ? deputyRows(people) : [];
  const roles = people ? roleRows(people) : [];
  return <div className="wbody scr scr-ctrl"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.4fr) minmax(0,1fr)',
    gridTemplateRows: 'minmax(0,1fr) minmax(0,1fr)' }}>
    <Panel title="People and seats" badge={badge} style={{ gridRow: '1/3' }}
      tag={people ? <span className="tag o">{seats.length}</span> : null}
      foot={people ? `${visible.length} of ${seats.length} seats · invitations are private`
        : 'League people from the server'}>
      <div className="fbar">
        <input aria-label="Search seats" value={search} placeholder="Team or owner"
          onChange={event => setSearch(event.target.value)} />
        <HQAct className="chip" selected={filter === 'all'} onAction={() => setFilter('all')}>
          {`All ${seats.length}`}</HQAct>
        <HQAct className="chip" selected={filter === 'vacant'} onAction={() => setFilter('vacant')}>
          {`Vacant ${vacant.length}`}</HQAct>
        <HQAct className="chip" disabled title="Last-seen activity is not in this server read">
          Quiet · unavailable</HQAct>
        <HQAct className="chip" disabled title="Staff identities are not in this server read">
          Staff · unavailable</HQAct>
        <HQAct className="btn sm" disabled={reading.pending} onAction={() => {
          query.invalidate(); void query.load([league]);
        }}>Refresh people</HQAct>
      </div>
      <p className="m">Quiet / last-seen and staff status are not returned by commish.people.read.
        These filters stay unavailable until their server reads exist.</p>
      {reading.error && <St><span role="alert">{reading.error}</span></St>}
      {pending && !reading.error && waiting}
      {people && <>
        {visible.length === 0 && <St>No seats match this search.</St>}
        {visible.map(seat => <div key={seat.franchise} className="row" data-seat={seat.franchise}
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto minmax(0,1fr) auto' }}>
          <span className="ell" title={`${seat.name} (${seat.abbreviation})`}><b>{seat.name}</b></span>
          <span className="tag">{seatState(seat)}</span>
          <span className="ell m" title={seatHolder(seat)}>{seatHolder(seat) || '—'}</span>
          <span className="m">{seatRoles(people, seat)}</span></div>)}
        <div className="sec">SEAT TOOLS</div>
        <div className="acts"><ToolButtons onTool={setAction} commands={['commish.seat.invite',
          'commish.seat.caretaker', 'commish.seat.replace', 'commish.seat.remove']} /></div>
      </>}
      {action && <section aria-label={action.label}>
        <div className="sec">{action.label}</div>
        <HQAct className="btn sm" onAction={() => { room.cancel(); setAction(undefined); }}>Close tool</HQAct>
        <HostedCommissionerActTray definition={action} league={league} room={room} port={port} />
      </section>}
    </Panel>
    <Panel title="League health" badge={badge} foot="orphan teams get a caretaker seat">
      {pending ? waiting : !people ? null : vacant.length === 0 ? <St>No orphan teams.</St>
        : vacant.map(seat => <div key={seat.franchise} className="row"
          style={{ gridTemplateColumns: 'auto minmax(0,1fr)' }}>
          <span className="tag">ORPHAN TEAM</span>
          <span className="ell">{seat.name}: no owner</span></div>)}
    </Panel>
    <Panel title="Deputies and succession" badge={badge}
      foot="deputies act in your name · every action they take is in the ledger">
      {pending ? waiting : !people ? null : <>
        <div className="sec">DEPUTIES · {deputies.length}</div>
        {deputies.length === 0 && <St>Actor permissions are not included in this reading for your
          role.</St>}
        {deputies.map(d => <div key={d.person} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto minmax(0,1.4fr)' }}>
          <span className="ell" title={d.person}><b>{d.team}</b></span><span className="tag">{d.roles}</span>
          <span className="ell m" title={d.permissions}>{d.permissions}</span></div>)}
        <div className="sec">ROLES · {roles.length}</div>
        {roles.length === 0 && <St>No league roles are listed.</St>}
        {roles.map((role, index) => <div key={`${role.person}:${role.role}:${index}`} className="row"
          style={{ gridTemplateColumns: 'auto minmax(0,1fr)' }}>
          <span className="tag">{role.role}</span>
          <span className="ell" title={role.person}>{role.name}</span></div>)}
        <div className="acts"><ToolButtons onTool={setAction} commands={['commish.role.grant',
          'commish.role.revoke', 'commish.deputy.order', 'commish.abilities.set']} /></div>
        <div className="sec">SUCCESSION</div>
        <p className="m">{people.succession || '—'}</p>
        <p className="m">
          {people.invites.filter(i => i.status === 'pending').length} pending invitations ·{' '}
          {people.votes.filter(v => v.status === 'open').length} open votes</p>
        <div className="acts"><ToolButtons onTool={setAction} commands={['commish.succession.stepdown',
          'commish.succession.inactivity.set', 'commish.succession.voteout.open',
          'commish.succession.election.open', 'commish.succession.periods.set']} /></div>
      </>}
    </Panel>
  </div></div>;
}
