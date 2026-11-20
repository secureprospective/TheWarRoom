import { useCallback, useEffect, useMemo, useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { onHostedTopics } from '../data/hostedReads';
import { HostedRules } from './HostedRules';
import { HostedCommissionerActTray } from './CommissionerActTray';
import { HostedBackupNow } from './HostedBackupNow';
import { HostedControlRoomJobs } from './HostedControlRoomJobs';
import { HostedControlRoomPicks } from './HostedControlRoomPicks';
import { HostedControlRoomPeople } from './HostedControlRoomPeople';
import { HostedControlRoomFeedback } from './HostedControlRoomFeedback';
import { HostedControlRoomSituations } from './HostedControlRoomSituations';
import { HostedControlRoomFix } from './HostedControlRoomFix';
import { HostedControlRoomLeague } from './HostedControlRoomLeague';
import { HostedControlRoomSeason } from './HostedControlRoomSeason';
import { createHostedFixQuery, createHostedJobsQuery, createHostedLedgerQuery, hostedControlRoomNotOnServer,
  hostedControlTabs, type HostedControlTab } from './hostedControlRoomData';
import { createHostedLeagueRoom, type HostedLeagueRoom, type HostedPort } from './hostedRoom';
import type { ActValues } from './actBody';
import type { NativeRequest } from '../commands/native';
import { activeJobSnoozes, parseJobSnoozes, type Job, type JobSnooze, type JobSnoozes } from './jobs';
import type { ActDefinition } from './actTable';
import { hostedWorkflow } from '../control/nativeApi';
import './commissioner.css';
import '../control/room/controlRoom.css';

export function HostedControlRoom({ league, port, heldRoom }: {
  league: string; port?: HostedPort; heldRoom?: HostedLeagueRoom;
}) {
  const room = useMemo(() => heldRoom ?? createHostedLeagueRoom(league, port),
    [heldRoom, league, port]);
  const jobsQuery = useMemo(() => createHostedJobsQuery(league, port), [league, port]);
  const fixQuery = useMemo(() => createHostedFixQuery(league, port), [league, port]);
  const roomState = room.use();
  const jobsState = jobsQuery.use();
  const fixState = fixQuery.use();
  const [tab, setTab] = useState<HostedControlTab>('jobs');
  const [definition, setDefinition] = useState<ActDefinition>();
  const [openedTab, setOpenedTab] = useState<HostedControlTab>('jobs');
  const [initialValues, setInitialValues] = useState<ActValues>();
  const [job, setJob] = useState<Job>();
  const [routeWords, setRouteWords] = useState('');
  const [feedbackRefreshVersion, setFeedbackRefreshVersion] = useState(0);
  const [jobSnoozes, setJobSnoozes] = useState<JobSnoozes>();
  const [snoozeError, setSnoozeError] = useState('');
  const [snoozePending, setSnoozePending] = useState('');
  const [jobNow, setJobNow] = useState(Date.now());
  const preferencePort = port ?? hostedWorkflow;
  const loadJobSnoozes = useCallback(async (isCurrent: () => boolean = () => true) => {
    if (isCurrent()) { setJobSnoozes(undefined); setSnoozeError(''); }
    try {
      const response = await preferencePort({ command: 'job.snoozes.read', mode: 'read', arguments: {} });
      const state = parseJobSnoozes(response.data);
      if (isCurrent()) { setJobSnoozes(state); setSnoozeError(''); }
    } catch {
      if (isCurrent()) {
        setJobSnoozes(undefined);
        setSnoozeError('Your private job snoozes could not be read. Jobs are shown normally.');
      }
    }
  }, [preferencePort]);

  useEffect(() => {
    let active = true;
    setJobSnoozes(undefined);
    setSnoozeError('');
    void loadJobSnoozes(() => active);
    return () => { active = false; };
  }, [league, loadJobSnoozes]);

  useEffect(() => {
    const next = activeJobSnoozes(jobSnoozes, league, jobNow)
      .map(snooze => Date.parse(snooze.until)).sort((a, b) => a - b)[0];
    if (next === undefined) return;
    const delay = Math.min(2_147_483_647, Math.max(0, next - jobNow + 1));
    const timer = window.setTimeout(() => setJobNow(Date.now()), delay);
    return () => window.clearTimeout(timer);
  }, [jobNow, jobSnoozes, league]);

  useEffect(() => {
    const refreshJobs = () => {
      jobsQuery.invalidate();
      void jobsQuery.load([league], { overlap: 'after' });
    };
    const refreshFixes = () => {
      fixQuery.invalidate();
      void fixQuery.load([league], { overlap: 'after' });
    };
    if (!heldRoom) void room.refresh();
    refreshJobs();
    const unsubscribe = onHostedTopics(topics => {
      if (!topics.some(topic => topic.league === league)) return;
      refreshJobs();
      if (fixQuery.peek().fixes || fixQuery.peek().pending) refreshFixes();
      if (topics.some(topic => topic.kind === 'league') && !room.peek().pending)
        void room.refresh();
    });
    return () => {
      unsubscribe();
      if (!heldRoom) room.dispose();
      jobsQuery.dispose();
      fixQuery.dispose();
    };
  }, [fixQuery, heldRoom, jobsQuery, league, room]);

  const refreshFixes = () => {
    fixQuery.invalidate();
    void fixQuery.load([league], { overlap: 'after' });
  };
  const writeJobSnooze = async (request: NativeRequest, jobID: string) => {
    if (snoozePending) return;
    setSnoozePending(jobID);
    setSnoozeError('');
    try {
      const response = await preferencePort(request);
      setJobSnoozes(parseJobSnoozes(response.data));
    } catch {
      setSnoozeError('Could not update your private job snoozes. Refresh and try again.');
    } finally {
      setSnoozePending('');
    }
  };
  const snoozeJob = (source: Job, until: string) => void writeJobSnooze({
    command: 'job.snooze.set', mode: 'execute',
    arguments: { league, jobId: source.id, until },
  }, source.id);
  const wakeJob = (source: JobSnooze) => void writeJobSnooze({
    command: 'job.snooze.wake', mode: 'execute',
    arguments: { league: source.league, jobId: source.jobId },
  }, source.jobId);
  const openTool = (next: ActDefinition, source?: Job, destination: HostedControlTab = 'jobs',
    initial?: ActValues) => {
    setDefinition(next);
    setOpenedTab(destination);
    setInitialValues(initial);
    setJob(source);
    setRouteWords('');
    setTab(destination);
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
    setInitialValues(undefined);
    setJob(undefined);
    setRouteWords('');
    if (next === 'fix') refreshFixes();
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
        <HQAct className="hosted-control-refresh" disabled={roomState.pending || jobsState.pending
          || snoozePending !== ''}
          onAction={() => { void room.refresh(); jobsQuery.invalidate(); void jobsQuery.load([league]);
            void loadJobSnoozes(); if (tab === 'fix') refreshFixes();
            if (tab === 'feedback') setFeedbackRefreshVersion(version => version + 1); }}>
          Refresh
        </HQAct>
      </div>
      <section className="cr-main on hosted-control-content"
        aria-label={hostedControlTabs.find(([id]) => id === tab)?.[1]}>
        {routeWords && <p role="status" className="cr-note">{routeWords}</p>}
        {tab === 'jobs' && <div className="hosted-jobs-column">
          {jobsState.error && <p role="alert" className="cr-note">{jobsState.error}</p>}
          <HostedControlRoomJobs league={league} jobs={jobsState.jobs} snoozes={jobSnoozes} now={jobNow}
            snoozeError={snoozeError} pendingJob={snoozePending} onSnooze={snoozeJob} onWake={wakeJob}
            onOpen={openTool} onUnavailable={showRoute} />
          {definition && openedTab === 'jobs' && <HostedCommissionerActTray definition={definition}
            league={league} room={room} port={port} initial={initialValues} />}
          {job && <p className="hosted-job-source">Opened from: {job.name}</p>}
        </div>}
        {tab === 'situations' && <HostedControlRoomSituations
          onTab={navigateTo} onTool={next => openTool(next)} onUnavailable={showRoute} />}
        {tab === 'rules' && roomState.error && <p role="alert">{roomState.error}</p>}
        {tab === 'rules' && roomState.stale &&
          <p role="status">Rules are read-only until refreshed from the league server.</p>}
        {tab === 'rules' && roomState.message && <p role="status">{roomState.message}</p>}
        {tab === 'rules' && roomState.reading
          ? <HostedRules rules={roomState.reading.rules} commissioner room={room} />
          : tab === 'rules' && <p role="status">{roomState.error || 'Reading league rules…'}</p>}
        {tab === 'league' && <HostedControlRoomLeague key={league} league={league}
          room={room} port={port} />}
        {tab === 'season' && <div className="hosted-jobs-column">
          <HostedControlRoomSeason league={league} port={port} onTool={openSeasonTool} />
          {definition && <HostedCommissionerActTray definition={definition} league={league}
            room={room} port={port} />}
        </div>}
        {tab === 'server' && <HostedServerTab league={league} port={port} />}
        {tab === 'picks' && <HostedControlRoomPicks league={league} room={room} port={port} />}
        {tab === 'people' && <HostedControlRoomPeople league={league} room={room} port={port} />}
        {tab === 'feedback' && <HostedControlRoomFeedback league={league} port={port}
          refreshVersion={feedbackRefreshVersion} />}
        {tab === 'fix' && <HostedControlRoomFix league={league} room={room} port={port}
          readings={fixState.fixes} pending={fixState.pending} error={fixState.error}
          definition={openedTab === 'fix' ? definition : undefined} initial={initialValues}
          onOpen={(next, initial) => openTool(next, undefined, 'fix', initial)} />}
        {!['jobs', 'situations', 'rules', 'season', 'server', 'picks', 'people', 'feedback', 'fix', 'league']
          .includes(tab)
          && <p className="cr-note hosted-unported">{hostedControlRoomNotOnServer}</p>}
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
    void ledgerQuery.load([league, '']);
    return () => ledgerQuery.dispose();
  }, [league, ledgerQuery]);
  const ledger = ledgerState.ledger;
  const error = ledgerState.error;
  return <div className="cr-note" aria-label="Server status">
    <h3>Server</h3>
    <p>Commissioners can save and verify a fresh league backup. Restore, feed controls and upgrades
      are operator-only.</p>
    <HostedBackupNow league={league} port={port} />
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
