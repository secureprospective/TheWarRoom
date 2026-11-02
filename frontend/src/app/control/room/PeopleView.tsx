import { useRoomSurface } from './useRoomSurface';
import { Act } from '../../commands/Act';
import { useState } from 'react';
import type { PeopleSeat } from '../../commissioner/people';
import { openTeam } from './openers';
import { Desk, Kv, NotYet, useDefaultInspector, useRoom } from './room';
import { seenAgo, signInOf } from './roomReads';

export function PeopleView() {
  const room = useRoom();
  const people = room.state.people;
  const [mode, setMode] = useState<'seats' | 'perm'>('seats');
  const [sel, setSel] = useState('');
  const seats = people?.seats ?? [];
  const rolesOf = (s: PeopleSeat) =>
    people?.roles
      .filter((r) => r.person === s.owner)
      .map((r) => r.role)
      .join(', ') ?? '';
  const count = (role: string) =>
    people?.roles.filter((r) => r.role.toLowerCase().includes(role)).length ?? 0;
  const openVotes = people?.votes.filter((v) => v.status === 'open') ?? [];
  useDefaultInspector(
    <>
      <h3>People</h3>
      <div className="sub">
        Who holds each team, and who holds each role. Role grants and invites are private; seat changes are
        visible to owners.
      </div>
    </>,
  );
  const half = (xs: PeopleSeat[]) => (
    <table className="cr-t cr-ppl">
      <thead>
        <tr>
          <th>TEAM</th>
          <th>SEAT</th>
          <th>ROLE</th>
          <th style={{ width: 50 }}>SEEN</th>
          <th style={{ width: 40 }} />
        </tr>
      </thead>
      <tbody>
        {xs.map((s) => (
          <tr key={s.franchise} className={`rw${sel === s.franchise ? ' sel' : ''}`} data-team={s.franchise}>
            <td>
              <Act
                verb="commissioner.surface"
                variant="surface"
                label={s.name}
                args={{ screen: 'room-people', action: 'open', value: s.franchise }}
              >
                {s.name}
              </Act>
            </td>
            <td className="seat">
              {s.caretaker ? (
                <span className="cr-st pend">CARETAKER</span>
              ) : s.owner ? (
                'owner seated'
              ) : (
                <span style={{ color: 'var(--amber)' }}>vacant</span>
              )}
            </td>
            <td style={{ color: 'var(--ink2)' }}>{rolesOf(s)}</td>
            <td className="seat" style={{ color: 'var(--ink3)' }}>
              {seenAgo(signInOf(room.data, s.owner)?.lastSeen, Date.now())}
            </td>
            <td className="seat" style={{ color: 'var(--ink3)' }}>
              {signInOf(room.data, s.owner)?.authenticator ? '2FA' : ''}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
  const seg = (
    <div className="cr-seg">
      <Act
        active={mode === 'seats'}
        verb="commissioner.surface"
        variant="surface"
        label={'SEATS & ROLES'}
        args={{ screen: 'room-people', action: 'view', value: 'seats' }}
      >
        SEATS &amp; ROLES
      </Act>
      <Act
        active={mode === 'perm'}
        verb="commissioner.surface"
        variant="surface"
        label={'PERMISSIONS'}
        args={{ screen: 'room-people', action: 'view', value: 'perm' }}
      >
        PERMISSIONS
      </Act>
    </div>
  );
  useRoomSurface('room-people', ({ action, value }) => {
    if (action === 'view' && (value === 'seats' || value === 'perm')) setMode(value);
    if (action === 'open') {
      setSel(value);
      openTeam(room, value);
    }
  });
  const mid = Math.ceil(seats.length / 2);
  if (mode === 'perm')
    return (
      <>
        <div className="cr-col" style={{ width: 300, flex: 'none' }}>
          {seg}
          <div className="cr-panel" style={{ flex: 1 }}>
            <div className="cr-ph">
              PERMISSION SETS
              <span className="sp" />
              <span className="ct">extra tasks, not a promotion</span>
            </div>
            <div className="cr-pb">
              <NotYet what="permission sets are ruled, not built (commish.permissions)" />
            </div>
          </div>
          <div className="cr-panel">
            <div className="cr-ph">BUILT-IN ROLES</div>
            <div className="cr-pb">
              <Kv rows={(people?.roles ?? []).map((r) => [r.role, r.person] as [string, string])} />
            </div>
          </div>
        </div>
        <div className="cr-col" style={{ flex: 1 }}>
          <div className="cr-panel" style={{ flex: 1 }}>
            <div className="cr-ph">WHAT IT ALLOWS</div>
            <div className="cr-pb">
              <NotYet what="no permission set exists to show" />
            </div>
          </div>
        </div>
      </>
    );
  return (
    <>
      <div className="cr-col" style={{ flex: 1 }}>
        {seg}
        <div className="cr-panel" style={{ flex: 1 }}>
          <div className="cr-ph">
            FRANCHISES &amp; SEATS
            <span className="sp" />
            <span className="ct">click a team</span>
          </div>
          <div className="cr-pb cr-two" style={{ flex: 1, alignItems: 'start' }}>
            {!people && <div className="cr-note">{room.state.error || 'Reading the people…'}</div>}
            {half(seats.slice(0, mid))}
            {half(seats.slice(mid))}
          </div>
        </div>
      </div>
      <div className="cr-col" style={{ width: 380, flex: 'none' }}>
        <Desk
          title="ROLES"
          sub={`${count('commissioner') - count('co-')} commish · ${count('co-')} co-commish · ${
            people?.dotCount ?? 0} DOT`}
          kv={[
            ['Your role', people?.yourRole || '—'],
            ['DOT', people?.dotStatus || '—'],
            ['Grants', 'private'],
          ]}
          acts={[
            ['GRANT ROLE', 'commish.role.grant'],
            ['REVOKE ROLE', 'commish.role.revoke'],
            ['DEPUTY ORDER', 'commish.deputy.order'],
          ]}
        />
        <Desk
          title="SUCCESSION"
          sub={openVotes.length ? `${openVotes.length} vote open` : 'no vote open'}
          kv={[
            ['Succession', people?.succession || '—'],
            ['Invites waiting', String(people?.invites.filter((i) => i.status === 'pending').length ?? 0)],
          ]}
          acts={[
            ['STEP DOWN', 'commish.succession.stepdown'],
            ['INACTIVITY RULES', 'commish.succession.inactivity.set'],
            ['OPEN VOTE-OUT', 'commish.succession.voteout.open'],
            ['OPEN ELECTION', 'commish.succession.election.open'],
          ]}
        />
        <Desk
          title="ABILITIES & AI"
          sub={`all ${seats.length} identical`}
          kv={[
            ['Owner abilities', <NotYet key="a" />],
            ['AI agents', 'none connected'],
          ]}
          acts={[
            ['SET ABILITIES', 'commish.abilities.set'],
            ['WHO SEES WHAT', 'commish.visibility.set'],
            ['CONNECT AN AI AGENT ›', 'agent'],
            ['PERMISSION SETS ›', 'commish.permissions'],
          ]}
        />
      </div>
    </>
  );
}
