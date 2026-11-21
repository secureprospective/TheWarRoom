// Shared full player card used by the desktop v3 drawer and the phone Team sheet:
// header (PlayerPhoto + playerCardModel), News/Stats/Projections tabs, ContractPanel, status and note.
import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { formatMoney } from '../cards/format';
import { playerCardModel } from '../cards/playerModel';
import { PlayerPhoto, usePhoto } from '../cards/PlayerPhoto';
import { hostedWorkflow } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { NativeCut, NativeHQ } from '../commissioner/hqReading';
import { ContractPanel } from '../contracts/Contracts';
import type { ContractsProvider, PlayerContract, Reading } from '../contracts/contract';
import type { WhatIf, WhatIfProvider } from '../contracts/whatif';
import { hostedPlayerReadingUnavailable } from '../control/nativeApi';
import type { Provenance, Snapshot } from '../data/contract';
import { readPlayerStatus, type PlayerStatus } from '../phone/playerStatus';
import { readPlayerUsage } from '../research/api';
import { formatBioBorn, readPlayerNews, type PlayerNews } from '../research/espnNews';
import ResearchPlayerInspector from '../research/PlayerInspector';
import { HQAct } from './HQAct';
import { KV, UsageRows } from './HQInspector';
import { parseUsage, type Usage } from './hqPlayerData';
import {
  fixedPoints, PlayerFormBars, PlayerNoteField, playerRange, PlayerStatusRows,
  playerWeekPoints, rosterStatusLabel,
} from './playerCardParts';
import '../v3/playerCard.css';

type Tab = 'news' | 'stats' | 'proj';
const NA = 'Not available';
const fallbackProvenance: Provenance = {
  source: 'League ledger', kind: 'live',
  freshness: { state: 'live', fetchedAt: '', note: '' },
};

function safeSnapshot(snapshot: Snapshot): Snapshot {
  if (snapshot.players.provenance && snapshot.rosters.provenance) return snapshot;
  return {
    ...snapshot,
    players: { ...snapshot.players, provenance: snapshot.players.provenance ?? fallbackProvenance },
    rosters: { ...snapshot.rosters, provenance: snapshot.rosters.provenance ?? fallbackProvenance },
  };
}

function bornText(birthdate?: number, age?: number): string {
  if (birthdate === undefined) return NA;
  const date = new Date(birthdate * 1000);
  if (!Number.isFinite(date.getTime())) return NA;
  const formatted = date.toLocaleDateString('en-US', {
    month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC',
  });
  return age === undefined ? formatted : `${formatted} (${age})`;
}

function hostedContractsProvider(reading: NativeHQ, franchiseId: string): ContractsProvider {
  const { snapshot } = reading;
  const season = snapshot.league.value.season;
  const provenance = snapshot.rosters.provenance ?? fallbackProvenance;
  const base = {
    provenance, count: 0, threshold: 0, baseline: '', cutoff: '',
    method: 'League ledger', inputSHA: '', buildStamp: '',
    howCounted: 'Annual salary commitments from the league ledger.',
  };
  const empty = (): Reading<PlayerContract> => ({
    ...base, status: 'not_wired', absenceReason: NA,
  });
  return {
    contracts: () => Promise.reject(new Error(NA)),
    onChange: () => () => {},
    player: async playerId => {
      const roster = snapshot.rosters.value.find(r => r.franchiseId === franchiseId)
        ?? snapshot.rosters.value.find(r => r.players.some(p => p.id === playerId));
      const contract = roster?.players.find(p => p.id === playerId);
      const person = snapshot.players.value.find(p => p.id === playerId);
      const annual = reading.contracts?.get(playerId) ?? [];
      const schedule = annual.length ? annual
        : contract?.salary != null ? [{ season, cents: contract.salary }] : [];
      if (!contract || !schedule.length) return empty();
      const team = snapshot.franchises.value.find(f => f.id === roster?.franchiseId)?.name
        ?? roster?.franchiseId ?? '';
      const expiration = Math.max(...schedule.map(y => y.season));
      return {
        ...base, status: 'ok', absenceReason: null, count: schedule.length,
        value: {
          player: {
            id: playerId, franchiseId: roster?.franchiseId ?? franchiseId, team,
            name: person?.name ?? playerId, position: person?.position ?? '',
            salary: { cents: contract.salary ?? 0, text: formatMoney(contract.salary ?? 0) },
            expiration,
            status: contract.contractStatus === 'FLAG' ? '' : contract.contractStatus,
            info: '',
          },
          years: schedule.map(y => ({
            season: y.season, cost: { cents: y.cents, text: formatMoney(y.cents) },
            inferred: false, tag: 'Observed' as const, reason: 'Recorded league contract',
          })),
        },
      };
    },
  };
}

const emptyWhatIf: WhatIfProvider = {
  query: async (): Promise<Reading<WhatIf>> => ({
    status: 'not_wired', absenceReason: NA, count: 0, threshold: 0, baseline: '', cutoff: '',
    method: '', inputSHA: '', buildStamp: '', howCounted: '', provenance: fallbackProvenance,
  }),
  change: async () => {},
  onChange: () => () => {},
};

export type PlayerCardProps = {
  id: string; reading: NativeHQ; league: string; mine: string; port?: HostedPort;
  cut?: NativeCut; status?: PlayerStatus; fetchStatus?: boolean; statusEmpty?: string;
  noteInputId?: string;
};

export type PlayerCardSections = {
  title: string; sub: string; tabs: ReactNode; main: ReactNode; side: ReactNode;
};

export function usePlayerCard({
  id, reading, league, mine, port = hostedWorkflow, cut, status: suppliedStatus,
  fetchStatus = true, statusEmpty = NA, noteInputId = 'card-note',
}: PlayerCardProps): PlayerCardSections {
  const [tab, setTab] = useState<Tab>('news');
  const [loadedStatus, setLoadedStatus] = useState<PlayerStatus>();
  const [usage, setUsage] = useState<{ ready: boolean; value?: Usage }>({ ready: false });
  const [news, setNews] = useState<{ ready: boolean; value?: PlayerNews }>({ ready: false });
  const { snapshot } = reading;
  const season = snapshot.league.value.season;
  const ownerRoster = snapshot.rosters.value.find(r => r.players.some(p => p.id === id));
  const franchiseId = ownerRoster?.franchiseId ?? mine;
  const ownerName = snapshot.franchises.value.find(f => f.id === franchiseId)?.name;
  const player = snapshot.players.value.find(p => p.id === id);
  const rosterPlayer = ownerRoster?.players.find(p => p.id === id);
  const model = useMemo(() => playerCardModel(safeSnapshot(snapshot), franchiseId, id, new Date(), true),
    [snapshot, franchiseId, id]);
  const photo = usePhoto(id);
  const contracts = useMemo(() => hostedContractsProvider(reading, franchiseId), [reading, franchiseId]);

  useEffect(() => {
    if (!fetchStatus) return;
    let live = true;
    setLoadedStatus(undefined);
    readPlayerStatus(league, port).then(s => { if (live) setLoadedStatus(s); })
      .catch(() => { if (live) setLoadedStatus(undefined); });
    return () => { live = false; };
  }, [fetchStatus, league, port]);

  useEffect(() => {
    let live = true;
    setUsage({ ready: false });
    void readPlayerUsage(id, league, port).then(raw => {
      if (!live) return;
      const value = hostedPlayerReadingUnavailable(raw) ? undefined : parseUsage(raw);
      setUsage({ ready: true, value });
    }).catch(() => { if (live) setUsage({ ready: true }); });
    return () => { live = false; };
  }, [id, league, port]);

  useEffect(() => {
    let live = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    setNews({ ready: false });
    const load = (attempt: number) => {
      void readPlayerNews(league, id, port).then(value => {
        if (!live) return;
        setNews({ ready: true, value });
        if (value && value.espnId && !value.asOf && value.stale && attempt < 2) {
          timer = setTimeout(() => load(attempt + 1), 1200);
        }
      }).catch(() => { if (live) setNews({ ready: true }); });
    };
    load(0);
    return () => { live = false; if (timer) clearTimeout(timer); };
  }, [id, league, port]);

  const injuries = fetchStatus ? loadedStatus : suppliedStatus;
  const bio = news.value?.bio;
  // ESPN's own headshot URL is blocked by img-src 'self'; photos come only from /headshot/.
  const effectivePhoto = photo?.src ? { src: photo.src, alt: `Photo of ${model.name}` } : undefined;
  const annual = reading.contracts?.get(id) ?? [];
  const ends = annual.length ? Math.max(...annual.map(y => y.season))
    : rosterPlayer?.yearsRemaining !== undefined ? season + rosterPlayer.yearsRemaining - 1 : undefined;
  const weeks = reading.playerForm?.[id] ?? [];
  const range = playerRange(reading.playerHistory?.[id] ?? (weeks.length ? weeks : undefined));
  const weekPts = playerWeekPoints(reading, mine, id);
  const projection = reading.projections?.get(id);
  const leagueValue = reading.values?.players.find(v => v.player === id);
  const statusLine = [ownerName, rosterStatusLabel(rosterPlayer?.rosterStatus, '')]
    .filter(Boolean).join(' · ') || NA;
  const contractStatus = rosterPlayer?.contractStatus && rosterPlayer.contractStatus !== 'FLAG'
    ? rosterPlayer.contractStatus : NA;
  const sub = [model.team || bio?.team, model.position || bio?.position, ownerName]
    .filter(Boolean).join(' · ') || NA;

  const meta: [string, string][] = [
    ['Ht', bio?.height || NA],
    ['Wt', bio?.weight || NA],
    ['Born', formatBioBorn(bio?.born, bio?.age) || bornText(player?.birthdate, model.age)],
    ['College', bio?.college || NA],
    ['Draft', bio?.draft || NA],
    ['Status', statusLine],
    ['Acquired', NA],
    ['Salary', rosterPlayer?.salary != null ? model.salary : NA],
    ['Contract Year', ends ? String(ends) : NA],
    ['Contract Status', contractStatus],
    ['Years', model.years ?? NA],
    ['Contract Notes', NA],
  ];

  const tabs = <>
    <HQAct className="chip" active={tab === 'news'} onAction={() => setTab('news')}>Player News</HQAct>
    <HQAct className="chip" active={tab === 'stats'} onAction={() => setTab('stats')}>
      {season} Stats</HQAct>
    <HQAct className="chip" active={tab === 'proj'} onAction={() => setTab('proj')}>Projections</HQAct>
  </>;

  const main = <div className="pcd-main">
    <header className="pcd-head">
      <div className="pcd-photo">
        <PlayerPhoto view={effectivePhoto} team={model.team || bio?.team} />
        <div className="pcd-badges">
          {(model.position || bio?.position) && <span className="pos">
            {model.position || bio?.position}</span>}
          {model.status.map(s => <span key={s.label} className="tag">{s.label}</span>)}
        </div>
      </div>
      <dl className="pcd-meta">
        {meta.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}
      </dl>
    </header>
    {tab === 'news' && <div className="pcd-panel">
      <ResearchPlayerInspector id={id} identity={false} league={league} port={port}
        position={model.position || bio?.position} usage={usage.value} usageDone={usage.ready}
        news={news.value} newsDone={news.ready} />
    </div>}
    {tab === 'stats' && <div className="pcd-panel">
      <div className="pcd-nums">
        <div><span>THIS WEEK</span><b>{weekPts !== undefined ? fixedPoints(weekPts) : NA}</b></div>
        <div><span>PROJECTION</span><b>{projection !== undefined ? projection.toFixed(1) : NA}</b></div>
        <div><span>RANGE · 6 WK</span><b>{range
          ? range.map(x => x.toFixed(0)).join('–') : NA}</b></div>
      </div>
      <div className="pcd-lbl">Last five weeks</div>
      <PlayerFormBars weeks={weeks} empty={reading.formError || reading.formNote || NA}
        className="pcd-bars" emptyClassName="pcd-note" />
      <div className="pcd-lbl">Usage and value</div>
      <div>
        <UsageRows usage={usage.value} projection={projection} position={model.position} />
        <KV label="On-field-now PPG"
          value={leagueValue?.available ? leagueValue.nowPPG.toFixed(1) : NA} />
        <KV label="Dynasty value"
          value={leagueValue?.available ? leagueValue.dynastyPPG.toFixed(1) : NA} />
      </div>
    </div>}
    {tab === 'proj' && <div className="pcd-panel">
      <div className="pcd-nums">
        <div><span>WEEK {reading.week ?? '—'} PROJ</span>
          <b>{projection !== undefined ? projection.toFixed(1) : NA}</b></div>
        <div><span>ON-FIELD-NOW</span>
          <b>{leagueValue?.available ? leagueValue.nowPPG.toFixed(1) : NA}</b></div>
        <div><span>DYNASTY PPG</span>
          <b>{leagueValue?.available ? leagueValue.dynastyPPG.toFixed(1) : NA}</b></div>
      </div>
      <div className="pcd-lbl">Matchup and usage</div>
      <div>
        <UsageRows usage={usage.value} projection={projection} position={model.position} />
      </div>
    </div>}
  </div>;

  const side = <div className="pcd-side">
    <div className="pcd-contract">
      <ContractPanel id={id} mine="" provider={contracts} whatIf={emptyWhatIf} />
      {cut?.player === id && cut.deadNow !== undefined &&
        <KV label="Dead cap if cut today" value={formatMoney(cut.deadNow)} />}
    </div>
    <div className="pcd-lbl">Status</div>
    <PlayerStatusRows id={id} rosterStatus={rosterPlayer?.rosterStatus} injuries={injuries}
      empty={statusEmpty} rowClassName="pcd-status" />
    <PlayerNoteField league={league} id={id} port={port} inputId={noteInputId}
      labelClassName="pcd-lbl" fieldClassName="pcd-note-fld" />
  </div>;

  return { title: model.name, sub, tabs, main, side };
}
