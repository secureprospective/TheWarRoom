import { useEffect, useMemo, useState } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { HQAct } from '../shell/HQAct';
import { hostedCommandAvailable, type NativeRequest } from '../commands/native';
import { createReadQuery } from '../data/query';
import { hostedSnapshot, onHostedChange, onHostedTopics } from '../data/hostedReads';
import { hostedWorkflow, type HostedResponse } from '../control/nativeApi';
import { HostedCommissionerActTray } from './CommissionerActTray';
import { actTable, type ActDefinition } from './actTable';
import type { HostedLeagueRoom, HostedPort } from './hostedRoom';
import { parsePeople, type People, type PeopleSeat } from './people';

const panels = [
  ['seats', 'Seats'], ['roles', 'Roles & succession'], ['permissions', 'Permissions'],
] as const;
type Panel = (typeof panels)[number][0];
type SeatFilter = 'all' | 'vacant' | 'quiet' | 'staff';

function definition(command: string): ActDefinition | undefined {
  return actTable.find(action => action.command === command);
}

function createHostedPeopleQuery(league: string, port: HostedPort = hostedWorkflow) {
  return createReadQuery('commissioner.TargetCommissionerPeople', {
    initial: (): { people?: People } => ({}),
    read: async (scope: string): Promise<People> => {
      const response: HostedResponse = await port({ command: 'commish.people.read', mode: 'read',
        arguments: { league: scope } } as NativeRequest);
      const people = parsePeople(response.data);
      if (people.league !== league || scope !== league)
        throw new Error('People belong to another league.');
      return people;
    },
    accept: (people: People) => ({ people }),
  });
}

function ToolButton({ command, onTool }: {
  command: string; onTool(definition: ActDefinition): void;
}) {
  const action = definition(command);
  if (!action) return <span role="status">{command} has no hosted form.</span>;
  const available = hostedCommandAvailable(command);
  return <HQAct className="cr-btn" disabled={!available} label={action.label}
    attributes={{ 'data-people-command': command }} onAction={() => onTool(action)}>
    {action.label}
  </HQAct>;
}

function seatState(seat: PeopleSeat): string {
  if (seat.owner) return 'Owner seated';
  if (seat.caretaker) return 'Caretaker seated';
  return 'Vacant';
}

export function HostedControlRoomPeople({ league, room, port }: {
  league: string; room: HostedLeagueRoom; port?: HostedPort;
}) {
  const query = useMemo(() => createHostedPeopleQuery(league, port), [league, port]);
  const reading = query.use();
  const [panel, setPanel] = useState<Panel>('seats');
  const [filter, setFilter] = useState<SeatFilter>('all');
  const [search, setSearch] = useState('');
  const [action, setAction] = useState<ActDefinition>();
  const people = !reading.pending && !reading.error ? reading.people : undefined;
  const seats = people?.seats ?? [];
  const roles = people?.roles ?? [];
  const visibleSeats = useMemo(() => {
    const needle = search.trim().toLocaleLowerCase();
    return seats.filter(seat => {
      if (filter === 'vacant' && (seat.owner || seat.caretaker)) return false;
      if (filter === 'quiet' || filter === 'staff') return false;
      return !needle || [seat.name, seat.abbreviation, seat.owner, seat.caretaker]
        .some(value => value?.toLocaleLowerCase().includes(needle));
    });
  }, [filter, search, seats]);
  const vacantCount = seats.filter(seat => !seat.owner && !seat.caretaker).length;

  useEffect(() => {
    const refresh = () => {
      query.invalidate();
      void query.load([league], { overlap: 'after' });
    };
    refresh();
    let identity = hostedSnapshot();
    const unsubscribeIdentity = onHostedChange(() => {
      const next = hostedSnapshot();
      if (next.blocked) query.reset();
      else if (identity.blocked || next.key !== identity.key) {
        query.reset();
        refresh();
      }
      identity = next;
    });
    const unsubscribeTopics = onHostedTopics(topics => {
      if (topics.some(topic => topic.league === league)) refresh();
    });
    return () => {
      unsubscribeIdentity();
      unsubscribeTopics();
      query.dispose();
      room.cancel();
    };
  }, [league, query, room]);

  return <div className="cr-col hosted-people" aria-label="People" style={{ flex: 1, overflowY: 'auto' }}>
    <div className="hosted-people-toolbar">
      <span>League people from the server</span>
      <HQAct disabled={reading.pending} onAction={() => {
        query.invalidate(); void query.load([league]);
      }}>Refresh people</HQAct>
    </div>
    <div className="cr-seg" role="tablist" aria-label="People sections">
      {panels.map(([id, label]) => <HQAct key={id} role="tab" selected={panel === id}
        label={label} onAction={() => setPanel(id)}>{label}</HQAct>)}
    </div>
    {reading.error && <FailureAlert
      message={reading.error}
      code={reading.errorCode}
      className="cr-note"
      league={league}
      screen="Control Room · People"
      route="control/people"
      port={port}
    />}
    {!people && !reading.error && <p role="status" className="cr-note">
      {reading.pending ? 'Reading people from the league server…' : 'People have not been read yet.'}
    </p>}
    {people && panel === 'seats' && <>
      <section className="cr-panel" aria-label="Franchises and seats">
        <div className="cr-ph">FRANCHISES &amp; SEATS<span className="sp" />
          <span className="ct">{visibleSeats.length} of {seats.length}</span>
        </div>
        <div className="cr-pb">
          <label className="hosted-people-search">Search seats
            <input aria-label="Search seats" value={search}
              onChange={event => setSearch(event.target.value)} placeholder="Team or owner" />
          </label>
          <div className="cr-chips" role="group" aria-label="Seat filters">
            <HQAct className="cr-chip" selected={filter === 'all'} onAction={() => setFilter('all')}>
              All <span className="c">{seats.length}</span>
            </HQAct>
            <HQAct className="cr-chip" selected={filter === 'vacant'}
              onAction={() => setFilter('vacant')}>
              Vacant <span className="c">{vacantCount}</span>
            </HQAct>
            <HQAct className="cr-chip" disabled title="Last-seen activity is not in this server read">
              Quiet · unavailable
            </HQAct>
            <HQAct className="cr-chip" disabled title="Staff identities are not in this server read">
              Staff · unavailable
            </HQAct>
          </div>
          <p className="cr-note">Quiet / last-seen and staff status are not returned by
            commish.people.read. These filters stay unavailable until their server reads exist.</p>
          <div className="hosted-people-table">
            <table className="cr-t">
              <thead><tr><th>TEAM</th><th>SEAT</th><th>OWNER</th><th>ROLE</th></tr></thead>
              <tbody>{visibleSeats.map(seat => <tr key={seat.franchise} data-seat={seat.franchise}>
                <td title={`${seat.name} (${seat.abbreviation})`}>{seat.name}</td>
                <td>{seatState(seat)}</td>
                <td title={seat.owner || seat.caretaker || ''}>{seat.owner || seat.caretaker || '—'}</td>
                <td>{roles.filter(role => role.person === seat.owner).map(role => role.role)
                  .join(', ') || '—'}</td>
              </tr>)}</tbody>
            </table>
            {visibleSeats.length === 0 && <p role="status">No seats match this search.</p>}
          </div>
        </div>
      </section>
      <section className="cr-panel" aria-label="Seat tools">
        <div className="cr-ph">SEAT TOOLS<span className="sp" />
          Invitations are private; changes go through server review.</div>
        <div className="cr-pb cr-btns">
          {['commish.seat.invite', 'commish.seat.caretaker', 'commish.seat.replace',
            'commish.seat.remove'].map(command => <ToolButton key={command} command={command}
              onTool={setAction} />)}
        </div>
      </section>
    </>}
    {people && panel === 'roles' && <>
      <section className="cr-panel" aria-label="League roles">
        <div className="cr-ph">LEAGUE ROLES<span className="sp" />
          {roles.filter(role => role.role === 'co-commissioner').length} co-commissioners</div>
        <div className="cr-pb">
          {roles.length === 0 ? <p>No league roles are listed.</p> :
            <table className="cr-t"><thead><tr><th>ROLE</th><th>PERSON</th></tr></thead>
              <tbody>{roles.map((role, index) => <tr key={`${role.person}:${role.role}:${index}`}>
                <td>{role.role}</td><td>{role.person}</td>
              </tr>)}</tbody></table>}
          <div className="cr-btns">
            {['commish.role.grant', 'commish.role.revoke', 'commish.deputy.order'].map(command =>
              <ToolButton key={command} command={command} onTool={setAction} />)}
          </div>
        </div>
      </section>
      <section className="cr-panel" aria-label="Succession">
        <div className="cr-ph">SUCCESSION<span className="sp" />
          {people.votes.filter(vote => vote.status === 'open').length} open votes</div>
        <div className="cr-pb">
          <p>{people.succession || '—'}</p>
          <p>{people.invites.filter(invite => invite.status === 'pending').length} pending invitations</p>
          <div className="cr-btns">
            {['commish.succession.stepdown', 'commish.succession.inactivity.set',
              'commish.succession.voteout.open', 'commish.succession.election.open',
              'commish.succession.periods.set'].map(command => <ToolButton key={command}
              command={command} onTool={setAction} />)}
          </div>
        </div>
      </section>
    </>}
    {people && panel === 'permissions' && <>
      <section className="cr-panel" aria-label="Owner abilities">
        <div className="cr-ph">OWNER ABILITIES<span className="sp" /></div>
        <div className="cr-pb">
          <p>The people read does not include each franchise’s current ability values. The server
            previews the current value before any change.</p>
          <ToolButton command="commish.abilities.set" onTool={setAction} />
        </div>
      </section>
      <section className="cr-panel" aria-label="Deputy permission sets">
        <div className="cr-ph">DEPUTY PERMISSION SETS<span className="sp" /></div>
        <div className="cr-pb">
          <p>Custom permission sets and their six-permission editor are not available in this server
            app yet (D65). Effective deputy permissions come from commish.people.read.</p>
          {people.actors.length === 0 ? <p role="status">Actor permissions are not included in this
            reading for your role.</p> : <table className="cr-t">
            <thead><tr><th>PERSON</th><th>EFFECTIVE DEPUTY PERMISSIONS</th></tr></thead>
            <tbody>{people.actors.map(actor => <tr key={actor.person}>
              <td>{people.seats.find(seat => seat.owner === actor.person)?.name ?? actor.person}</td>
              <td>{actor.effectivePermissions.join(' · ') || 'No deputy permissions'}</td>
            </tr>)}</tbody>
          </table>}
        </div>
      </section>
    </>}
    {action && <section className="cr-panel hosted-people-tool" aria-label={action.label}>
      <div className="cr-ph">{action.label}<span className="sp" />
        <HQAct onAction={() => { room.cancel(); setAction(undefined); }}>Close tool</HQAct>
      </div>
      <div className="cr-pb">
        <HostedCommissionerActTray definition={action} league={league} room={room} port={port} />
      </div>
    </section>}
  </div>;
}
