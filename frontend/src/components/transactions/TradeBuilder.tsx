import { useEffect, useMemo, useRef, useState } from 'react';
import {
  GetFranchises,
  GetRoster,
  GetLegalOps,
  PreviewTransaction,
  ExecuteTransaction,
} from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { ConfirmModal, type Pending } from './ConfirmModal';
import { money, initials, Th, Empty } from './format';

// TradeBuilder is the M4 slice-3 multi-franchise/multi-leg TRADE surface (design doc §"TRADE gets
// its own builder"). Unlike the subject-centric single-player workspace, a trade spans several
// franchises: the operator browses any roster, ADDS players to a cart, sets each leg's DESTINATION
// franchise, then previews → confirms → commits ONE atomic N-leg swap (the engine's Trade request,
// every leg lands or none). No mflID is typed (D2): names come from the server, the id rides hidden
// on each leg. The whole trade goes through the same D5 quote → D4 re-send-intent commit path the
// single-move ops use, reusing ConfirmModal. Every list is guarded `?? []` (D9).

// TradeLeg is one player's move as staged in the cart: the id crosses the wire, the rest is display.
type TradeLeg = {
  mflID: string;
  name: string;
  position: string;
  fromFranchiseID: string;
  toFranchiseID: string; // '' until the operator picks a destination
};

export function TradeBuilder() {
  const [franchises, setFranchises] = useState<main.M4Franchise[]>([]);
  const [browseFr, setBrowseFr] = useState<string | null>(null);
  const [roster, setRoster] = useState<main.RosterResult | null>(null);
  const [legs, setLegs] = useState<TradeLeg[]>([]);
  const [tradeLegal, setTradeLegal] = useState<boolean | null>(null);
  const [phase, setPhase] = useState('…');
  const [filter, setFilter] = useState('');

  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  // Generation token (same discipline as the single-move workspace, GLM L3): a PreviewTransaction
  // that resolves after the operator dismissed/restaged the modal is discarded, never reopening it.
  const stageGen = useRef(0);

  useEffect(() => {
    void (async () => {
      const [fr, ops] = await Promise.all([GetFranchises(), GetLegalOps()]);
      setFranchises(fr.ok ? (fr.franchises ?? []) : []);
      setTradeLegal(ops.ok ? (ops.kinds ?? []).includes('TRADE') : false);
      setPhase(ops.ok ? ops.phase : `? (${ops.detail})`);
    })();
  }, []);

  async function pickFranchise(id: string) {
    setBrowseFr(id);
    // Clear the roster FIRST so RosterPicker shows "Loading…" (no Add buttons) during the fetch. Without
    // this, the rail highlights the new franchise while the OLD roster is still on screen with live Add
    // buttons — and addLeg would stamp the new franchise onto a player who belongs to the old one (GLM M1).
    setRoster(null);
    setRoster(await GetRoster(id));
  }

  const inCart = useMemo(() => new Set(legs.map((l) => l.mflID)), [legs]);

  function addLeg(p: main.M4Player) {
    // Source the leg's origin from the roster actually rendered, not the rail selection, so it can't
    // drift from what the operator sees (GLM M1). With the pickFranchise fix the two are equal once loaded.
    const from = roster?.franchiseID ?? browseFr;
    if (!from) return;
    setLegs((cur) =>
      // Dedup INSIDE the updater so a same-tick double-click can't double-add (n2); the disabled
      // "Staged" button and the engine's doubled-player rejection are the outer guards.
      cur.some((l) => l.mflID === p.mflID)
        ? cur
        : [...cur, { mflID: p.mflID, name: p.name, position: p.position, fromFranchiseID: from, toFranchiseID: '' }],
    );
  }

  function setDestination(mflID: string, to: string) {
    setLegs((cur) => cur.map((l) => (l.mflID === mflID ? { ...l, toFranchiseID: to } : l)));
  }

  function removeLeg(mflID: string) {
    setLegs((cur) => cur.filter((l) => l.mflID !== mflID));
  }

  function frName(id: string) {
    const f = franchises.find((x) => x.franchiseID === id);
    return f?.name || id;
  }

  // A trade is stageable once it has at least one leg and every leg has a destination that isn't the
  // player's own franchise (an unset destination or a no-op self-move would be rejected in-tx anyway).
  const stageable =
    legs.length > 0 && legs.every((l) => l.toFranchiseID && l.toFranchiseID !== l.fromFranchiseID);

  function cancel() {
    stageGen.current++;
    setPending(null);
  }

  async function stage() {
    if (!stageable) return;
    const req = main.TransactionRequest.createFrom({
      kind: 'TRADE',
      moves: legs.map((l) => ({ mflID: l.mflID, toFranchiseID: l.toFranchiseID })),
    });
    const gen = ++stageGen.current;
    const franchiseCount = new Set(legs.flatMap((l) => [l.fromFranchiseID, l.toFranchiseID])).size;
    const base: Pending = {
      kind: 'TRADE',
      title: 'Trade · atomic swap',
      subject: `${legs.length}-leg trade`,
      meta: `${franchiseCount} franchises · ${phase}`,
      note: 'Every leg lands together or the whole trade rolls back. The engine re-checks each move on confirm; rosters and caps refresh after it commits.',
      destructive: false,
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
      if (gen !== stageGen.current) return; // cancelled/restaged while the preview was in flight (L3)
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
        detail: `Couldn't preview this trade — the engine was unreachable (${e instanceof Error ? e.message : String(e)}).`,
      });
    }
  }

  async function confirm() {
    if (!pending) return;
    setBusy(true);
    try {
      // ExecuteTransaction encodes a rejection as ok:false + detail (it does not throw); the engine
      // can still reject at commit after an OK preview (a phase flip, a roster change). Surface that
      // instead of a false success (GLM L1).
      const res = await ExecuteTransaction(pending.request);
      if (!res.ok) {
        setPending({ ...pending, previewing: false, previewOK: false, detail: res.detail });
        return;
      }
      setPending(null);
      setLegs([]);
      if (browseFr) setRoster(await GetRoster(browseFr));
    } finally {
      setBusy(false);
    }
  }

  const shownFranchises = useMemo(() => {
    const f = filter.trim().toLowerCase();
    if (!f) return franchises;
    return franchises.filter(
      (x) => x.franchiseID.toLowerCase().includes(f) || (x.name || '').toLowerCase().includes(f),
    );
  }, [franchises, filter]);

  if (tradeLegal === false) {
    return (
      <div className="flex h-[calc(100vh-190px)] min-h-[520px] items-center justify-center border border-[#29344a] bg-[#0e1420] text-[13px] text-[#64748b]">
        Trades are not legal in the current phase ({phase}).
      </div>
    );
  }

  return (
    <div className="flex h-[calc(100vh-190px)] min-h-[520px] border border-[#29344a] bg-[#0e1420] text-[#e2e8f0]">
      {/* Left rail — browse any franchise's roster */}
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
            <button
              key={f.franchiseID}
              type="button"
              onClick={() => void pickFranchise(f.franchiseID)}
              className={`flex w-full items-center gap-2.5 border-l-2 px-3 py-[7px] text-left transition-colors ${
                browseFr === f.franchiseID
                  ? 'border-[#5b9dff] bg-[rgba(91,157,255,0.12)]'
                  : 'border-transparent hover:bg-[#161d2b]'
              }`}
            >
              <span
                className={`w-8 text-[11px] font-bold ${
                  browseFr === f.franchiseID ? 'text-[#5b9dff]' : 'text-[#93a1b8]'
                }`}
              >
                {f.franchiseID}
              </span>
              <span className="truncate text-[12.5px]">{f.name || f.franchiseID}</span>
              <span className="ml-auto text-[10.5px] tabular-nums text-[#64748b]">{f.playerCount}</span>
            </button>
          ))}
        </div>
      </nav>

      {/* Center — the browsed roster; click Add to stage a leg */}
      <section className="flex min-w-0 flex-1 flex-col">
        <RosterPicker
          roster={roster}
          browseFr={browseFr}
          label={browseFr ? frName(browseFr) : ''}
          inCart={inCart}
          onAdd={addLeg}
        />
      </section>

      {/* Right — the trade cart */}
      <aside className="flex w-[360px] flex-none flex-col border-l border-[#29344a] bg-[#161d2b]">
        <div className="border-b border-[#202a3d] px-[18px] pb-3.5 pt-[18px]">
          <h2 className="text-[16px] font-bold">Trade builder</h2>
          <p className="mt-0.5 text-[11.5px] text-[#93a1b8]">
            {legs.length === 0 ? 'Add players from any roster to build a swap.' : `${legs.length} leg(s) staged · ${phase}`}
          </p>
        </div>

        <div className="flex-1 overflow-y-auto px-[18px] py-4">
          {legs.length === 0 ? (
            <p className="text-[12px] text-[#64748b]">
              Pick a franchise on the left, then click a player to add a trade leg. Set each player's
              destination franchise here.
            </p>
          ) : (
            <div className="space-y-2.5">
              {legs.map((l) => (
                <LegRow
                  key={l.mflID}
                  leg={l}
                  franchises={franchises}
                  frName={frName}
                  onDestination={(to) => setDestination(l.mflID, to)}
                  onRemove={() => removeLeg(l.mflID)}
                />
              ))}
            </div>
          )}
        </div>

        <div className="border-t border-[#202a3d] px-[18px] py-3.5">
          <button
            type="button"
            disabled={!stageable}
            onClick={() => void stage()}
            className="h-[38px] w-full bg-[#1e2636] text-[13px] font-semibold text-[#e2e8f0] outline outline-1 outline-[#29344a] hover:bg-[rgba(91,157,255,0.12)] hover:text-[#5b9dff] hover:outline-[#5b9dff] disabled:opacity-40"
          >
            Review trade…
          </button>
          {legs.length > 0 && !stageable && (
            <p className="mt-2 text-[11px] text-[#f0b429]">Set a destination for every leg to continue.</p>
          )}
        </div>
      </aside>

      <ConfirmModal pending={pending} busy={busy} onConfirm={() => void confirm()} onCancel={cancel} />
    </div>
  );
}

function RosterPicker({
  roster,
  browseFr,
  label,
  inCart,
  onAdd,
}: {
  roster: main.RosterResult | null;
  browseFr: string | null;
  label: string;
  inCart: Set<string>;
  onAdd: (p: main.M4Player) => void;
}) {
  if (!browseFr) return <Empty text="Pick a franchise from the rail to browse its roster." />;
  if (!roster) return <Empty text="Loading…" />;
  if (!roster.ok) return <Empty text={roster.detail || 'Could not load this roster.'} />;
  const players = roster.players ?? [];

  return (
    <>
      <div className="border-b border-[#202a3d] px-[22px] pb-3.5 pt-4">
        <h1 className="text-[21px] font-bold tracking-[-0.01em]">{label || roster.franchiseID}</h1>
        <div className="mt-2 text-[13px]">
          Cap used <b className="tabular-nums text-[#f0b429]">{money(roster.capUsed)}</b>
        </div>
        {roster.warning && <div className="mt-1 text-[11px] text-[#f0b429]">{roster.warning}</div>}
      </div>
      <div className="flex-1 overflow-auto">
        <table className="w-full border-collapse">
          <thead>
            <tr className="text-[#64748b]">
              <Th>Player</Th>
              <Th>Pos</Th>
              <Th right>Cap</Th>
              <Th right>Trade</Th>
            </tr>
          </thead>
          <tbody>
            {players.map((p) => {
              const staged = inCart.has(p.mflID);
              return (
                <tr key={p.mflID} className="border-b border-[#202a3d]">
                  <td className="px-3.5 py-[9px] text-[13px] font-semibold">{p.name}</td>
                  <td className="px-3.5 py-[9px] text-[12px] text-[#93a1b8]">{p.position}</td>
                  <td className="px-3.5 py-[9px] text-right text-[13px] tabular-nums text-[#f0b429]">
                    {money(p.capSalary)}
                  </td>
                  <td className="px-3.5 py-[9px] text-right">
                    <button
                      type="button"
                      disabled={staged}
                      onClick={() => onAdd(p)}
                      className="border border-[#29344a] px-2.5 py-1 text-[11.5px] font-semibold text-[#93a1b8] hover:border-[#5b9dff] hover:text-[#5b9dff] disabled:opacity-40"
                    >
                      {staged ? 'Staged' : 'Add →'}
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </>
  );
}

function LegRow({
  leg,
  franchises,
  frName,
  onDestination,
  onRemove,
}: {
  leg: TradeLeg;
  franchises: main.M4Franchise[];
  frName: (id: string) => string;
  onDestination: (to: string) => void;
  onRemove: () => void;
}) {
  return (
    <div className="border border-[#29344a] bg-[#1e2636] p-2.5">
      <div className="flex items-center gap-2.5">
        <div className="grid h-8 w-8 flex-none place-items-center bg-[#223049] text-[12px] font-bold text-[#5b9dff]">
          {initials(leg.name)}
        </div>
        <div className="min-w-0 flex-1">
          <div className="truncate text-[13px] font-semibold">{leg.name}</div>
          <div className="text-[11px] text-[#93a1b8]">
            {leg.position} · from {frName(leg.fromFranchiseID)}
          </div>
        </div>
        <button
          type="button"
          onClick={onRemove}
          aria-label="Remove leg"
          className="flex-none px-2 text-[15px] leading-none text-[#64748b] hover:text-[#f87171]"
        >
          ×
        </button>
      </div>
      <div className="mt-2 flex items-center gap-2">
        <span className="text-[11px] text-[#64748b]">→</span>
        <select
          className="h-8 min-w-0 flex-1 border border-[#29344a] bg-[#161d2b] px-2 text-[12.5px] outline-none focus:border-[#5b9dff]"
          value={leg.toFranchiseID}
          onChange={(e) => onDestination(e.target.value)}
          aria-label="Destination franchise"
        >
          <option value="">Destination…</option>
          {franchises
            .filter((f) => f.franchiseID !== leg.fromFranchiseID)
            .map((f) => (
              <option key={f.franchiseID} value={f.franchiseID}>
                {f.name || f.franchiseID}
              </option>
            ))}
        </select>
      </div>
    </div>
  );
}

