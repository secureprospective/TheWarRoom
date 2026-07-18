import { useEffect, useRef, useState } from 'react';
import {
  ExecuteTransaction,
  GetCurrentPhase,
  GetFranchises,
  PreviewTransaction,
} from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { ConfirmModal, type Pending } from './ConfirmModal';

// LeagueControls is the commissioner off-common-path surface (design D6): the ops that are NOT
// per-player asset moves live here, away from the subject-centric Transactions workspace. Two
// groups — the season CALENDAR (advance phase / roll season / signing window) and, under a red
// divider, the destructive COMMISSIONER powers (retirement / death / cap relief). Every op runs
// the same D5 preview → ConfirmModal → D4 re-send-intent path as the workspace, reusing the shared
// modal so a commissioner action can never commit without the engine confirming it first. This is
// the D8 cutover target for these ops — the raw dev panel keeps them only until this surface lands.

const PHASES = ['OFFSEASON', 'REGULAR_SEASON', 'PLAYOFFS'] as const;

export function LeagueControls() {
  const [phase, setPhase] = useState('…');
  const [franchises, setFranchises] = useState<main.M4Franchise[]>([]);
  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  // A ref token that a stage() stamps and confirm()/cancel() compare, so a preview that resolves
  // after the operator cancels or restages is discarded instead of reopening the modal (GLM L3).
  const stageGen = useRef(0);

  // Calendar inputs.
  const [toPhase, setToPhase] = useState<string>('REGULAR_SEASON');
  const [phaseNote, setPhaseNote] = useState('');
  const [rolloverNote, setRolloverNote] = useState('');
  const [windowOpen, setWindowOpen] = useState(true);
  const [windowNote, setWindowNote] = useState('');
  // Commissioner inputs.
  const [retireID, setRetireID] = useState('');
  const [deathID, setDeathID] = useState('');
  const [reliefFranchise, setReliefFranchise] = useState('');
  const [reliefAmount, setReliefAmount] = useState('');
  const [reliefReason, setReliefReason] = useState('');

  async function refreshPhase() {
    const r = await GetCurrentPhase();
    setPhase(r.ok ? r.phase : `? (${r.detail})`);
  }

  async function refreshFranchises() {
    const r = await GetFranchises();
    setFranchises(r.ok ? (r.franchises ?? []) : []);
  }

  useEffect(() => {
    void refreshPhase();
    void refreshFranchises();
  }, []);

  function frLabel(id: string) {
    const f = franchises.find((x) => x.franchiseID === id);
    return f ? `${f.franchiseID} · ${f.name || f.franchiseID}` : id;
  }

  function cancel() {
    stageGen.current++;
    setPending(null);
  }

  // stage builds the request, opens the modal in its "previewing" state, dry-runs it through the
  // engine (D5), and folds the authoritative result back in — unless a newer stage/cancel has since
  // bumped the token (L3). It never encodes the outcome as input to confirm; confirm re-sends intent.
  async function stage(
    base: Omit<Pending, 'previewing' | 'previewOK' | 'detail' | 'playersAffected' | 'capDeltas'>,
  ) {
    const gen = ++stageGen.current;
    const p: Pending = {
      ...base,
      previewing: true,
      previewOK: null,
      detail: '',
      playersAffected: 0,
      capDeltas: [],
    };
    setPending(p);
    const res = await PreviewTransaction(base.request);
    if (gen !== stageGen.current) return; // cancelled/restaged mid-flight
    setPending({
      ...p,
      previewing: false,
      previewOK: res.ok,
      detail: res.detail,
      playersAffected: res.playersAffected,
      capDeltas: res.capDeltas ?? [],
    });
  }

  async function confirm() {
    if (!pending) return;
    setBusy(true);
    try {
      // ExecuteTransaction encodes a rejection as ok:false + detail (no throw); the engine can still
      // reject at commit after an OK preview (e.g. a concurrent phase flip). Surface it (GLM L1).
      const res = await ExecuteTransaction(pending.request);
      if (!res.ok) {
        setPending({ ...pending, previewing: false, previewOK: false, detail: res.detail });
        return;
      }
      setPending(null);
      // a calendar op moved the phase; a commissioner op (retirement/death) removed a player, so the
      // cap-relief picker's counts would go stale (GLM L2) — refresh both, cheap either way.
      await refreshPhase();
      await refreshFranchises();
    } finally {
      setBusy(false);
    }
  }

  function stageAdvancePhase() {
    void stage({
      kind: 'ADVANCE_PHASE',
      title: 'Calendar · advance phase',
      subject: `Set phase → ${toPhase}`,
      meta: `Current: ${phase}`,
      note: 'Moves the league to the selected phase, gating which ops are legal. A commissioner correction/rollback is allowed to any phase.',
      destructive: false,
      request: main.TransactionRequest.createFrom({ kind: 'ADVANCE_PHASE', toPhase, note: phaseNote }),
    });
  }

  function stageRollover() {
    void stage({
      kind: 'ROLLOVER_SEASON',
      title: 'Calendar · roll season',
      subject: 'Roll the season over (§14)',
      meta: `Current: ${phase} · requires PLAYOFFS`,
      note: 'Advances the league from PLAYOFFS to next OFFSEASON: rolls contracts a year, expires UFAs. Monotonic — moves the season forward by one and cannot be undone.',
      destructive: true,
      request: main.TransactionRequest.createFrom({ kind: 'ROLLOVER_SEASON', note: rolloverNote }),
    });
  }

  function stageSigningWindow() {
    void stage({
      kind: 'SET_SIGNING_WINDOW',
      title: 'Calendar · signing window',
      subject: windowOpen ? 'Open the signing window' : 'Close the signing window',
      meta: `Current phase: ${phase}`,
      note: 'Toggles the §6 free-agency signing window. Persists across phase transitions and rollovers until the commissioner toggles it back.',
      destructive: false,
      request: main.TransactionRequest.createFrom({ kind: 'SET_SIGNING_WINDOW', windowOpen, note: windowNote }),
    });
  }

  function stageRetirement() {
    if (!retireID.trim()) return;
    void stage({
      kind: 'RETIREMENT',
      title: 'Commissioner · retirement (§13)',
      subject: `Retire player ${retireID.trim()}`,
      meta: 'Terminal — the player leaves his roster',
      note: 'Removes the player from his roster as RETIRED, relieving his cap. Any dead-cap charge is recorded separately. There is no undo.',
      destructive: true,
      request: main.TransactionRequest.createFrom({ kind: 'RETIREMENT', mflID: retireID.trim() }),
    });
  }

  function stageDeath() {
    if (!deathID.trim()) return;
    void stage({
      kind: 'DEATH',
      title: 'Commissioner · death (§13)',
      subject: `Record death — player ${deathID.trim()}`,
      meta: 'Terminal — the player leaves his roster',
      note: 'Removes the player from his roster as DECEASED, relieving his cap. There is no undo.',
      destructive: true,
      request: main.TransactionRequest.createFrom({ kind: 'DEATH', mflID: deathID.trim() }),
    });
  }

  function stageCapRelief() {
    if (!reliefFranchise || !reliefAmount.trim() || !reliefReason.trim()) return;
    void stage({
      kind: 'CAP_RELIEF',
      title: 'Commissioner · cap relief (§13)',
      subject: `Grant $${reliefAmount.trim()}M relief`,
      meta: frLabel(reliefFranchise),
      note: 'Credits the franchise a cap reduction for the current league year (career-ending injury, recurring injury, behavioral suspension). Recorded as an append-only credit.',
      destructive: false,
      request: main.TransactionRequest.createFrom({
        kind: 'CAP_RELIEF',
        franchiseID: reliefFranchise,
        amountMillions: reliefAmount.trim(),
        reason: reliefReason.trim(),
      }),
    });
  }

  return (
    <div className="border border-[#29344a] bg-[#0e1420] p-8 text-[#e2e8f0]">
      <div className="flex items-center gap-3">
        <h2 className="text-[18px] font-bold">League Controls</h2>
        <div className="inline-flex items-center gap-2 border border-[rgba(240,180,41,0.35)] bg-[rgba(240,180,41,0.14)] px-2.5 py-1 text-[11px] font-bold uppercase tracking-[0.07em] text-[#f0b429]">
          <span className="h-[7px] w-[7px] rounded-full bg-[#f0b429]" /> {phase}
        </div>
      </div>
      <p className="mt-2 max-w-[560px] text-[13px] text-[#93a1b8]">
        Commissioner calendar and off-common-path powers. Every action is checked with the engine
        before it commits.
      </p>

      {/* Calendar section */}
      <h3 className="mt-7 text-[11px] font-semibold uppercase tracking-[0.08em] text-[#5b9dff]">
        Season calendar
      </h3>
      <div className="mt-3 grid max-w-[860px] grid-cols-3 gap-4">
        <Card title="Advance phase">
          <Select value={toPhase} onChange={setToPhase} options={PHASES} />
          <TextInput value={phaseNote} onChange={setPhaseNote} placeholder="Note (optional)" />
          <Action onClick={stageAdvancePhase}>Advance phase…</Action>
        </Card>
        <Card title="Roll season (§14)">
          <p className="text-[11.5px] text-[#93a1b8]">Requires the current phase to be PLAYOFFS.</p>
          <TextInput value={rolloverNote} onChange={setRolloverNote} placeholder="Note (optional)" />
          <Action onClick={stageRollover} destructive>
            Roll season over…
          </Action>
        </Card>
        <Card title="Signing window (§6)">
          <Select
            value={windowOpen ? 'open' : 'closed'}
            onChange={(v) => setWindowOpen(v === 'open')}
            options={['open', 'closed'] as const}
          />
          <TextInput value={windowNote} onChange={setWindowNote} placeholder="Note (optional)" />
          <Action onClick={stageSigningWindow}>Apply window…</Action>
        </Card>
      </div>

      {/* Commissioner (destructive) section — red divider */}
      <div className="mt-8 border-t border-[rgba(248,113,113,0.35)] pt-5">
        <h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-[#f87171]">
          Commissioner powers — irreversible
        </h3>
        <div className="mt-3 grid max-w-[860px] grid-cols-3 gap-4">
          <Card title="Retirement (§13)">
            <TextInput value={retireID} onChange={setRetireID} placeholder="Player MFL id" />
            <Action onClick={stageRetirement} destructive>
              Retire player…
            </Action>
          </Card>
          <Card title="Death (§13)">
            <TextInput value={deathID} onChange={setDeathID} placeholder="Player MFL id" />
            <Action onClick={stageDeath} destructive>
              Record death…
            </Action>
          </Card>
          <Card title="Cap relief (§13)">
            <Select
              value={reliefFranchise}
              onChange={setReliefFranchise}
              options={['', ...franchises.map((f) => f.franchiseID)]}
              labels={(id) => (id === '' ? 'Select franchise…' : frLabel(id))}
            />
            <TextInput value={reliefAmount} onChange={setReliefAmount} placeholder="Amount ($M)" />
            <TextInput value={reliefReason} onChange={setReliefReason} placeholder="Reason (required)" />
            <Action onClick={stageCapRelief}>Grant relief…</Action>
          </Card>
        </div>
      </div>

      <ConfirmModal pending={pending} busy={busy} onConfirm={() => void confirm()} onCancel={cancel} />
    </div>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-2.5 border border-[#29344a] bg-[#0c121d] p-4">
      <div className="text-[13px] font-bold">{title}</div>
      {children}
    </div>
  );
}

function TextInput({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
}) {
  return (
    <input
      className="h-[32px] w-full border border-[#29344a] bg-[#161d2b] px-2.5 text-[12.5px] outline-none focus:border-[#5b9dff]"
      placeholder={placeholder}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}

function Select({
  value,
  onChange,
  options,
  labels,
}: {
  value: string;
  onChange: (v: string) => void;
  options: readonly string[];
  labels?: (v: string) => string;
}) {
  return (
    <select
      className="h-[32px] w-full border border-[#29344a] bg-[#161d2b] px-2 text-[12.5px] outline-none focus:border-[#5b9dff]"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    >
      {options.map((o) => (
        <option key={o} value={o}>
          {labels ? labels(o) : o}
        </option>
      ))}
    </select>
  );
}

function Action({
  onClick,
  destructive,
  children,
}: {
  onClick: () => void;
  destructive?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`mt-auto h-[34px] text-[12.5px] font-semibold outline outline-1 disabled:opacity-40 ${
        destructive
          ? 'bg-[#1e2636] text-[#f87171] outline-[rgba(248,113,113,0.4)] hover:bg-[rgba(248,113,113,0.12)]'
          : 'bg-[#1e2636] text-[#e2e8f0] outline-[#29344a] hover:bg-[rgba(91,157,255,0.12)] hover:text-[#5b9dff] hover:outline-[#5b9dff]'
      }`}
    >
      {children}
    </button>
  );
}
