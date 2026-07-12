import { useEffect, useMemo, useRef, useState } from 'react';
import {
  GetFranchises,
  GetRoster,
  GetFreeAgentPool,
  GetCurrentPhase,
  PreviewTransaction,
  ExecuteTransaction,
} from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { ConfirmModal, type Pending } from './ConfirmModal';

// TransactionWorkspace is the M4 operator UI (design doc §"Slice 1"): a subject-centric IA
// (D1) — pick a franchise from the rail → see its named roster → click a player → the action
// panel offers ONLY that player's phase-legal moves. No mflID is ever typed (D2): names come
// from the server, the id rides hidden on the request. Slice 1 wires three ops — ROSTER_STATUS
// (plain confirm), WAIVER (preview → confirm), SIGN (form → preview → confirm). Every list is
// guarded `?? []` (D9) so a Go nil-slice→JSON-null can never unmount the root (handoff-36 bug).

const FA = '__FA__'; // sentinel rail selection = the free-agent pool
const money = (m: number) => `$${m.toFixed(1)}M`;
const initials = (name: string) =>
  name
    .split(' ')
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase();

export function TransactionWorkspace() {
  const [franchises, setFranchises] = useState<main.M4Franchise[]>([]);
  const [selectedFr, setSelectedFr] = useState<string | null>(null);
  const [roster, setRoster] = useState<main.RosterResult | null>(null);
  const [pool, setPool] = useState<main.FreeAgentPoolResult | null>(null);
  const [selected, setSelected] = useState<main.M4Player | null>(null);
  const [phase, setPhase] = useState('…');
  const [filter, setFilter] = useState('');

  // SIGN form (only shown when a free agent is selected on a signable phase)
  const [signFranchise, setSignFranchise] = useState('');
  const [signSalary, setSignSalary] = useState('');
  const [signYears, setSignYears] = useState('1');

  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  // Generation token: bumped on every stage() and on cancel, so a PreviewTransaction that
  // resolves AFTER the user dismissed (or restaged) the modal is discarded instead of
  // reopening a modal for an abandoned — possibly destructive — move (GLM L3).
  const stageGen = useRef(0);

  const signable = phase === 'OFFSEASON' || phase === 'REGULAR_SEASON';

  useEffect(() => {
    void (async () => {
      const [fr, ph] = await Promise.all([GetFranchises(), GetCurrentPhase()]);
      setFranchises(fr.ok ? (fr.franchises ?? []) : []);
      setPhase(ph.ok ? ph.phase : `? (${ph.detail})`);
    })();
  }, []);

  async function pickFranchise(id: string) {
    setSelectedFr(id);
    setSelected(null);
    if (id === FA) {
      setPool(await GetFreeAgentPool());
      setRoster(null);
    } else {
      setRoster(await GetRoster(id));
      setPool(null);
    }
  }

  async function refreshCurrent() {
    if (selectedFr === FA) setPool(await GetFreeAgentPool());
    else if (selectedFr) setRoster(await GetRoster(selectedFr));
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
        request: req,
      });
      return;
    }

    // Priced op — open the modal in a previewing state, then fill it from PreviewTransaction (D5).
    const base: Pending = {
      kind,
      title: kind === 'WAIVER' ? 'Waiver · cut' : 'Free agency · sign',
      subject: kind === 'WAIVER' ? `Cut ${player.name}?` : `Sign ${player.name}?`,
      meta: `${player.position} · ${kind === 'WAIVER' ? selectedFr : 'Free agent'}`,
      note:
        kind === 'WAIVER'
          ? 'Cutting releases the player and charges §8 dead cap. The dead-cap figure lands on the roster after you confirm.'
          : `Signs ${player.name} to franchise ${signFranchise} at ${money(Number(signSalary) || 0)}/yr for ${signYears} year(s).`,
      destructive: kind === 'WAIVER',
      previewing: true,
      previewOK: null,
      detail: '',
      playersAffected: 0,
      request: req,
    };
    setPending(base);
    const res = await PreviewTransaction(req);
    if (gen !== stageGen.current) return; // cancelled or restaged while the preview was in flight (L3)
    setPending({
      ...base,
      previewing: false,
      previewOK: res.ok,
      detail: res.detail,
      playersAffected: res.playersAffected,
    });
  }

  function buildReq(kind: Pending['kind'], player: main.M4Player): main.TransactionRequest | null {
    if (kind === 'ROSTER_STATUS') {
      return main.TransactionRequest.createFrom({
        kind,
        mflID: player.mflID,
        status: player.rosterStatus === 'TAXI_SQUAD' ? 'ROSTER' : 'TAXI_SQUAD',
      });
    }
    if (kind === 'WAIVER') {
      return main.TransactionRequest.createFrom({ kind, mflID: player.mflID });
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
    <div className="flex h-[calc(100vh-190px)] min-h-[520px] border border-[#29344a] bg-[#0e1420] text-[#e2e8f0]">
      {/* Left rail — franchises + free agents */}
      <nav className="flex w-52 flex-none flex-col border-r border-[#29344a] bg-[#0c121d]">
        <div className="border-b border-[#202a3d] p-2.5">
          <input
            className="h-[30px] w-full border border-[#29344a] bg-[#161d2b] px-2.5 text-[12.5px] outline-none focus:border-[#5b9dff]"
            placeholder="Filter franchises…"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
        <div className="flex-1 overflow-y-auto py-1.5">
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
          <div className="mx-3 my-1.5 h-px bg-[#202a3d]" />
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

      {/* Center — roster or free-agent table */}
      <section className="flex min-w-0 flex-1 flex-col">
        {selectedFr === FA ? (
          <FreeAgentTable pool={pool} selected={selected} onSelect={setSelected} />
        ) : (
          <RosterTable roster={roster} selectedFr={selectedFr} selected={selected} onSelect={setSelected} />
        )}
      </section>

      {/* Right — contextual action panel */}
      <aside className="flex w-[336px] flex-none flex-col border-l border-[#29344a] bg-[#161d2b]">
        {!selected ? (
          <div className="m-auto max-w-[220px] p-8 text-center text-[13px] text-[#64748b]">
            Select a {selectedFr === FA ? 'free agent' : 'player'} to see the moves available this
            phase.
          </div>
        ) : (
          <ActionPanel
            player={selected}
            isFreeAgent={selectedFr === FA}
            signable={signable}
            signFranchise={signFranchise}
            signSalary={signSalary}
            signYears={signYears}
            setSignFranchise={setSignFranchise}
            setSignSalary={setSignSalary}
            setSignYears={setSignYears}
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
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-center gap-2.5 border-l-2 px-3 py-[7px] text-left transition-colors ${
        active
          ? 'border-[#5b9dff] bg-[rgba(91,157,255,0.12)]'
          : 'border-transparent hover:bg-[#161d2b]'
      }`}
    >
      <span
        className={`w-8 text-[11px] font-bold ${
          gold ? 'text-[#f0b429]' : active ? 'text-[#5b9dff]' : 'text-[#93a1b8]'
        }`}
      >
        {abbr}
      </span>
      <span className={`truncate text-[12.5px] ${gold ? 'text-[#f0b429]' : 'text-[#e2e8f0]'}`}>{label}</span>
      {count !== undefined && (
        <span className="ml-auto text-[10.5px] tabular-nums text-[#64748b]">{count}</span>
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
      <div className="border-b border-[#202a3d] px-[22px] pb-3.5 pt-4">
        <h1 className="text-[21px] font-bold tracking-[-0.01em]">{roster.franchiseID}</h1>
        <div className="mt-2 text-[13px]">
          Cap used <b className="tabular-nums text-[#f0b429]">{money(roster.capUsed)}</b>
        </div>
        {roster.warning && <div className="mt-1 text-[11px] text-[#f0b429]">{roster.warning}</div>}
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
      <div className="border-b border-[#202a3d] px-[22px] pb-3.5 pt-4">
        <h1 className="text-[21px] font-bold tracking-[-0.01em] text-[#f0b429]">Free Agents</h1>
        <div className="mt-2 text-[13px] text-[#93a1b8]">
          <span className="tabular-nums">{players.length}</span> signable player(s)
        </div>
        {pool.warning && <div className="mt-1 text-[11px] text-[#f0b429]">{pool.warning}</div>}
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
    <div className="flex-1 overflow-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr className="text-[#64748b]">
            <Th>Player</Th>
            <Th>Pos</Th>
            {showCap && <Th>Status</Th>}
            {showCap && <Th right>Salary</Th>}
            {showCap && <Th right>Cap</Th>}
          </tr>
        </thead>
        <tbody>
          {players.map((p) => (
            <tr
              key={p.mflID}
              onClick={() => onSelect(p)}
              className={`cursor-pointer border-b border-[#202a3d] transition-colors ${
                selected?.mflID === p.mflID
                  ? 'bg-[rgba(91,157,255,0.12)] shadow-[inset_2px_0_0_#5b9dff]'
                  : 'hover:bg-[#161d2b]'
              }`}
            >
              <td className="px-3.5 py-[9px] text-[13px] font-semibold">{p.name}</td>
              <td className="px-3.5 py-[9px] text-[12px] text-[#93a1b8]">{p.position}</td>
              {showCap && (
                <td className="px-3.5 py-[9px]">
                  <span
                    className={`px-[7px] py-0.5 text-[10.5px] font-semibold ${
                      p.rosterStatus === 'TAXI_SQUAD'
                        ? 'bg-[rgba(91,157,255,0.15)] text-[#5b9dff]'
                        : 'bg-[#223049] text-[#a9c0e4]'
                    }`}
                  >
                    {p.rosterStatus === 'TAXI_SQUAD' ? 'TAXI' : 'ROSTER'}
                  </span>
                </td>
              )}
              {showCap && (
                <td className="px-3.5 py-[9px] text-right text-[13px] tabular-nums">{money(p.salary)}</td>
              )}
              {showCap && (
                <td className="px-3.5 py-[9px] text-right text-[13px] tabular-nums text-[#f0b429]">
                  {money(p.capSalary)}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ActionPanel({
  player,
  isFreeAgent,
  signable,
  signFranchise,
  signSalary,
  signYears,
  setSignFranchise,
  setSignSalary,
  setSignYears,
  onStage,
}: {
  player: main.M4Player;
  isFreeAgent: boolean;
  signable: boolean;
  signFranchise: string;
  signSalary: string;
  signYears: string;
  setSignFranchise: (v: string) => void;
  setSignSalary: (v: string) => void;
  setSignYears: (v: string) => void;
  onStage: (kind: Pending['kind'], p: main.M4Player) => void;
}) {
  const toTaxi = player.rosterStatus !== 'TAXI_SQUAD';
  return (
    <>
      <div className="border-b border-[#202a3d] px-[18px] pb-3.5 pt-[18px]">
        <div className="flex items-center gap-[11px]">
          <div className="grid h-10 w-10 place-items-center bg-[#223049] text-[15px] font-bold text-[#5b9dff]">
            {initials(player.name)}
          </div>
          <div>
            <div className="text-[16px] font-bold">{player.name}</div>
            <div className="mt-px text-[11.5px] text-[#93a1b8]">
              {player.position} · {isFreeAgent ? 'Free agent' : player.rosterStatus === 'TAXI_SQUAD' ? 'Taxi squad' : 'Active roster'}
            </div>
          </div>
        </div>
      </div>

      <div className="overflow-y-auto px-[18px] py-4">
        {isFreeAgent ? (
          <>
            <div className="mb-2.5 text-[10.5px] font-semibold uppercase tracking-[0.09em] text-[#64748b]">
              Sign to a franchise
            </div>
            {!signable ? (
              <p className="text-[12px] text-[#64748b]">
                Signing is closed this phase (free agency runs in the offseason and regular season).
              </p>
            ) : (
              <div className="space-y-2">
                <input
                  className="h-9 w-full border border-[#29344a] bg-[#1e2636] px-2.5 text-[13px] outline-none focus:border-[#5b9dff]"
                  placeholder="→ franchise id"
                  value={signFranchise}
                  onChange={(e) => setSignFranchise(e.target.value)}
                />
                <div className="flex gap-2">
                  <input
                    className="h-9 min-w-0 flex-1 border border-[#29344a] bg-[#1e2636] px-2.5 text-[13px] outline-none focus:border-[#5b9dff]"
                    placeholder="salary/yr ($M)"
                    value={signSalary}
                    onChange={(e) => setSignSalary(e.target.value)}
                  />
                  <select
                    className="h-9 border border-[#29344a] bg-[#1e2636] px-2 text-[13px] outline-none focus:border-[#5b9dff]"
                    value={signYears}
                    onChange={(e) => setSignYears(e.target.value)}
                  >
                    <option value="1">1 yr</option>
                    <option value="2">2 yr</option>
                    <option value="3">3 yr</option>
                    <option value="4">4 yr</option>
                  </select>
                </div>
                <button
                  type="button"
                  disabled={!signFranchise.trim() || !signSalary.trim()}
                  onClick={() => onStage('SIGN', player)}
                  className="h-[38px] w-full bg-[#1e2636] text-[13px] font-semibold text-[#e2e8f0] outline outline-1 outline-[#29344a] hover:bg-[rgba(91,157,255,0.12)] hover:text-[#5b9dff] hover:outline-[#5b9dff] disabled:opacity-40"
                >
                  Review signing…
                </button>
              </div>
            )}
          </>
        ) : (
          <>
            <div className="mb-2.5 text-[10.5px] font-semibold uppercase tracking-[0.09em] text-[#64748b]">
              Roster moves
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Act onClick={() => onStage('ROSTER_STATUS', player)}>{toTaxi ? 'Move to Taxi' : 'Activate'}</Act>
              <Act danger onClick={() => onStage('WAIVER', player)}>
                Cut
              </Act>
            </div>
            <p className="mt-3 text-[11px] text-[#64748b]">
              Trade, Tag, Extend, Restructure and the commissioner moves land in later slices.
            </p>
          </>
        )}
      </div>
    </>
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
      onClick={onClick}
      className={`flex h-[38px] items-center justify-center bg-[#1e2636] text-[13px] font-semibold text-[#e2e8f0] outline outline-1 outline-[#29344a] transition-colors ${
        danger
          ? 'hover:bg-[rgba(248,113,113,0.12)] hover:text-[#f87171] hover:outline-[#f87171]'
          : 'hover:bg-[rgba(91,157,255,0.12)] hover:text-[#5b9dff] hover:outline-[#5b9dff]'
      }`}
    >
      {children}
    </button>
  );
}

function Th({ children, right }: { children: React.ReactNode; right?: boolean }) {
  return (
    <th
      className={`sticky top-0 z-[1] border-b border-[#29344a] bg-[#0e1420] px-3.5 py-[9px] text-[10.5px] font-semibold uppercase tracking-[0.07em] ${
        right ? 'text-right' : 'text-left'
      }`}
    >
      {children}
    </th>
  );
}

function Empty({ text }: { text: string }) {
  return <div className="m-auto max-w-[260px] p-8 text-center text-[13px] text-[#64748b]">{text}</div>;
}
