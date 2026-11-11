import { useRoomSurface } from './useRoomSurface';
import { Act } from '../../commands/Act';
import { actsFor } from '../../commissioner/actTable';
import { countdown, type Job, type JobState } from '../../commissioner/jobs';
import { launch } from './guide';
import { openStatCard } from './openers';
import { Kv, NotYet, useDefaultInspector, plainStamps, useRoom, when, type Room } from './room';
import type { RoomData } from './roomReads';

const STATE: Record<JobState, [string, string]> = {
  live: ['st-live', 'NEEDS YOU'],
  upcoming: ['st-up', 'UPCOMING'],
  blocked: ['st-block', 'BLOCKED'],
  unbuilt: ['st-unbuilt', 'NOT BUILT'],
};
const jobWhen = (j: Job) =>
  j.closes ? countdown(j.closes, Date.now()) : j.opens ? `opens ${when(j.opens)}` : '—';

/** What a job's OPEN does: the stat queue and the trade review get their cards; a job whose control has
 *  forms opens the first one; a job about the rules opens the rules. */
export function openJobTool(room: Room, job: Job) {
  if (job.controls.includes('CM-031')) {
    openStatCard(room);
    return;
  }
  if (job.controls.includes('CM-037')) {
    openTradeCard(room);
    return;
  }
  const command = job.controls.flatMap((c) => actsFor(c))[0]?.command;
  if (command) {
    room.run(command);
    return;
  }
  room.go('rules');
}

export function openTradeCard(room: Room) {
  const trades = room.data.trades?.trades ?? [];
  room.card({
    title: `Trade review · ${trades.length} open`,
    wide: true,
    sub: 'The DOT decides; you get the override and the appeal ruling',
    body: (
      <>
        {!room.data.trades ? (
          <div className="cr-note">{room.data.errors.trades ?? 'Reading the trades…'}</div>
        ) : trades.length === 0 ? (
          <div className="cr-note ok">No trade is waiting for review.</div>
        ) : (
          <ul className="cr-sides" style={{ display: 'block', padding: 0 }}>
            {trades.map((t) => (
              <li
                key={t.id}
                style={{ listStyle: 'none', padding: '6px 0', borderBottom: '1px solid var(--rule)' }}
              >
                {t.words}
              </li>
            ))}
          </ul>
        )}
        <div className="cr-btns">
          <span className="sub">Votes belong to the DOT</span>
          <span style={{ flex: 1 }} />
          <Act
            className="cr-btn"
            verb="commissioner.surface"
            variant="surface"
            label={'REVIEW…'}
            args={{
              screen: 'control-room',
              action: 'open',
              value: 'trade.review',
            }}
          >
            REVIEW…
          </Act>
          <Act
            className="cr-hold"
            verb="commissioner.surface"
            variant="surface"
            label={'Open'}
            args={{
              screen: 'control-room',
              action: 'open',
              value: 'trade.override',
            }}
          >
            <i />
            <span>OVERRIDE…</span>
          </Act>
          <Act
            className="cr-hold"
            verb="commissioner.surface"
            variant="surface"
            label={'Open'}
            args={{
              screen: 'control-room',
              action: 'open',
              value: 'trade.appeal.rule',
            }}
          >
            <i />
            <span>RULE ON AN APPEAL…</span>
          </Act>
        </div>
        <div className="sub">Same card the owners see in War Room › Trades; you get the extra buttons.</div>
      </>
    ),
  });
}

function windowCloses(data: RoomData, action: string): string | undefined {
  const w = data.calendar?.clock.windows.find((x) => x.live && x.opens.some((o) => o.includes(action)));
  return w?.close as unknown as string | undefined;
}
function nextWindow(data: RoomData): string | undefined {
  const now = Date.now();
  const w = data.calendar?.clock.windows
    .filter((x) => !x.live && Date.parse(x.start as unknown as string) > now)
    .sort((a, b) => Date.parse(a.start as unknown as string) - Date.parse(b.start as unknown as string))[0];
  return w ? `${w.opens.join(', ')} · ${when(w.start as unknown as string)}` : undefined;
}

export function JobsView() {
  const room = useRoom();
  const { jobs } = room.state;
  const { data } = room;
  const live = jobs?.jobs.filter((j) => j.state === 'live') ?? [];
  const blocked = jobs?.jobs.filter((j) => j.state === 'blocked') ?? [];
  const recent = room.state.ledger?.entries.slice(0, 7) ?? [];
  const role = room.state.people?.yourRole;
  useDefaultInspector(
    <>
      <h3>Control Room</h3>
      <div className="sub">
        {live.length} things need you · {blocked.length} blocked
      </div>
      <div className="lbl">HOW TO FIND ANYTHING</div>
      <div className="sub">
        1 · Jobs come to you (this page).
        <br />2 · Situations: pick what happened, get the tools in order.
        <br />3 · Search box or <b>/</b>: any rule, team, job or fix.
        <br />4 · Type a command at the bottom.
      </div>
      <div className="lbl">SAFETY</div>
      <div className="sub">
        Every change previews who it affects, asks the right confirm (click · hold · typed reason), lands on
        the ledger owners can read, and can be undone.
      </div>
      <div className="lbl">LATEST ON THE LEDGER</div>
      <div className="hist">
        {recent.map((l) => (
          <div key={l.id}>
            <b style={{ color: 'var(--ink)' }}>{l.who}</b> · {plainStamps(l.what)}
            <br />
            <span className="mono" style={{ fontSize: 10.5 }}>
              {when(l.when)}
            </span>
          </div>
        ))}
      </div>
      <div className="lbl">YOUR ROLE</div>
      <div className="sub">{role || '—'}</div>
    </>,
    [jobs, room.state.ledger, role],
  );
  const openJob = (j: Job) =>
    room.inspect(
      <>
        <h3>{j.name}</h3>
        <div className="sub">{j.words}</div>
        <Kv
          rows={[
            ['State', STATE[j.state][1]],
            ['When', jobWhen(j)],
            ['Census', j.controls.join(' · ') || '—'],
          ]}
        />
        <div className="foot">
          <Act
            className="cr-btn pri"
            verb="commissioner.surface"
            variant="surface"
            label={'OPEN'}
            args={{ screen: 'room-jobs', action: 'open', value: j.id }}
          >
            OPEN
          </Act>
        </div>
      </>,
    );
  const row = (j: Job) => (
    <Act
      key={j.id}
      className="cr-job"
      attributes={{ 'data-job': j.id }}
      verb="commissioner.surface"
      variant="surface"
      label={j.name}
      args={{ screen: 'room-jobs', action: 'pick', value: j.id }}
    >
      <i className={`dot ${STATE[j.state][0]}`} />
      <span className="nm">
        <b>{j.name}</b>
        <span>{j.words}</span>
      </span>
      <span className="cnt" />
      <span className="when">{jobWhen(j)}</span>
    </Act>
  );
  const sec = (s: JobState, label: string) => {
    const g = jobs?.jobs.filter((j) => j.state === s) ?? [];
    return g.length ? (
      <>
        <div className="cr-grp">
          {label} · {g.length}
        </div>
        {g.map(row)}
      </>
    ) : null;
  };
  const launchers: Record<string, () => void> = {
    stat: () => openStatCard(room),
    trade: () => openTradeCard(room),
    waivers: () => room.go('season'),
    notice: () => room.go('league'),
    invite: () => room.go('people'),
    pick: () => room.go('picks'),
    fix: () => room.go('fix'),
    rule: () => room.go('rules'),
  };
  useRoomSurface('room-jobs', ({ action, value }) => {
    if (action === 'tray') launchers[value]?.();
    else {
      const job = jobs?.jobs.find((j) => j.id === value);
      if (job) {
        if (action === 'open') openJobTool(room, job);
        else openJob(job);
      }
    }
  });
  const deadline = data.calendar?.clock.tradeDeadline as unknown as string | undefined;
  const week = data.week?.current?.week ?? data.week?.nextWeek;
  const weeks = data.week?.schedule.games.length;
  return (
    <>
      <div className="cr-col" style={{ width: 250, flex: 'none' }}>
        <div className="cr-panel">
          <div className="cr-ph">QUICK ACTIONS</div>
          <div className="cr-pb cr-launch">
            {launch.map(([label, key]) => (
              <Act
                key={key}
                verb="commissioner.surface"
                variant="surface"
                label={label}
                args={{ screen: 'room-jobs', action: 'tray', value: key }}
                attributes={{
                  'data-launch': String(key),
                }}
              >
                {label}
                <span className="kb">›</span>
              </Act>
            ))}
          </div>
        </div>
        <div className="cr-panel" style={{ flex: 1 }}>
          <div className="cr-ph">THIS WEEK</div>
          <div className="cr-pb">
            <Kv
              rows={[
                ['Week', week ? `${week}${weeks ? ` of ${weeks}` : ''}` : <NotYet key="w" />],
                ['Lineups lock', windowCloses(data, 'lineup') ? when(windowCloses(data, 'lineup')) : '—'],
                ['Waivers run', windowCloses(data, 'waiver') ? when(windowCloses(data, 'waiver')) : '—'],
                ['Trade deadline', deadline ? when(deadline) : 'not set'],
                ['Next window', nextWindow(data) ?? '—'],
                ['Last backup', <NotYet key="b" />],
              ]}
            />
          </div>
        </div>
      </div>
      <div className="cr-col" style={{ flex: 1 }}>
        <div className="cr-panel" style={{ flex: 1 }}>
          <div className="cr-ph">
            JOBS
            <span className="sp" />
            <span className="ct">live first · then upcoming · blocked · not built</span>
          </div>
          <div className="cr-pb cr-two" style={{ paddingTop: 0, flex: 1 }}>
            {!jobs && <div className="cr-note">{room.state.landingError || 'Reading the jobs…'}</div>}
            <div>{sec('live', 'NEEDS YOU')}</div>
            <div>
              {sec('upcoming', 'UPCOMING')}
              {sec('blocked', 'BLOCKED')}
              {sec('unbuilt', 'NOT BUILT')}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
