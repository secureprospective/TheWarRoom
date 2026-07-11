import { useEffect, useState } from 'react';
import {
  ExecuteTransaction,
  GetFranchiseState,
  GetCurrentPhase,
  GetFreeAgents,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// TransactionsPanel is the B7a dev surface (functional gate): execute a trade or a
// roster-status change through the Coordinator, then re-read a franchise to confirm the
// committed effect on state + cap. Debuggability over polish — a full transaction UI is
// B7b. Every mutation goes through one IPC call (ExecuteTransaction), which the backend
// runs as one atomic transaction; a failed transaction changes nothing.

type Kind =
  | 'TRADE'
  | 'ROSTER_STATUS'
  | 'WAIVER'
  | 'RESTRUCTURE'
  | 'TAG'
  | 'EXTENSION'
  | 'BUYOUT'
  | 'ADVANCE_PHASE'
  | 'ROLLOVER_SEASON'
  | 'RETIREMENT'
  | 'DEATH'
  | 'CAP_RELIEF'
  | 'SIGN';
type Leg = { mflID: string; toFranchiseID: string };
const PHASES = ['OFFSEASON', 'REGULAR_SEASON', 'PLAYOFFS'] as const;

export function TransactionsPanel() {
  const [kind, setKind] = useState<Kind>('TRADE');
  const [legs, setLegs] = useState<Leg[]>([{ mflID: '', toFranchiseID: '' }]);
  const [statusMflID, setStatusMflID] = useState('');
  const [status, setStatus] = useState('ROSTER');
  const [waiverMflID, setWaiverMflID] = useState('');
  const [restructureMflID, setRestructureMflID] = useState('');
  const [moveMillions, setMoveMillions] = useState('');
  const [tagMflID, setTagMflID] = useState('');
  const [extMflID, setExtMflID] = useState('');
  const [addedYears, setAddedYears] = useState('1');
  const [buyoutMflID, setBuyoutMflID] = useState('');
  const [retireMflID, setRetireMflID] = useState('');
  const [deathMflID, setDeathMflID] = useState('');
  const [reliefFranchise, setReliefFranchise] = useState('');
  const [reliefAmount, setReliefAmount] = useState('');
  const [reliefReason, setReliefReason] = useState('');
  const [signMflID, setSignMflID] = useState('');
  const [signFranchise, setSignFranchise] = useState('');
  const [signSalary, setSignSalary] = useState('');
  const [signYears, setSignYears] = useState('1');
  const [freeAgents, setFreeAgents] = useState<string[]>([]);
  const [toPhase, setToPhase] = useState<(typeof PHASES)[number]>('REGULAR_SEASON');
  const [phaseNote, setPhaseNote] = useState('');
  const [result, setResult] = useState<main.TransactionResult | null>(null);
  const [busy, setBusy] = useState(false);

  const [phase, setPhase] = useState<string>('…');
  const [lookupID, setLookupID] = useState('');
  const [franchise, setFranchise] = useState<main.FranchiseStateResult | null>(null);

  async function refreshPhase() {
    const r = await GetCurrentPhase();
    setPhase(r.ok ? r.phase : `? (${r.detail})`);
  }
  useEffect(() => {
    void refreshPhase();
    void refreshFreeAgents();
  }, []);

  async function run() {
    setBusy(true);
    try {
      const req = main.TransactionRequest.createFrom({
        kind,
        moves: kind === 'TRADE' ? legs : [],
        mflID:
          kind === 'ROSTER_STATUS'
            ? statusMflID
            : kind === 'WAIVER'
              ? waiverMflID
              : kind === 'RESTRUCTURE'
                ? restructureMflID
                : kind === 'TAG'
                  ? tagMflID
                  : kind === 'EXTENSION'
                    ? extMflID
                    : kind === 'BUYOUT'
                      ? buyoutMflID
                      : kind === 'RETIREMENT'
                        ? retireMflID
                        : kind === 'DEATH'
                          ? deathMflID
                          : kind === 'SIGN'
                            ? signMflID
                            : '',
        status: kind === 'ROSTER_STATUS' ? status : '',
        moveMillions: kind === 'RESTRUCTURE' ? moveMillions : '',
        addedYears: kind === 'EXTENSION' ? Number(addedYears) || 0 : 0,
        toPhase: kind === 'ADVANCE_PHASE' ? toPhase : '',
        note: kind === 'ADVANCE_PHASE' || kind === 'ROLLOVER_SEASON' ? phaseNote : '',
        franchiseID: kind === 'CAP_RELIEF' ? reliefFranchise : kind === 'SIGN' ? signFranchise : '',
        amountMillions: kind === 'CAP_RELIEF' ? reliefAmount : '',
        reason: kind === 'CAP_RELIEF' ? reliefReason : '',
        salaryMillions: kind === 'SIGN' ? signSalary : '',
        years: kind === 'SIGN' ? Number(signYears) || 0 : 0,
      });
      setResult(await ExecuteTransaction(req));
      if (kind === 'ADVANCE_PHASE' || kind === 'ROLLOVER_SEASON') await refreshPhase();
      // A signing or a rollover changes the pool — refresh it so the gate sees the effect.
      if (kind === 'SIGN' || kind === 'ROLLOVER_SEASON') await refreshFreeAgents();
      if (franchise) setFranchise(await GetFranchiseState(franchise.franchiseID)); // auto-confirm
    } finally {
      setBusy(false);
    }
  }

  async function loadFranchise() {
    if (!lookupID.trim()) return;
    setFranchise(await GetFranchiseState(lookupID.trim()));
  }

  async function refreshFreeAgents() {
    const r = await GetFreeAgents();
    setFreeAgents(r.ok ? r.mflIDs : []);
  }

  return (
    <div className="grid grid-cols-2 gap-6">
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Execute Transaction</h2>
        <div className="flex items-center gap-2 rounded bg-slate-800 px-3 py-1.5 text-sm">
          <span className="text-slate-400">Season phase:</span>
          <span className="font-semibold text-amber-300">{phase}</span>
          <button
            type="button"
            className="ml-auto rounded bg-slate-700 px-2 py-0.5 text-xs hover:bg-slate-600"
            onClick={() => void refreshPhase()}
          >
            refresh
          </button>
        </div>
        <div className="flex flex-wrap gap-2">
          {(
            [
              'TRADE',
              'ROSTER_STATUS',
              'WAIVER',
              'RESTRUCTURE',
              'TAG',
              'EXTENSION',
              'BUYOUT',
              'ADVANCE_PHASE',
              'ROLLOVER_SEASON',
              'RETIREMENT',
              'DEATH',
              'CAP_RELIEF',
              'SIGN',
            ] as Kind[]
          ).map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => setKind(k)}
              className={`rounded px-3 py-1 text-sm ${
                kind === k ? 'bg-emerald-700 text-white' : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
              }`}
            >
              {k === 'TRADE'
                ? 'Trade'
                : k === 'ROSTER_STATUS'
                  ? 'Roster status'
                  : k === 'WAIVER'
                    ? 'Waiver (cut)'
                    : k === 'RESTRUCTURE'
                      ? 'Restructure'
                      : k === 'TAG'
                        ? 'Tag (§9)'
                        : k === 'EXTENSION'
                          ? 'Extend (§10)'
                          : k === 'BUYOUT'
                            ? 'Buyout (§12)'
                            : k === 'ADVANCE_PHASE'
                              ? 'Advance phase'
                              : k === 'ROLLOVER_SEASON'
                                ? 'Roll season (§14)'
                                : k === 'RETIREMENT'
                                ? 'Retire (§13)'
                                : k === 'DEATH'
                                  ? 'Death (§13)'
                                  : k === 'CAP_RELIEF'
                                    ? 'Cap relief (§13)'
                                    : 'Sign (§6)'}
            </button>
          ))}
        </div>

        {kind === 'TRADE' ? (
          <div className="space-y-2">
            {legs.map((leg, i) => (
              <div key={i} className="flex gap-2">
                <input
                  className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
                  placeholder="player mflID"
                  value={leg.mflID}
                  onChange={(e) =>
                    setLegs(legs.map((l, j) => (j === i ? { ...l, mflID: e.target.value } : l)))
                  }
                />
                <input
                  className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
                  placeholder="→ franchise"
                  value={leg.toFranchiseID}
                  onChange={(e) =>
                    setLegs(legs.map((l, j) => (j === i ? { ...l, toFranchiseID: e.target.value } : l)))
                  }
                />
                {legs.length > 1 && (
                  <button
                    type="button"
                    className="rounded bg-slate-700 px-2 text-sm hover:bg-slate-600"
                    onClick={() => setLegs(legs.filter((_, j) => j !== i))}
                  >
                    ✕
                  </button>
                )}
              </div>
            ))}
            <button
              type="button"
              className="rounded bg-slate-700 px-3 py-1 text-sm hover:bg-slate-600"
              onClick={() => setLegs([...legs, { mflID: '', toFranchiseID: '' }])}
            >
              + add leg
            </button>
          </div>
        ) : kind === 'ROSTER_STATUS' ? (
          <div className="flex gap-2">
            <input
              className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
              placeholder="player mflID"
              value={statusMflID}
              onChange={(e) => setStatusMflID(e.target.value)}
            />
            <select
              className="rounded bg-slate-800 px-2 py-1 text-sm"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
            >
              <option value="ROSTER">ROSTER</option>
              <option value="TAXI_SQUAD">TAXI_SQUAD</option>
            </select>
          </div>
        ) : kind === 'WAIVER' ? (
          <div className="space-y-1">
            <input
              className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
              placeholder="player mflID"
              value={waiverMflID}
              onChange={(e) => setWaiverMflID(e.target.value)}
            />
            <p className="text-xs text-slate-400">
              Cut releases the player and charges §8 dead cap (35% × salary × remaining
              years, 50% if restructured) against the current season.
            </p>
          </div>
        ) : kind === 'RESTRUCTURE' ? (
          <div className="space-y-1">
            <div className="flex gap-2">
              <input
                className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="player mflID"
                value={restructureMflID}
                onChange={(e) => setRestructureMflID(e.target.value)}
              />
              <input
                className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="move ($M) e.g. 3"
                value={moveMillions}
                onChange={(e) => setMoveMillions(e.target.value)}
              />
            </div>
            <p className="text-xs text-slate-400">
              §11 restructure: lowers the player's cap-counting salary by the move (owner's
              choice, bounded by the tier max — ≥$3M→$1M, ≥$6M→$2M, ≥$12M→$3M) and flags the
              contract restructured (a later cut then charges 50% dead cap). One per team per
              year, one per contract.
            </p>
          </div>
        ) : kind === 'TAG' ? (
          <div className="space-y-1">
            <input
              className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
              placeholder="player mflID"
              value={tagMflID}
              onChange={(e) => setTagMflID(e.target.value)}
            />
            <p className="text-xs text-slate-400">
              §9 franchise tag: sets the player's salary to the average of the top-5 salaries
              at his position league-wide, floored at 120% of his prior-year salary. The price
              is resolved server-side (nothing to enter). One tag per team per year.
            </p>
          </div>
        ) : kind === 'EXTENSION' ? (
          <div className="space-y-1">
            <div className="flex gap-2">
              <input
                className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="player mflID"
                value={extMflID}
                onChange={(e) => setExtMflID(e.target.value)}
              />
              <select
                className="rounded bg-slate-800 px-2 py-1 text-sm"
                value={addedYears}
                onChange={(e) => setAddedYears(e.target.value)}
              >
                <option value="1">+1 year</option>
                <option value="2">+2 years</option>
                <option value="3">+3 years</option>
              </select>
            </div>
            <p className="text-xs text-slate-400">
              §10 extension: adds 1–3 years (max 6 total) at 150% of the highest-paid remaining
              year, raised to the position floor. Priced server-side (nothing to enter but the
              year count). One extension per team per year; not for UFAs or already-extended
              contracts; unlocks one more restructure.
            </p>
          </div>
        ) : kind === 'BUYOUT' ? (
          <div className="space-y-1">
            <input
              className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
              placeholder="player mflID"
              value={buyoutMflID}
              onChange={(e) => setBuyoutMflID(e.target.value)}
            />
            <p className="text-xs text-slate-400">
              §12 buyout (OFFSEASON only): releases the player and charges dead cap of 60/75/90%
              (for 2/3/4 years remaining) of his average remaining salary against the current
              season. Priced server-side. Two per team per season; 1 or 5+ remaining years route
              to the §13 commissioner path.
            </p>
          </div>
        ) : kind === 'ADVANCE_PHASE' ? (
          <div className="space-y-1">
            <div className="flex gap-2">
              <select
                className="rounded bg-slate-800 px-2 py-1 text-sm"
                value={toPhase}
                onChange={(e) => setToPhase(e.target.value as (typeof PHASES)[number])}
              >
                {PHASES.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
              <input
                className="w-40 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="note (optional)"
                value={phaseNote}
                onChange={(e) => setPhaseNote(e.target.value)}
              />
            </div>
            <p className="text-xs text-slate-400">
              D3 season phase (commissioner): appends a transition to the append-only phase log.
              Gates which ops are legal (e.g. §12 buyouts are OFFSEASON-only). A no-op (already in
              the target phase) is rejected.
            </p>
          </div>
        ) : kind === 'ROLLOVER_SEASON' ? (
          <div className="space-y-1">
            <input
              className="w-40 rounded bg-slate-800 px-2 py-1 text-sm"
              placeholder="note (optional)"
              value={phaseNote}
              onChange={(e) => setPhaseNote(e.target.value)}
            />
            <p className="text-xs text-slate-400">
              §14 season rollover (commissioner, PLAYOFFS only): closes the season, moving
              PLAYOFFS(N) → OFFSEASON(N+1). The cap rolls to next year's contract cells, per-season
              op limits reset, and this season's dead cap / cap relief drop off. One-way — the season
              never moves backward.
            </p>
          </div>
        ) : kind === 'RETIREMENT' ? (
          <div className="space-y-1">
            <input
              className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
              placeholder="player mflID"
              value={retireMflID}
              onChange={(e) => setRetireMflID(e.target.value)}
            />
            <p className="text-xs text-slate-400">
              §13 retirement: releases the player and charges dead cap of 30% of his remaining
              contract (the salary of every year after the current season) against the current
              season. Resolved server-side. Any phase, no per-season limit.
            </p>
          </div>
        ) : kind === 'DEATH' ? (
          <div className="space-y-1">
            <input
              className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
              placeholder="player mflID"
              value={deathMflID}
              onChange={(e) => setDeathMflID(e.target.value)}
            />
            <p className="text-xs text-slate-400">
              §13 Gaines Adams Rule (death): removes the player from his roster with NO cap penalty
              — his salary leaves and no dead cap lands. Any phase.
            </p>
          </div>
        ) : kind === 'CAP_RELIEF' ? (
          <div className="space-y-1">
            <div className="flex gap-2">
              <input
                className="w-28 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="franchise id"
                value={reliefFranchise}
                onChange={(e) => setReliefFranchise(e.target.value)}
              />
              <input
                className="w-28 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="relief ($M) e.g. 3"
                value={reliefAmount}
                onChange={(e) => setReliefAmount(e.target.value)}
              />
              <input
                className="w-40 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="reason (required)"
                value={reliefReason}
                onChange={(e) => setReliefReason(e.target.value)}
              />
            </div>
            <p className="text-xs text-slate-400">
              §13 Cap Relief Appeal (commissioner): reduces the franchise's cap hit by the relief
              amount (career-ending injury, recurring injury, behavioral suspension). Appends a
              credit to the append-only cap-relief ledger; CapUsed drops (floored at $0). Any phase.
            </p>
          </div>
        ) : (
          <div className="space-y-1">
            <div className="flex flex-wrap gap-2">
              <input
                className="w-28 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="player mflID"
                value={signMflID}
                onChange={(e) => setSignMflID(e.target.value)}
              />
              <input
                className="w-28 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="→ franchise"
                value={signFranchise}
                onChange={(e) => setSignFranchise(e.target.value)}
              />
              <input
                className="w-28 rounded bg-slate-800 px-2 py-1 text-sm"
                placeholder="salary/yr ($M)"
                value={signSalary}
                onChange={(e) => setSignSalary(e.target.value)}
              />
              <select
                className="rounded bg-slate-800 px-2 py-1 text-sm"
                value={signYears}
                onChange={(e) => setSignYears(e.target.value)}
              >
                <option value="1">1 year</option>
                <option value="2">2 years</option>
                <option value="3">3 years</option>
                <option value="4">4 years</option>
              </select>
            </div>
            <p className="text-xs text-slate-400">
              §6 free-agency signing (offseason + regular season): rosters a free agent on a NEW flat
              1–4 year contract at the entered salary. Only a FREE_AGENT is signable (retired/deceased
              barred); a bought-out player is locked until the following offseason (§12). The cap is
              not blocked (it just reflects the signing). The min-salary floor is skipped until
              experience data exists. See the free-agent pool on the right.
            </p>
          </div>
        )}

        <button
          type="button"
          disabled={busy}
          onClick={() => void run()}
          className="rounded bg-emerald-700 px-4 py-1.5 text-sm font-medium text-white hover:bg-emerald-600 disabled:opacity-50"
        >
          {busy ? 'Executing…' : 'Execute'}
        </button>

        {result && (
          <pre
            className={`whitespace-pre-wrap rounded p-3 text-xs ${
              result.ok ? 'bg-emerald-950 text-emerald-200' : 'bg-red-950 text-red-200'
            }`}
          >
            {result.ok
              ? `OK · ${result.kind} · ${result.playersAffected} player(s) · ${result.at}`
              : `FAILED · ${result.detail}`}
          </pre>
        )}
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Franchise state (confirm)</h2>
        <div className="flex gap-2">
          <input
            className="w-32 rounded bg-slate-800 px-2 py-1 text-sm"
            placeholder="franchise id"
            value={lookupID}
            onChange={(e) => setLookupID(e.target.value)}
          />
          <button
            type="button"
            className="rounded bg-slate-700 px-3 py-1 text-sm hover:bg-slate-600"
            onClick={() => void loadFranchise()}
          >
            Load
          </button>
        </div>
        {franchise &&
          (franchise.ok ? (
            <div className="rounded bg-slate-800 p-3 text-sm">
              <div className="mb-2 font-medium">
                {franchise.franchiseID} · cap used {franchise.capUsed}
              </div>
              <ul className="space-y-1 text-xs text-slate-300">
                {franchise.players.map((p) => (
                  <li key={p.mflID}>
                    {p.mflID} · {p.rosterStatus} · ${p.salary}
                    {p.capSalary !== p.salary ? ` (cap $${p.capSalary})` : ''}
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <p className="text-xs text-red-300">{franchise.detail}</p>
          ))}

        <div className="flex items-center gap-2 pt-2">
          <h3 className="text-sm font-semibold">Free-agent pool</h3>
          <span className="text-xs text-slate-400">({freeAgents.length})</span>
          <button
            type="button"
            className="ml-auto rounded bg-slate-700 px-2 py-0.5 text-xs hover:bg-slate-600"
            onClick={() => void refreshFreeAgents()}
          >
            refresh
          </button>
        </div>
        {freeAgents.length === 0 ? (
          <p className="text-xs text-slate-500">No free agents (sign or roll a season to populate).</p>
        ) : (
          <ul className="max-h-40 space-y-0.5 overflow-y-auto rounded bg-slate-800 p-2 text-xs text-slate-300">
            {freeAgents.map((id) => (
              <li key={id}>
                <button
                  type="button"
                  className="hover:text-emerald-300"
                  onClick={() => {
                    setKind('SIGN');
                    setSignMflID(id);
                  }}
                >
                  {id}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
