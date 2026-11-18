import { useEffect, useMemo, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { onHostedTopics } from '../data/hostedReads';
import { HostedRules } from './HostedRules';
import { HostedCommissionerActTray } from './CommissionerActTray';
import { HostedControlRoomJobs } from './HostedControlRoomJobs';
import { HostedControlRoomSituations } from './HostedControlRoomSituations';
import { HostedControlRoomSeason } from './HostedControlRoomSeason';
import { createHostedJobsQuery, createHostedLedgerQuery, hostedControlTabs,
  type HostedControlTab } from './hostedControlRoomData';
import { createHostedLeagueRoom, type HostedLeagueRoom, type HostedPort } from './hostedRoom';
import type { Job } from './jobs';
import type { ActDefinition } from './actTable';
import { hostedWorkflow } from '../control/nativeApi';
import './commissioner.css';
import '../control/room/controlRoom.css';

const notOnServer = 'This part of the Control Room is not on the server app yet.';

export function HostedControlRoom({ league, port, heldRoom }: {
  league: string; port?: HostedPort; heldRoom?: HostedLeagueRoom;
}) {
  const room = useMemo(() => heldRoom ?? createHostedLeagueRoom(league, port),
    [heldRoom, league, port]);
  const jobsQuery = useMemo(() => createHostedJobsQuery(league, port), [league, port]);
  const roomState = room.use();
  const jobsState = jobsQuery.use();
  const [tab, setTab] = useState<HostedControlTab>('jobs');
  const [definition, setDefinition] = useState<ActDefinition>();
  const [job, setJob] = useState<Job>();
  const [routeWords, setRouteWords] = useState('');

  useEffect(() => {
    const refreshJobs = () => {
      jobsQuery.invalidate();
      void jobsQuery.load([league], { overlap: 'after' });
    };
    if (!heldRoom) void room.refresh();
    refreshJobs();
    const unsubscribe = onHostedTopics(topics => {
      if (!topics.some(topic => topic.league === league)) return;
      refreshJobs();
      if (topics.some(topic => topic.kind === 'league') && !room.peek().pending)
        void room.refresh();
    });
    return () => {
      unsubscribe();
      if (!heldRoom) room.dispose();
      jobsQuery.dispose();
    };
  }, [heldRoom, jobsQuery, league, room]);

  const openTool = (next: ActDefinition, source?: Job) => {
    setDefinition(next);
    setJob(source);
    setRouteWords('');
    setTab('jobs');
  };
  const openSeasonTool = (next: ActDefinition) => {
    setDefinition(next);
    setJob(undefined);
    setRouteWords('');
  };
  const showRoute = (words: string) => {
    setDefinition(undefined);
    setJob(undefined);
    setRouteWords(words);
  };
  const navigateTo = (next: HostedControlTab) => {
    setTab(next);
    setDefinition(undefined);
    setJob(undefined);
    setRouteWords('');
  };
  const jobCount = jobsState.jobs?.jobs.length;

  return <section className="crx hosted-control-room" data-testid="hosted-control-room"
    aria-label="Hosted Control Room">
    <main className="work">
      <div className="ws-tabs" role="tablist" aria-label="Control Room">
        {hostedControlTabs.map(([id, label]) => <HQAct key={id} role="tab" selected={tab === id}
          label={label} onAction={() => navigateTo(id)}>
          {label}{id === 'jobs' && jobCount !== undefined
            ? <> <span className="cr-st">{jobCount}</span></> : null}
        </HQAct>)}
        <HQAct className="hosted-control-refresh" disabled={roomState.pending || jobsState.pending}
          onAction={() => { void room.refresh(); jobsQuery.invalidate(); void jobsQuery.load([league]); }}>
          Refresh
        </HQAct>
      </div>
      <section className="cr-main on hosted-control-content"
        aria-label={hostedControlTabs.find(([id]) => id === tab)?.[1]}>
        {routeWords && <p role="status" className="cr-note">{routeWords}</p>}
        {tab === 'jobs' && <div className="hosted-jobs-column">
          {jobsState.error && <p role="alert" className="cr-note">{jobsState.error}</p>}
          <HostedControlRoomJobs jobs={jobsState.jobs} onOpen={openTool} onUnavailable={showRoute} />
          {definition && <HostedCommissionerActTray definition={definition} league={league}
            room={room} port={port} />}
          {job && <p className="hosted-job-source">Opened from: {job.name}</p>}
        </div>}
        {tab === 'situations' && <HostedControlRoomSituations
          onTab={navigateTo} onTool={next => openTool(next)} onUnavailable={showRoute} />}
        {tab === 'rules' && roomState.reading
          ? <HostedRules rules={roomState.reading.rules} commissioner room={room} />
          : tab === 'rules' && <p role="status">{roomState.error || 'Reading league rules…'}</p>}
        {tab === 'season' && <div className="hosted-jobs-column">
          <HostedControlRoomSeason league={league} port={port} onTool={openSeasonTool} />
          {definition && <HostedCommissionerActTray definition={definition} league={league}
            room={room} port={port} />}
        </div>}
        {tab === 'server' && <HostedServerTab league={league} port={port} />}
        {!['jobs', 'situations', 'rules', 'season', 'server'].includes(tab)
          && <p className="cr-note hosted-unported">{notOnServer}</p>}
      </section>
    </main>
    <aside className="insp" aria-label="League health">
      <div className="cr-ins">
        <h3>League health</h3>
        <div className="lbl">CURRENT JOB COUNT</div>
        <div className="big">{jobCount === undefined ? '—' : jobCount}</div>
        <p className="sub">One count from the league jobs reading. No separate health
          producer is available here.</p>
        <div className="lbl">SERVER RULES</div>
        <p className="sub">{roomState.reading?.rules.name || roomState.error
          || 'Reading from the league server…'}</p>
      </div>
    </aside>
  </section>;
}

function HostedServerTab({ league, port = hostedWorkflow }: { league: string; port?: HostedPort }) {
  const ledgerQuery = useMemo(() => createHostedLedgerQuery(league, port), [league, port]);
  const ledgerState = ledgerQuery.use();
  useEffect(() => {
    void ledgerQuery.load([league]);
    return () => ledgerQuery.dispose();
  }, [league, ledgerQuery]);
  const ledger = ledgerState.ledger;
  const error = ledgerState.error;
  return <div className="cr-note" aria-label="Server status">
    <h3>Server</h3>
    <p>Commissioner view is read-only. Restore, feed controls and upgrades are operator-only.</p>
    <h4>Backups</h4>
    <p>Back up now is not available through the hosted server command path.</p>
    <p>Restore is operator-only and unavailable here.</p>
    <h4>Feeds and upgrades</h4>
    <p>Feed controls and upgrades are operator-only and unavailable here.</p>
    <h4>Ledger</h4>
    {error && <p role="alert">{error}</p>}
    {!ledger && !error && <p role="status">Reading server ledger…</p>}
    {ledger && (ledger.entries.length ? <ol>{ledger.entries.map(entry => <li key={entry.id}>
      <strong>{entry.what}</strong><p>By {entry.who} · {new Date(entry.when).toLocaleString()}</p>
      <p>Reason: {entry.why || 'No reason recorded'}</p><p>Safeguard: {entry.gravity}</p>
    </li>)}</ol> : <p>No changes are visible to you in this league yet.</p>)}
  </div>;
}
