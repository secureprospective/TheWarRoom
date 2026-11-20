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

/** The Control Room's reads, tab state and actions; the old room and the desktop v3 bodies both use it. */
export function useHostedControlRoom(league: string, port?: HostedPort, heldRoom?: HostedLeagueRoom) {
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
  const refreshAll = () => {
    void room.refresh(); jobsQuery.invalidate(); void jobsQuery.load([league]);
    void loadJobSnoozes(); if (tab === 'fix') refreshFixes();
    if (tab === 'feedback') setFeedbackRefreshVersion(version => version + 1);
  };
  return {
    room, roomState, jobsState, fixState, tab, definition, openedTab, initialValues, job, routeWords,
    feedbackRefreshVersion, jobSnoozes, snoozeError, snoozePending, jobNow, snoozeJob, wakeJob, openTool,
    openSeasonTool, showRoute, navigateTo, jobCount, refreshAll,
  };
}

export type HostedControlRoomState = ReturnType<typeof useHostedControlRoom>;

export function HostedControlRoom({ league, port, heldRoom }: {
  league: string; port?: HostedPort; heldRoom?: HostedLeagueRoom;
}) {
  const c = useHostedControlRoom(league, port, heldRoom);

  return <section className="crx hosted-control-room" data-testid="hosted-control-room"
    aria-label="Hosted Control Room">
    <main className="work">
      <div className="ws-tabs" role="tablist" aria-label="Control Room">
        {hostedControlTabs.map(([id, label]) => <HQAct key={id} role="tab" selected={c.tab === id}
          label={label} onAction={() => c.navigateTo(id)}>
          {label}{id === 'jobs' && c.jobCount !== undefined
            ? <> <span className="cr-st">{c.jobCount}</span></> : null}
        </HQAct>)}
        <HQAct className="hosted-control-refresh" disabled={c.roomState.pending || c.jobsState.pending
          || c.snoozePending !== ''}
          onAction={c.refreshAll}>
          Refresh
        </HQAct>
      </div>
      <section className="cr-main on hosted-control-content"
        aria-label={hostedControlTabs.find(([id]) => id === c.tab)?.[1]}>
        {c.routeWords && <p role="status" className="cr-note">{c.routeWords}</p>}
        {c.tab === 'jobs' && <div className="hosted-jobs-column">
          {c.jobsState.error && <p role="alert" className="cr-note">{c.jobsState.error}</p>}
          <HostedControlRoomJobs league={league} jobs={c.jobsState.jobs} snoozes={c.jobSnoozes} now={c.jobNow}
            snoozeError={c.snoozeError} pendingJob={c.snoozePending} onSnooze={c.snoozeJob} onWake={c.wakeJob}
            onOpen={c.openTool} onUnavailable={c.showRoute} />
          {c.definition && c.openedTab === 'jobs' && <HostedCommissionerActTray definition={c.definition}
            league={league} room={c.room} port={port} initial={c.initialValues} />}
          {c.job && <p className="hosted-job-source">Opened from: {c.job.name}</p>}
        </div>}
        {c.tab === 'situations' && <HostedControlRoomSituations
          onTab={c.navigateTo} onTool={next => c.openTool(next)} onUnavailable={c.showRoute} />}
        {c.tab === 'rules' && c.roomState.error && <p role="alert">{c.roomState.error}</p>}
        {c.tab === 'rules' && c.roomState.stale &&
          <p role="status">Rules are read-only until refreshed from the league server.</p>}
        {c.tab === 'rules' && c.roomState.message && <p role="status">{c.roomState.message}</p>}
        {c.tab === 'rules' && c.roomState.reading
          ? <HostedRules rules={c.roomState.reading.rules} commissioner room={c.room} />
          : c.tab === 'rules' && <p role="status">{c.roomState.error || 'Reading league rules…'}</p>}
        {c.tab === 'league' && <HostedControlRoomLeague key={league} league={league}
          room={c.room} port={port} />}
        {c.tab === 'season' && <div className="hosted-jobs-column">
          <HostedControlRoomSeason league={league} port={port} onTool={c.openSeasonTool} />
          {c.definition && <HostedCommissionerActTray definition={c.definition} league={league}
            room={c.room} port={port} />}
        </div>}
        {c.tab === 'server' && <HostedServerTab league={league} port={port} />}
        {c.tab === 'picks' && <HostedControlRoomPicks league={league} room={c.room} port={port} />}
        {c.tab === 'people' && <HostedControlRoomPeople league={league} room={c.room} port={port} />}
        {c.tab === 'feedback' && <HostedControlRoomFeedback league={league} port={port}
          refreshVersion={c.feedbackRefreshVersion} />}
        {c.tab === 'fix' && <HostedControlRoomFix league={league} room={c.room} port={port}
          readings={c.fixState.fixes} pending={c.fixState.pending} error={c.fixState.error}
          definition={c.openedTab === 'fix' ? c.definition : undefined} initial={c.initialValues}
          onOpen={(next, initial) => c.openTool(next, undefined, 'fix', initial)} />}
        {!['jobs', 'situations', 'rules', 'season', 'server', 'picks', 'people', 'feedback', 'fix', 'league']
          .includes(c.tab)
          && <p className="cr-note hosted-unported">{hostedControlRoomNotOnServer}</p>}
      </section>
    </main>
    <aside className="insp" aria-label="League health">
      <div className="cr-ins">
        <h3>League health</h3>
        <div className="lbl">CURRENT JOB COUNT</div>
        <div className="big">{c.jobCount === undefined ? '—' : c.jobCount}</div>
        <p className="sub">One count from the league jobs reading. No separate health
          producer is available here.</p>
        <div className="lbl">SERVER RULES</div>
        <p className="sub">{c.roomState.reading?.rules.name || c.roomState.error
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
