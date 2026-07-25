import { useEffect, useMemo, useRef, useState } from 'react';
import { PreviewTransaction, ExecuteTransaction } from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { useTransactionsStore } from '../../store/transactions';
import { ConfirmModal, type Pending } from './ConfirmModal';
import { money, initials, Empty } from './format';

// TransactionWorkspace is the M4 operator UI (design doc §"Slice 1"): a subject-centric IA
// (D1) — pick a franchise from the rail → see its named roster → click a player → the action
// panel offers ONLY that player's phase-legal moves. No mflID is ever typed (D2): names come
// from the server, the id rides hidden on the request. Slice 1 wires three ops — ROSTER_STATUS
// (plain confirm), WAIVER (preview → confirm), SIGN (form → preview → confirm). Every list is
// guarded `?? []` (D9) so a Go nil-slice→JSON-null can never unmount the root (handoff-36 bug).
//
// B-2 restyle: Session-C token contract; the roster / free-agent lists migrated onto the shared
// .twr-board* grammar with the .is-selected row axis wired to the click-to-select flow. Rail +
// action panel on tokens/controls.

const FA = '__FA__'; // sentinel rail selection = the free-agent pool

// Board templates: a roster shows the cap columns, the FA pool is a lean two-column scan.
const ROSTER_COLS = '1fr 46px 70px 80px 80px';
const FA_COLS = '1fr 60px';

export function TransactionWorkspace() {
  const franchises = useTransactionsStore((s) => s.franchises);
  const roster = useTransactionsStore((s) => s.roster);
  const pool = useTransactionsStore((s) => s.pool);
  const phase = useTransactionsStore((s) => s.phase);
  // legalOps = the op kinds the engine says are phase-legal right now (GetLegalOps → phasePolicy,
  // the single source of truth). A contract button renders iff its kind is in this set (D1) — so an
  // offseason-only buyout is ABSENT (not greyed) mid-season, and never re-encodes the engine's rules.
  const legalOps = useTransactionsStore((s) => s.legalOps);
  const loadFranchises = useTransactionsStore((s) => s.loadFranchises);
  const loadPhase = useTransactionsStore((s) => s.loadPhase);
  const loadLegalOps = useTransactionsStore((s) => s.loadLegalOps);
  const loadRoster = useTransactionsStore((s) => s.loadRoster);
  const clearRoster = useTransactionsStore((s) => s.clearRoster);
  const loadPool = useTransactionsStore((s) => s.loadPool);
  const clearPool = useTransactionsStore((s) => s.clearPool);

  const [selectedFr, setSelectedFr] = useState<string | null>(null);
  const [selected, setSelected] = useState<main.M4Player | null>(null);
  const [filter, setFilter] = useState('');

  // SIGN form (only shown when a free agent is selected on a signable phase)
  const [signFranchise, setSignFranchise] = useState('');
  const [signSalary, setSignSalary] = useState('');
  const [signYears, setSignYears] = useState('1');

  // Contract-op inputs for a rostered player: EXTENSION added-years (1..3) and the §11 RESTRUCTURE
  // move ($M into signing bonus). TAG and BUYOUT need no input (price/dead-cap resolve server-side).
  const [extYears, setExtYears] = useState('1');
  const [restructureMove, setRestructureMove] = useState('');

  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  // Generation token: bumped on every stage() and on cancel, so a PreviewTransaction that
  // resolves AFTER the user dismissed (or restaged) the modal is discarded instead of
  // reopening a modal for an abandoned — possibly destructive — move (GLM L3).
  const stageGen = useRef(0);

  // SIGN legality reads from the engine's legalOps (GetLegalOps → phasePolicy), NOT a re-encoded
  // phase check — the same single-source rule the contract buttons use (D1). Keeping a second
  // hardcoded `phase === 'OFFSEASON' || …` would be exactly the drift D1 exists to prevent.
  const signable = legalOps.includes('SIGN');

  useEffect(() => {
    void Promise.all([loadFranchises(), loadPhase(), loadLegalOps()]);
  }, [loadFranchises, loadPhase, loadLegalOps]);

  async function pickFranchise(id: string) {
    setSelectedFr(id);
    setSelected(null);
    // Clear the OTHER slot FIRST (GLM 5.2 review lead B2, Session 44): clearing after the
    // await left a stale opposite-slot value on screen for one intermediate render — e.g.
    // the FA rail badge briefly showing the old pool count while the new roster was already
    // in. Same discipline as TradeBuilder.pickFranchise's pre-existing clear-first fix.
    if (id === FA) {
      clearRoster();
      await loadPool();
    } else {
      clearPool();
      await loadRoster(id);
    }
  }

  async function refreshCurrent() {
    if (selectedFr === FA) await loadPool();
    else if (selectedFr) await loadRoster(selectedFr);
    setSelected(null);
  }

  function cancel() {
    stageGen.current++; // invalidate any in-flight preview so it can't reopen the modal (L3)
    setPending(null);
  }

  // Build the request for the selected op, run a preview if the op is priced, and open the modal.
  async function stage(kind: Pending['kind'], player: main.M4Player) {
    const req = buildReq(kind, player);
    if (!req) return;
    const gen = ++stageGen.current;

    if (kind === 'ROSTER_STATUS') {
      // Costless — no preview (design D7). Straight to a plain confirm.
      const toTaxi = player.rosterStatus !== 'TAXI_SQUAD';
      setPending({
        kind,
        title: 'Roster status',
        subject: `${toTaxi ? 'Move to taxi' : 'Activate'} — ${player.name}`,
        meta: `${player.position} · ${selectedFr}`,
        note: `No cap change. ${player.name} moves to the ${toTaxi ? 'taxi squad' : 'active roster'}.`,
        destructive: false,
        previewing: false,
        previewOK: null,
        detail: '',
        playersAffected: 0,
        capDeltas: [],
        request: req,
      });
      return;
    }

    // Priced op — open the modal in a previewing state, then fill it from PreviewTransaction (D5).
    const copy = opCopy(kind, player);
    const base: Pending = {
      kind,
      title: copy.title,
      subject: copy.subject,
      meta: copy.meta,
      note: copy.note,
      destructive: copy.destructive,
      previewing: true,
      previewOK: null,
      detail: '',
      playersAffected: 0,
      capDeltas: [],
      request: req,
    };
    setPending(base);
    try {
      const res = await PreviewTransaction(req);
      if (gen !== stageGen.current) return; // cancelled or restaged while the preview was in flight (L3)
      setPending({
        ...base,
        previewing: false,
        previewOK: res.ok,
        detail: res.detail,
        playersAffected: res.playersAffected,
        capDeltas: res.capDeltas ?? [],
      });
    } catch (e) {
      // A thrown IPC call (bridge down, panic) must not leave the modal stuck "previewing…"
      // forever — fall into the rejected terminal state so the operator can read it and close.
      if (gen !== stageGen.current) return;
      setPending({
        ...base,
        previewing: false,
        previewOK: false,
        detail: `Couldn't preview this move — the engine was unreachable (${e instanceof Error ? e.message : String(e)}).`,
      });
    }
  }

  // opCopy is the modal's human framing for a priced op — no money math (the engine owns the figures);
  // every dollar shown is either what the operator typed (SIGN/RESTRUCTURE) or lands on the roster
  // after the commit (WAIVER dead cap, TAG price, EXTENSION future cap, BUYOUT dead cap).
  function opCopy(kind: Pending['kind'], player: main.M4Player) {
    const rosterMeta = `${player.position} · ${selectedFr}`;
    switch (kind) {
      case 'WAIVER':
        return {
          title: 'Waiver · cut',
          subject: `Cut ${player.name}?`,
          meta: rosterMeta,
          note: 'Cutting releases the player and charges §8 dead cap. The dead-cap figure lands on the roster after you confirm.',
          destructive: true,
        };
      case 'TAG':
        return {
          title: 'Franchise tag · §9',
          subject: `Tag ${player.name}?`,
          meta: rosterMeta,
          note: 'The engine resolves the §9 tag price — the top-5-by-position average, floored at 120% of prior salary. The new cap figure lands on the roster after you confirm.',
          destructive: false,
        };
      case 'EXTENSION':
        return {
          title: 'Extension · §10',
          subject: `Extend ${player.name} +${extYears}yr?`,
          meta: rosterMeta,
          note: `Adds ${extYears} year(s) at the §10 price (150% of the highest remaining year, position-floored). Only FUTURE cap is added — the current season is unchanged.`,
          destructive: false,
        };
      case 'RESTRUCTURE':
        return {
          title: 'Restructure · §11',
          subject: `Restructure ${player.name}?`,
          meta: rosterMeta,
          note: `Moves ${money(Number(restructureMove) || 0)} into a signing bonus (§11). The engine resolves the exact cap effect and the new figure lands on the roster.`,
          destructive: false,
        };
      case 'BUYOUT':
        return {
          title: 'Buyout · §12',
          subject: `Buy out ${player.name}?`,
          meta: rosterMeta,
          note: 'Buys out the contract (offseason only, two per team per season). Dead cap is charged; the figure lands on the roster after you confirm.',
          destructive: true,
        };
      default: // SIGN
        return {
          title: 'Free agency · sign',
          subject: `Sign ${player.name}?`,
          meta: `${player.position} · Free agent`,
          note: `Signs ${player.name} to franchise ${signFranchise} at ${money(Number(signSalary) || 0)}/yr for ${signYears} year(s).`,
          destructive: false,
        };
    }
  }

  function buildReq(kind: Pending['kind'], player: main.M4Player): main.TransactionRequest | null {
    if (kind === 'ROSTER_STATUS') {
      return main.TransactionRequest.createFrom({
        kind,
        mflID: player.mflID,
        status: player.rosterStatus === 'TAXI_SQUAD' ? 'ROSTER' : 'TAXI_SQUAD',
      });
    }
    // WAIVER / TAG / BUYOUT carry only the player id — every figure resolves server-side.
    if (kind === 'WAIVER' || kind === 'TAG' || kind === 'BUYOUT') {
      return main.TransactionRequest.createFrom({ kind, mflID: player.mflID });
    }
    if (kind === 'EXTENSION') {
      return main.TransactionRequest.createFrom({
        kind,
        mflID: player.mflID,
        addedYears: Number(extYears) || 0,
      });
    }
    if (kind === 'RESTRUCTURE') {
      if (!restructureMove.trim()) return null;
      return main.TransactionRequest.createFrom({
        kind,
        mflID: player.mflID,
        moveMillions: restructureMove.trim(), // parsed to exact cents server-side, never a JS number
      });
    }
    // SIGN — validate the form
    if (!signFranchise.trim() || !signSalary.trim()) return null;
    return main.TransactionRequest.createFrom({
      kind,
      mflID: player.mflID,
      franchiseID: signFranchise.trim(),
      salaryMillions: signSalary.trim(),
      years: Number(signYears) || 0,
    });
  }

  async function confirm() {
    if (!pending) return;
    setBusy(true);
    try {
      // ExecuteTransaction encodes rejection as ok:false + detail (it does not throw). The engine
      // can still reject at commit even after an OK preview — a phase flip, a closed signing window,
      // a cap change between preview and commit. Surface that instead of showing a false success (L1).
      const res = await ExecuteTransaction(pending.request);
      if (!res.ok) {
        setPending({ ...pending, previewing: false, previewOK: false, detail: res.detail });
        return;
      }
      setPending(null);
      await refreshCurrent();
    } finally {
      setBusy(false);
    }
  }

  const shownFranchises = useMemo(() => {
    const f = filter.trim().toLowerCase();
    if (!f) return franchises;
    return franchises.filter((x) => x.franchiseID.toLowerCase().includes(f));
  }, [franchises, filter]);

  return (
    <div
      style={{
        display: 'flex',
        height: 'calc(100vh - 190px)',
        minHeight: 520,
        border: '1px solid var(--hairline)',
        background: 'var(--surface-canvas)',
        color: 'var(--text-primary)',
      }}
    >
      {/* Left rail — franchises + free agents */}
      <nav
        style={{
          display: 'flex',
          width: 208,
          flex: 'none',
          flexDirection: 'column',
          borderRight: '1px solid var(--hairline)',
          background: 'var(--surface-sunken)',
        }}
      >
        <div style={{ borderBottom: '1px solid var(--hairline)', padding: 10 }}>
          <input
            className="twr-input"
            style={{ width: '100%' }}
            placeholder="Filter franchises…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '6px 0' }}>
          {shownFranchises.map((f) => (
            <RailRow
              key={f.franchiseID}
              label={f.name || f.franchiseID}
              abbr={f.franchiseID}
              count={f.playerCount}
              active={selectedFr === f.franchiseID}
              onClick={() => void pickFranchise(f.franchiseID)}
            />
          ))}
          <div style={{ margin: '6px 12px', height: 1, background: 'var(--hairline)' }} />
          <RailRow
            label="Free Agents"
            abbr="FA"
            count={pool?.players?.length}
            active={selectedFr === FA}
            gold
            onClick={() => void pickFranchise(FA)}
          />
        </div>
      </nav>

      {/* Center — roster or free-agent board */}
      <section style={{ display: 'flex', minWidth: 0, flex: 1, flexDirection: 'column' }}>
        {selectedFr === FA ? (
          <FreeAgentTable pool={pool} selected={selected} onSelect={setSelected} />
        ) : (
          <RosterTable roster={roster} selectedFr={selectedFr} selected={selected} onSelect={setSelected} />
        )}
      </section>

      {/* Right — contextual action panel */}
      <aside
        style={{
          display: 'flex',
          width: 336,
          flex: 'none',
          flexDirection: 'column',
          borderLeft: '1px solid var(--hairline)',
          background: 'var(--surface-tile)',
        }}
      >
        {!selected ? (
          <div style={{ margin: 'auto', maxWidth: 220, padding: 32, textAlign: 'center', fontSize: 13, color: 'var(--text-tertiary)' }}>
            Select a {selectedFr === FA ? 'free agent' : 'player'} to see the moves available this
            phase.
          </div>
        ) : (
          <ActionPanel
            player={selected}
            isFreeAgent={selectedFr === FA}
            signable={signable}
            legalOps={legalOps}
            signFranchise={signFranchise}
            signSalary={signSalary}
            signYears={signYears}
            setSignFranchise={setSignFranchise}
            setSignSalary={setSignSalary}
            setSignYears={setSignYears}
            extYears={extYears}
            setExtYears={setExtYears}
            restructureMove={restructureMove}
            setRestructureMove={setRestructureMove}
            onStage={stage}
          />
        )}
      </aside>

      <ConfirmModal pending={pending} busy={busy} onConfirm={() => void confirm()} onCancel={cancel} />
    </div>
  );
}

function RailRow({
  label,
  abbr,
  count,
  active,
  gold,
  onClick,
}: {
  label: string;
  abbr: string;
  count?: number;
  active: boolean;
  gold?: boolean;
  onClick: () => void;
}) {
  const nameColor = gold ? 'var(--amber-base)' : 'var(--text-primary)';
  const abbrColor = gold ? 'var(--amber-base)' : active ? 'var(--text-primary)' : 'var(--text-secondary)';
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        display: 'flex',
        width: '100%',
        alignItems: 'center',
        gap: 10,
        borderLeft: `2px solid ${active ? 'var(--edge-selection)' : 'transparent'}`,
        background: active ? 'var(--surface-overlay)' : 'transparent',
        padding: '7px 12px',
        textAlign: 'left',
        cursor: 'pointer',
      }}
      onMouseEnter={(e) => {
        if (!active) e.currentTarget.style.background = 'var(--surface-raised)';
      }}
      onMouseLeave={(e) => {
        if (!active) e.currentTarget.style.background = 'transparent';
      }}
    >
      <span style={{ width: 32, fontSize: 11, fontWeight: 700, fontFamily: 'var(--mono)', color: abbrColor }}>
        {abbr}
      </span>
      <span
        style={{
          flex: 1,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          fontSize: 12.5,
          color: nameColor,
        }}
      >
        {label}
      </span>
      {count !== undefined && (
        <span
          style={{
            marginLeft: 'auto',
            fontSize: 10.5,
            fontFamily: 'var(--mono)',
            fontVariantNumeric: 'tabular-nums',
            color: 'var(--text-tertiary)',
          }}
        >
          {count}
        </span>
      )}
    </button>
  );
}

function RosterTable({
  roster,
  selectedFr,
  selected,
  onSelect,
}: {
  roster: main.RosterResult | null;
  selectedFr: string | null;
  selected: main.M4Player | null;
  onSelect: (p: main.M4Player) => void;
}) {
  if (!selectedFr) return <Empty text="Pick a franchise from the rail to load its roster." />;
  if (!roster) return <Empty text="Loading…" />;
  if (!roster.ok) return <Empty text={roster.detail || 'Could not load this roster.'} />;
  const players = roster.players ?? [];

  return (
    <>
      <div style={{ borderBottom: '1px solid var(--hairline)', padding: '16px 22px 14px' }}>
        <h1 style={{ margin: 0, fontSize: 21, fontWeight: 700, letterSpacing: '-0.01em' }}>{roster.franchiseID}</h1>
        <div style={{ marginTop: 8, fontSize: 13, color: 'var(--text-secondary)' }}>
          Cap used{' '}
          <b style={{ fontFamily: 'var(--mono)', fontVariantNumeric: 'tabular-nums', color: 'var(--amber-base)' }}>
            {money(roster.capUsed)}
          </b>
        </div>
        {roster.warning && <div style={{ marginTop: 4, fontSize: 11, color: 'var(--amber-base)' }}>{roster.warning}</div>}
      </div>
      <PlayerTable players={players} selected={selected} onSelect={onSelect} showCap />
    </>
  );
}

function FreeAgentTable({
  pool,
  selected,
  onSelect,
}: {
  pool: main.FreeAgentPoolResult | null;
  selected: main.M4Player | null;
  onSelect: (p: main.M4Player) => void;
}) {
  if (!pool) return <Empty text="Loading…" />;
  if (!pool.ok) return <Empty text={pool.detail || 'Could not load the free-agent pool.'} />;
  const players = pool.players ?? [];

  return (
    <>
      <div style={{ borderBottom: '1px solid var(--hairline)', padding: '16px 22px 14px' }}>
        <h1 style={{ margin: 0, fontSize: 21, fontWeight: 700, letterSpacing: '-0.01em', color: 'var(--amber-base)' }}>
          Free Agents
        </h1>
        <div style={{ marginTop: 8, fontSize: 13, color: 'var(--text-secondary)' }}>
          <span style={{ fontFamily: 'var(--mono)', fontVariantNumeric: 'tabular-nums' }}>{players.length}</span>{' '}
          signable player(s)
        </div>
        {pool.warning && <div style={{ marginTop: 4, fontSize: 11, color: 'var(--amber-base)' }}>{pool.warning}</div>}
      </div>
      {players.length === 0 ? (
        <Empty text="The pool is empty — sign or roll a season to populate it." />
      ) : (
        <PlayerTable players={players} selected={selected} onSelect={onSelect} showCap={false} />
      )}
    </>
  );
}

function PlayerTable({
  players,
  selected,
  onSelect,
  showCap,
}: {
  players: main.M4Player[];
  selected: main.M4Player | null;
  onSelect: (p: main.M4Player) => void;
  showCap: boolean;
}) {
  return (
    <div style={{ flex: 1, overflow: 'auto' }}>
      <div className="twr-board" style={{ ['--twr-cols' as string]: showCap ? ROSTER_COLS : FA_COLS }}>
        <div className="twr-board__sub">
          <span>Player</span>
          <span>Pos</span>
          {showCap && <span>Status</span>}
          {showCap && <span className="twr-r">Salary</span>}
          {showCap && <span className="twr-r">Cap</span>}
        </div>
        {players.map((p) => (
          <div
            key={p.mflID}
            onClick={() => onSelect(p)}
            className={`twr-board__row${selected?.mflID === p.mflID ? ' is-selected' : ''}`}
            style={{ cursor: 'pointer' }}
          >
            <span className="twr-c-name">{p.name}</span>
            <span className="twr-c-pos">{p.position}</span>
            {showCap && (
              <span>
                <span
                  style={{
                    fontFamily: 'var(--mono)',
                    fontSize: 10.5,
                    fontWeight: 600,
                    padding: '2px 6px',
                    border: '1px solid var(--hairline)',
                    color: p.rosterStatus === 'TAXI_SQUAD' ? 'var(--amber-base)' : 'var(--text-tertiary)',
                  }}
                >
                  {p.rosterStatus === 'TAXI_SQUAD' ? 'TAXI' : 'ROSTER'}
                </span>
              </span>
            )}
            {showCap && <span className="twr-c-num twr-r">{money(p.salary)}</span>}
            {showCap && (
              <span className="twr-c-num twr-r" style={{ color: 'var(--amber-base)' }}>
                {money(p.capSalary)}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function ActionPanel({
  player,
  isFreeAgent,
  signable,
  legalOps,
  signFranchise,
  signSalary,
  signYears,
  setSignFranchise,
  setSignSalary,
  setSignYears,
  extYears,
  setExtYears,
  restructureMove,
  setRestructureMove,
  onStage,
}: {
  player: main.M4Player;
  isFreeAgent: boolean;
  signable: boolean;
  legalOps: string[];
  signFranchise: string;
  signSalary: string;
  signYears: string;
  setSignFranchise: (v: string) => void;
  setSignSalary: (v: string) => void;
  setSignYears: (v: string) => void;
  extYears: string;
  setExtYears: (v: string) => void;
  restructureMove: string;
  setRestructureMove: (v: string) => void;
  onStage: (kind: Pending['kind'], p: main.M4Player) => void;
}) {
  const toTaxi = player.rosterStatus !== 'TAXI_SQUAD';
  const can = (kind: string) => legalOps.includes(kind); // phase-legal per the engine (D1)
  return (
    <>
      <div style={{ borderBottom: '1px solid var(--hairline)', padding: '18px 18px 14px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 11 }}>
          <div
            style={{
              display: 'grid',
              height: 40,
              width: 40,
              placeItems: 'center',
              background: 'var(--surface-overlay)',
              fontSize: 15,
              fontWeight: 700,
              fontFamily: 'var(--mono)',
              color: 'var(--text-secondary)',
            }}
          >
            {initials(player.name)}
          </div>
          <div>
            <div style={{ fontSize: 16, fontWeight: 700 }}>{player.name}</div>
            <div style={{ marginTop: 1, fontSize: 11.5, color: 'var(--text-secondary)' }}>
              {player.position} ·{' '}
              {isFreeAgent ? 'Free agent' : player.rosterStatus === 'TAXI_SQUAD' ? 'Taxi squad' : 'Active roster'}
            </div>
          </div>
        </div>
      </div>

      <div style={{ overflowY: 'auto', padding: '16px 18px' }}>
        {isFreeAgent ? (
          <>
            <SectionLabel>Sign to a franchise</SectionLabel>
            {!signable ? (
              <p style={{ margin: 0, fontSize: 12, color: 'var(--text-tertiary)' }}>
                Signing is closed this phase (free agency runs in the offseason and regular season).
              </p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                <input
                  className="twr-input"
                  style={{ width: '100%' }}
                  placeholder="→ franchise id"
                  value={signFranchise}
                  onChange={(e) => setSignFranchise(e.target.value)}
                />
                <div style={{ display: 'flex', gap: 8 }}>
                  <input
                    className="twr-input"
                    style={{ minWidth: 0, flex: 1 }}
                    placeholder="salary/yr ($M)"
                    value={signSalary}
                    onChange={(e) => setSignSalary(e.target.value)}
                  />
                  <select className="twr-select" value={signYears} onChange={(e) => setSignYears(e.target.value)}>
                    <option value="1">1 yr</option>
                    <option value="2">2 yr</option>
                    <option value="3">3 yr</option>
                    <option value="4">4 yr</option>
                  </select>
                </div>
                <button
                  type="button"
                  className="twr-btn"
                  style={{ width: '100%', padding: '10px 12px' }}
                  disabled={!signFranchise.trim() || !signSalary.trim()}
                  onClick={() => onStage('SIGN', player)}
                >
                  Review signing…
                </button>
              </div>
            )}
          </>
        ) : (
          <>
            <SectionLabel>Roster moves</SectionLabel>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              {can('ROSTER_STATUS') && (
                <Act onClick={() => onStage('ROSTER_STATUS', player)}>{toTaxi ? 'Move to Taxi' : 'Activate'}</Act>
              )}
              {can('WAIVER') && (
                <Act danger onClick={() => onStage('WAIVER', player)}>
                  Cut
                </Act>
              )}
            </div>

            {/* Contract moves — each shown only where the engine says it's phase-legal (D1). */}
            <div style={{ marginTop: 20 }}>
              <SectionLabel>Contract moves</SectionLabel>
            </div>
            {can('TAG') || can('EXTENSION') || can('RESTRUCTURE') || can('BUYOUT') ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                {can('TAG') && <Act onClick={() => onStage('TAG', player)}>Franchise tag (§9)</Act>}

                {can('EXTENSION') && (
                  <div style={{ display: 'flex', gap: 8 }}>
                    <select
                      className="twr-select"
                      value={extYears}
                      onChange={(e) => setExtYears(e.target.value)}
                      aria-label="Years to add"
                    >
                      <option value="1">+1 yr</option>
                      <option value="2">+2 yr</option>
                      <option value="3">+3 yr</option>
                    </select>
                    <div style={{ minWidth: 0, flex: 1 }}>
                      <Act onClick={() => onStage('EXTENSION', player)}>Extend (§10)</Act>
                    </div>
                  </div>
                )}

                {can('RESTRUCTURE') && (
                  <div style={{ display: 'flex', gap: 8 }}>
                    <input
                      className="twr-input"
                      style={{ minWidth: 0, flex: 1 }}
                      placeholder="move ($M)"
                      inputMode="decimal"
                      value={restructureMove}
                      onChange={(e) => setRestructureMove(e.target.value)}
                    />
                    <button
                      type="button"
                      className="twr-btn"
                      style={{ flex: 'none' }}
                      disabled={!restructureMove.trim()}
                      onClick={() => onStage('RESTRUCTURE', player)}
                    >
                      Restructure (§11)
                    </button>
                  </div>
                )}

                {can('BUYOUT') && (
                  <Act danger onClick={() => onStage('BUYOUT', player)}>
                    Buy out contract (§12)
                  </Act>
                )}
              </div>
            ) : (
              <p style={{ margin: 0, fontSize: 11, color: 'var(--text-tertiary)' }}>
                No contract moves are legal in this phase.
              </p>
            )}

            <p style={{ margin: '16px 0 0', fontSize: 11, color: 'var(--text-tertiary)' }}>
              Multi-player trades are built on the Trade tab; commissioner calendar and powers live on
              the League Controls tab.
            </p>
          </>
        )}
      </div>
    </>
  );
}

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        marginBottom: 10,
        fontSize: 10.5,
        fontWeight: 600,
        textTransform: 'uppercase',
        letterSpacing: '0.09em',
        color: 'var(--text-tertiary)',
      }}
    >
      {children}
    </div>
  );
}

function Act({
  children,
  danger,
  onClick,
}: {
  children: React.ReactNode;
  danger?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={`twr-btn${danger ? ' twr-btn--danger' : ''}`}
      style={{ width: '100%', padding: '10px 12px', textTransform: 'none', letterSpacing: 'normal', fontSize: 13 }}
      onClick={onClick}
    >
      {children}
    </button>
  );
}
