import { fixes, pickActs, situations } from './guide';
import { routeCommand, unbuiltWords } from './route';
import { Kv, tabs, type Room, type Tab } from './room';

const mark = {
  B: null,
  R: <span className="cr-st plan">RULED · NOT BUILT</span>,
  G: <span className="cr-st need">GAP</span>,
} as const;

/** A situation's step: a branch ("Server › Feeds"), a rule, or a command with its existing form. */
export function openStep(room: Room, step: string) {
  room.card(null);
  if (step.includes('›')) {
    const branch = step.split('›')[0]!.trim().toUpperCase();
    const tab = tabs.find(([, l]) => l === branch)?.[0] as Tab | undefined;
    if (tab) { room.go(tab); return; }
  }
  const name = step.split(' ')[0]!;
  const rule = room.state.rules?.sections.flatMap(s => s.rows).find(r => r.field === name);
  if (rule) { room.go('rules', () => room.rule(rule.field)); return; }
  if (name === 'stats.queue') { room.go('jobs', () => openStatCard(room)); return; }
  if (routeCommand(name).kind === 'unbuilt') { room.toast(unbuiltWords(name)); return; }
  room.run(name);
}

export function openSituation(room: Room, i: number) {
  const s = situations[i]!;
  room.card({ title: s[1], sub: s[0], body: <>
    <div className="cr-steps">{s[2].map((st, k) => <div key={st[1]}>
      <span className="no">{k + 1}</span>
      <span>{st[0]} {mark[st[2]]}<br /><code>{st[1]}</code></span>
      <button className="cr-btn sm" disabled={st[2] !== 'B'} style={st[2] === 'B' ? undefined : { opacity: 0.4 }}
        data-step={st[1]} onClick={() => openStep(room, st[1])}>OPEN ›</button>
    </div>)}</div>
    <div className="cr-note">Still up to you: {s[3]}</div>
    <div className="cr-note ok">Check afterwards: {s[4]}</div>
  </> });
}

export function openFix(room: Room, i: number) {
  const f = fixes[i]!;
  room.inspect(<><h3>Fix · {f[0]}</h3><div className="sub">{f[1]}</div>
    <div className="lbl">COMMAND</div><div className="sub mono">{f[2]}</div></>);
  room.run(f[2]);
}

export function openPickAct(room: Room, i: number) {
  room.run(pickActs[i]![2]);
}

export function openTeam(room: Room, franchise: string) {
  const people = room.state.people;
  const seat = people?.seats.find(s => s.franchise === franchise);
  if (!seat) return;
  const roles = people!.roles.filter(r => r.person === seat.owner).map(r => r.role);
  const season = room.data.assets?.snapshot?.season;
  const picks = (room.data.assets?.snapshot?.picks ?? [])
    .filter(p => p.owner === franchise && !p.player && !p.skipped && (!season || p.season === season + 1)).length;
  room.inspect(<><div className="sub">Franchise</div><h3>{seat.name}</h3>
    <Kv rows={[['Seat', seat.caretaker ? 'caretaker' : seat.owner ? 'owner seated' : 'vacant'],
      ['Role', roles.join(', ') || 'owner'], ['Authenticator', <span key="a" className="cr-na">Not available yet</span>],
      ['Permission sets', <span key="p" className="cr-na">Not available yet</span>],
      [`Picks ${season ? season + 1 : ''}`.trim(), String(picks)]]} />
    <div className="foot">
      <button className="cr-btn" onClick={() => room.run('commish.franchise.edit')}>RENAME / ABBREVIATION</button>
      <button className="cr-btn" onClick={() => room.run('commish.seat.invite')}>INVITE OWNER</button>
      <button className="cr-btn" onClick={() => room.run('commish.seat.caretaker')}>SEAT ME AS CARETAKER</button>
      <button className="cr-btn" onClick={() => room.run('account.totp.reset')}>RESET AUTHENTICATOR <span
        className="cr-st plan">NOT BUILT</span></button>
      <button className="cr-btn g3" onClick={() => room.run('commish.seat.replace')}>REPLACE OWNER</button>
      <button className="cr-btn g3" onClick={() => room.run('commish.seat.remove')}>REMOVE OWNER</button>
      <button className="cr-btn" onClick={() => room.go('picks')}>SEE PICKS ›</button>
    </div></>);
}

/** The stat-correction card: the league's open stat queue, each change with its flip words. */
export function openStatCard(room: Room) {
  const q = room.data.statQueue;
  const open = q?.open ?? [];
  room.card({ title: `Stat corrections · ${open.length} waiting`,
    sub: 'Feed changes queue here; nothing changes a score until you approve', wide: true, body: <>
      {!q ? <div className="cr-note">{room.data.errors.statQueue ?? 'Reading the stat queue…'}</div> :
        open.length === 0 ? <div className="cr-note ok">{q.words || 'No stat change is waiting.'}</div> :
        <table className="cr-t"><thead><tr><th style={{ width: '24%' }}>PLAYER</th><th>STAT</th>
          <th style={{ width: 90 }}>WAS</th><th style={{ width: 90 }}>NOW</th><th style={{ width: '28%' }}>EFFECT</th>
        </tr></thead><tbody>{open.map(i => <tr key={i.id}>
          <td>{i.name}</td><td>{i.changes.map(c => c.label).join(', ')} wk {i.week}</td>
          <td className="v">{i.changes.map(c => c.before).join(', ')}</td>
          <td className="v">{i.changes.map(c => c.after).join(', ')}</td>
          <td style={i.flip ? { color: 'var(--amber)' } : undefined}>{i.flipWords || i.words}</td>
        </tr>)}</tbody></table>}
      <div className="cr-note">A flip changes a result: standings, seeds and waiver order follow. You need a typed
        reason for that one.</div>
      <div className="cr-btns">
        <button className="cr-btn pri" disabled={!open.length}
          onClick={() => room.run('commish.stat.queue.approve')}>APPROVE…</button>
        <button className="cr-btn" disabled={!open.length}
          onClick={() => room.run('commish.stat.queue.reject')}>REJECT…</button>
      </div></> });
}
