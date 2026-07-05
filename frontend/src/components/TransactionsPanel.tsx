import { useState } from 'react';
import { ExecuteTransaction, GetFranchiseState } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// TransactionsPanel is the B7a dev surface (functional gate): execute a trade or a
// roster-status change through the Coordinator, then re-read a franchise to confirm the
// committed effect on state + cap. Debuggability over polish — a full transaction UI is
// B7b. Every mutation goes through one IPC call (ExecuteTransaction), which the backend
// runs as one atomic transaction; a failed transaction changes nothing.

type Kind = 'TRADE' | 'ROSTER_STATUS' | 'WAIVER' | 'RESTRUCTURE' | 'TAG';
type Leg = { mflID: string; toFranchiseID: string };

export function TransactionsPanel() {
  const [kind, setKind] = useState<Kind>('TRADE');
  const [legs, setLegs] = useState<Leg[]>([{ mflID: '', toFranchiseID: '' }]);
  const [statusMflID, setStatusMflID] = useState('');
  const [status, setStatus] = useState('ROSTER');
  const [waiverMflID, setWaiverMflID] = useState('');
  const [restructureMflID, setRestructureMflID] = useState('');
  const [moveMillions, setMoveMillions] = useState('');
  const [tagMflID, setTagMflID] = useState('');
  const [result, setResult] = useState<main.TransactionResult | null>(null);
  const [busy, setBusy] = useState(false);

  const [lookupID, setLookupID] = useState('');
  const [franchise, setFranchise] = useState<main.FranchiseStateResult | null>(null);

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
                  : '',
        status: kind === 'ROSTER_STATUS' ? status : '',
        moveMillions: kind === 'RESTRUCTURE' ? moveMillions : '',
      });
      setResult(await ExecuteTransaction(req));
      if (franchise) setFranchise(await GetFranchiseState(franchise.franchiseID)); // auto-confirm
    } finally {
      setBusy(false);
    }
  }

  async function loadFranchise() {
    if (!lookupID.trim()) return;
    setFranchise(await GetFranchiseState(lookupID.trim()));
  }

  return (
    <div className="grid grid-cols-2 gap-6">
      <section className="space-y-3">
        <h2 className="text-lg font-semibold">Execute Transaction</h2>
        <div className="flex gap-2">
          {(['TRADE', 'ROSTER_STATUS', 'WAIVER', 'RESTRUCTURE', 'TAG'] as Kind[]).map((k) => (
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
                      : 'Tag (§9)'}
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
        ) : (
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
      </section>
    </div>
  );
}
