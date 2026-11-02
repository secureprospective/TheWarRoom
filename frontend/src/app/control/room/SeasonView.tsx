import type { leagueclock } from '../../../../wailsjs/go/models';
import { Desk, ruleValue, useDefaultInspector, useRoom, when } from './room';
import { teamName } from './roomReads';

const MONTHS = ['MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC', 'JAN', 'FEB'];
const DAY = 86_400_000;
const iso = (t: unknown) => (typeof t === 'string' ? t : '');

/** The league year starts Mar 1 (the mock's axis); a window is placed by its own instants. */
export function placeWindows(windows: leagueclock.ActionWindow[], now: number) {
  const d = new Date(now);
  const year = d.getMonth() >= 2 ? d.getFullYear() : d.getFullYear() - 1;
  const start = Date.UTC(year, 2, 1);
  const span = Date.UTC(year + 1, 2, 1) - start;
  const pct = (t: number) => Math.min(100, Math.max(0, ((t - start) / span) * 100));
  const lanes: number[] = [];
  return {
    today: pct(now),
    items: windows.map(w => {
      const a = Date.parse(iso(w.start));
      const b = w.close ? Date.parse(iso(w.close)) : a + DAY;
      const left = pct(a), right = Math.max(pct(b), left + 2);
      let lane = lanes.findIndex(end => end <= left);
      if (lane < 0) { lane = lanes.length; lanes.push(right); } else lanes[lane] = right;
      return { id: w.id, label: `${w.opens.join(', ')} ${when(iso(w.start))}`, left, width: right - left,
        lane: lane % 4, live: w.live };
    }),
  };
}

export function SeasonView() {
  const room = useRoom();
  const { data } = room;
  const rv = (f: string) => ruleValue(room, f);
  const windows = data.calendar?.clock.windows ?? [];
  const cal = placeWindows(windows, Date.now());
  const pending = room.state.rules?.sections.flatMap(s => s.rows).filter(r => r.pending.length) ?? [];
  const week = data.week?.current?.week ?? data.week?.nextWeek;
  const games = data.schedule?.published.find(w => w.week === week)?.games ?? [];
  const missing = (source: keyof typeof data.errors) => data.errors[source];
  useDefaultInspector(<><h3>Season desk</h3><div className="sub">Running the year: the calendar on top, the machinery
    below. Rules say how things work; this is where you run them or step in.</div>
    <div className="lbl">AUTOMATIC</div><div className="sub">Waivers and bids resolve themselves at window close.
      These buttons are the manual trigger and the overrides.</div></>);
  const rows3 = (cells: string[][]) => cells.map((c, i) => <div key={i}>{[0, 1, 2].map(k => <span key={k}>{c[k] ?? ''}</span>)}</div>);
  return <div className="cr-col" style={{ flex: 1 }}>
    <div className="cr-panel"><div className="cr-ph">LEAGUE YEAR · click a window to edit<span className="sp" />
      <span className="ct">timezone {rv('calendar.timezone')} · owners see their own time</span></div>
      <div className="cr-pb"><div className="cr-cal">
        {MONTHS.map((m, i) => <div key={m} className="mo" style={{ left: `${(i / 12) * 100}%` }}>{m}</div>)}
        {cal.items.map(w => <div key={w.id} className={`w${w.live ? ' ins' : ''}`} data-win={w.id} title={w.label}
          style={{ left: `${w.left}%`, width: `${w.width}%`, top: 18 + w.lane * 26 }}
          onClick={() => room.rule('calendar.windows')}>{w.label}</div>)}
        {!windows.length && <div className="cr-na" style={{ position: 'absolute', top: 40, left: 8 }}>
          {missing('calendar') ?? 'No windows in the league calendar.'}</div>}
        <div className="today" style={{ left: `${cal.today}%` }} />
      </div></div></div>
    <div className="cr-desk">
      <Desk title="WAIVERS" sub={`${data.waivers?.claims.length ?? 0} claims · ${data.waivers?.windows.length ?? 0} players`}
        kv={[['Window', rv('waivers.duration')], ['Priority', rv('waivers.method')], ['Trigger', rv('waivers.trigger')]]}
        acts={[['RUN NOW', 'waivers.process'], ['SET ORDER BY HAND', 'waivers.order.set'],
          ['AUTO ORDER', 'waivers.order.auto'], ['FIX AN AWARD', 'waivers.award.override']]}>
        <div className="cr-grp">CLAIMS</div>
        {data.waivers ? rows3(data.waivers.claims.slice(0, 4).map(c => [c.words])) : <div><span>{missing('waivers')}</span></div>}
      </Desk>
      <Desk title="BIDS" sub={`${data.bids?.players.length ?? 0} auctions open`}
        kv={[['Mode', `UFA ${rv('bids.ufa')} · RFA ${rv('bids.rfa')}`], ['Stand to win', rv('bids.standFor')],
          ['Most active bids', rv('bids.maximumActive')]]}
        acts={[['PROCESS CLEARED', 'bids.process'], ['FIX AN AWARD', 'bids.award.override'],
          ['CLOSE O-LINE AUCTION', 'oline.auction.close']]}>
        <div className="cr-grp">OPEN AUCTIONS</div>
        {data.bids ? rows3(data.bids.players.slice(0, 3).map(p => [p.words])) : <div><span>{missing('bids')}</span></div>}
      </Desk>
      <Desk title="DRAFT" sub={`${rv('draft.mode')} · clock ${rv('draft.clock')}`}
        kv={[['Order', rv('draft.orderSource')], ['Rounds', rv('draft.rounds')], ['Startup draft', rv('draft.startupRounds')]]}
        acts={[['SET ORDER', 'draft.order.set'], ['START', 'draft.start'], ['OPEN DRAFT ROOM TRAY ›', 'draft'],
          ['REVERT', 'draft.revert'], ['ARCHIVE', 'draft.archive']]}>
        <div className="cr-grp">{data.draft ? `${data.draft.season} ROUND 1` : 'ROUND 1 ORDER'}</div>
        {roundOne(data).length ? rows3(chunk(roundOne(data), 3))
          : <div><span>{missing('draft') ?? data.draft?.order.find(r => r.round === 1)?.words
            ?? 'Go has not ordered round 1 yet.'}</span></div>}
      </Desk>
      <Desk title="SCHEDULE" sub={`${rv('schedule.regularWeeks')} weeks · ${rv('schedule.pattern')}`}
        kv={[['Published weeks', String(data.schedule?.published.length ?? '—')],
          ['Draft weeks waiting', String(data.schedule?.draft.length ?? '—')]]}
        acts={[['EDIT A GAME', 'schedule.edit'], ['REGENERATE', 'schedule.generate'], ['PUBLISH', 'schedule.publish']]}>
        <div className="cr-grp">WEEK {week ?? '—'}</div>
        {games.length ? rows3(chunk(games.slice(0, 6).map(g => `${teamName(data, g.away)} @ ${teamName(data, g.home)}`), 3)
          .concat(games.length > 6 ? [[`+ ${games.length - 6} more games`]] : []))
          : <div><span>{missing('schedule') ?? 'No published games this week.'}</span></div>}
      </Desk>
      <Desk title="PLAYOFFS" sub={`${rv('playoffs.perConference')} per conference · wk ${rv('playoffs.startWeek')}–${rv('playoffs.finalWeek')}`}
        kv={[['Seeding', rv('playoffs.seeding')], ['Bye seed', rv('playoffs.byeSeed')], ['Home field', rv('playoffs.homeBonus')]]}
        acts={[['BUILD BRACKET', 'bracket.build'], ['EDIT', 'bracket.edit'], ['SEED A SLOT', 'bracket.seed'],
          ['ADVANCE (OVERRIDE)', 'bracket.advance'], ['TITLE LABEL', 'title.set']]}>
        <div className="cr-grp">SEEDS IF IT ENDED TODAY</div><div><span>—</span><span>no decided game yet</span><span /></div>
      </Desk>
      <Desk title="ROLLOVER" sub={rv('calendar.rollover')}
        kv={[['Rollover', room.state.rules?.rollover ?? '—'], ['Rule changes waiting', String(pending.length)]]}
        acts={[['SET INSTANT', 'calendar.rollover'], ['PREVIEW (DRY RUN)', 'season.rollover'],
          ['RUN ROLLOVER', 'season.rollover']]}>
        <div className="cr-grp">BEFORE YOU RUN IT</div>
        {rows3([['Rollover instant', rv('calendar.rollover')],
          ['Pending rule changes', pending.length ? `${pending.length} (${pending.map(p => p.label).join(', ')})` : 'none'],
          ['Backup taken first', 'automatic']])}
      </Desk>
    </div>
  </div>;
}
/** Round 1 as Go orders it: slot and owner, by team name. */
export function roundOne(data: ReturnType<typeof useRoom>['data']): string[] {
  const round = data.draft?.order.find(r => r.round === 1);
  return (round?.picks ?? []).map(p => `${p.slot} ${teamName(data, p.owner)}`);
}
function chunk<T>(xs: T[], n: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < xs.length; i += n) out.push(xs.slice(i, i + n));
  return out;
}
