// Desktop v3 Control Room, after TheWarRoom-mock-v3-FINAL.html (screens.js control.*). The v3 shell draws
// the tab strip and rail leaves; this file draws the body for the chosen tab. Reads and actions are the
// hosted room's own (useHostedControlRoom), shared with the old room. A tab without a v3 body yet shows
// the old room's body for that tab, so every tool keeps working while the v3 bodies land one by one.
import { useCallback, useEffect, useMemo, useRef, type ReactNode } from 'react';
import { hostedSnapshot, onHostedChange, onHostedTopics } from '../data/hostedReads';
import type { HostedPort } from '../commissioner/hostedRoom';
import { HostedControlRoomBody, useHostedControlRoom, type HostedControlRoomState }
  from '../commissioner/HostedControlRoom';
import { createHostedPeopleQuery } from '../commissioner/HostedControlRoomPeople';
import type { People } from '../commissioner/people';
import type { HostedControlTab } from '../commissioner/hostedControlRoomData';
import { CtrlJobsV3 } from './CtrlJobsV3';
import { CtrlSituationsV3 } from './CtrlSituationsV3';
import { CtrlRulesV3 } from './CtrlRulesV3';
import { CtrlSeasonV3 } from './CtrlSeasonV3';
import { CtrlFixV3 } from './CtrlFixV3';
import { CtrlPicksV3 } from './CtrlPicksV3';
import { CtrlPeopleV3 } from './CtrlPeopleV3';
import { CtrlFeedbackV3 } from './CtrlFeedbackV3';
import { CtrlLeagueV3 } from './CtrlLeagueV3';
import { CtrlServerV3 } from './CtrlServerV3';
import '../commissioner/commissioner.css';
import '../control/room/controlRoom.css';

export type ControlBodyProps = { c: HostedControlRoomState; league: string; port?: HostedPort;
  badge: ReactNode; people?: People; peoplePending?: boolean; peopleError?: string;
  peopleErrorCode?: string; refreshPeople?: () => void };

/** v3 bodies by tab; a tab missing here falls back to the old room's body. */
const BODIES: Partial<Record<HostedControlTab, (p: ControlBodyProps) => ReactNode>> = {
  jobs: p => <CtrlJobsV3 {...p} />,
  situations: p => <CtrlSituationsV3 {...p} />,
  rules: p => <CtrlRulesV3 {...p} />,
  season: p => <CtrlSeasonV3 {...p} />,
  fix: p => <CtrlFixV3 {...p} />,
  picks: p => <CtrlPicksV3 {...p} />,
  people: p => <CtrlPeopleV3 {...p} />,
  feedback: p => <CtrlFeedbackV3 {...p} />,
  league: p => <CtrlLeagueV3 {...p} />,
  server: p => <CtrlServerV3 {...p} />,
};

export default function ControlRoomV3({ league, port, tab, setTab, badge }: { league: string;
  port?: HostedPort; tab: HostedControlTab; setTab: (t: HostedControlTab) => void; badge: ReactNode }) {
  const c = useHostedControlRoom(league, port);
  const peopleQuery = useMemo(() => createHostedPeopleQuery(league, port), [league, port]);
  const peopleReading = peopleQuery.use();
  const people = !peopleReading.error ? peopleReading.people : undefined;
  const refreshPeople = useCallback(() => {
    peopleQuery.invalidate();
    void peopleQuery.load([league], { overlap: 'after' });
  }, [league, peopleQuery]);
  useEffect(() => {
    refreshPeople();
    let identity = hostedSnapshot();
    const unsubscribeIdentity = onHostedChange(() => {
      const next = hostedSnapshot();
      if (next.blocked) peopleQuery.reset();
      else if (identity.blocked || next.key !== identity.key) { peopleQuery.reset(); refreshPeople(); }
      identity = next;
    });
    const unsubscribeTopics = onHostedTopics(topics => {
      if (topics.some(topic => topic.league === league)) refreshPeople();
    });
    return () => { unsubscribeIdentity(); unsubscribeTopics(); peopleQuery.dispose(); };
  }, [league, peopleQuery, refreshPeople]);
  // The shell owns the tab; the room may move it too (a job opens its tool on another tab).
  // On mount and whenever the shell picks a tab, the room follows the shell.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { if (c.tab !== tab) c.navigateTo(tab); }, [tab]);
  // After mount, a move the room makes itself (a job opening its tool on another tab) goes back to the shell.
  const settled = useRef(false);
  useEffect(() => {
    if (!settled.current) { settled.current = true; return; }
    if (c.tab !== tab) setTab(c.tab);
  }, [c.tab]); // eslint-disable-line react-hooks/exhaustive-deps
  const Body = BODIES[tab];
  return Body ? <>{Body({ c, league, port, badge, people, peoplePending: peopleReading.pending,
    peopleError: peopleReading.error, peopleErrorCode: peopleReading.errorCode, refreshPeople })}</>
    : <div className="wbody"><section className="crx hosted-control-room legacy-cr"
      style={{ height: '100%', overflow: 'auto' }}>
      <div className="cr-main on hosted-control-content">
        <HostedControlRoomBody c={{ ...c, tab }} league={league} port={port} /></div></section></div>;
}
