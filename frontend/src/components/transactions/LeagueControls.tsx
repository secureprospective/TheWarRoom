import { useEffect, useRef, useState } from 'react';
import {
  ExecuteTransaction,
  GetLeagueSetting,
  PreviewTransaction,
  SetLeagueSettingOverride,
} from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { useTransactionsStore } from '../../store/transactions';
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
  const phase = useTransactionsStore((s) => s.phase);
  const franchises = useTransactionsStore((s) => s.franchises);
  const loadPhase = useTransactionsStore((s) => s.loadPhase);
  const loadFranchises = useTransactionsStore((s) => s.loadFranchises);
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
  const [tradeDeadlineInput, setTradeDeadlineInput] = useState(''); // datetime-local; '' = clear
  const [tradeDeadlineNote, setTradeDeadlineNote] = useState('');
  // Commissioner inputs.
  const [retireID, setRetireID] = useState('');
  const [deathID, setDeathID] = useState('');
  const [reliefFranchise, setReliefFranchise] = useState('');
  const [reliefAmount, setReliefAmount] = useState('');
  const [reliefReason, setReliefReason] = useState('');
  // Roster-limit settings (Session 0): IR/taxi slots. "0" reads as the mechanic
  // OFF — one override key does both the slot-count and toggle-off job (the
  // rulebook's scopeSetting override already accepts any scalar, no new
  // machinery). Loaded on mount, saved on demand — not a B7 transaction (rulebook
  // writes are the admin-only path, AD-05), so no preview/ConfirmModal here.
  const [taxiSlots, setTaxiSlots] = useState('');
  const [irSlots, setIrSlots] = useState('');
  const [settingsError, setSettingsError] = useState('');
  const [settingsSaving, setSettingsSaving] = useState<'' | 'taxiSquad' | 'injuredReserve'>('');

  useEffect(() => {
    void loadPhase();
    void loadFranchises();
    void loadRosterLimitSettings();
  }, [loadPhase, loadFranchises]);

  async function loadRosterLimitSettings() {
    const [taxi, ir] = await Promise.all([GetLeagueSetting('taxiSquad'), GetLeagueSetting('injuredReserve')]);
    if (taxi.ok) setTaxiSlots(taxi.value);
    if (ir.ok) setIrSlots(ir.value);
  }

  async function saveRosterLimitSetting(key: 'taxiSquad' | 'injuredReserve', value: string) {
    if (!value.trim()) return;
    setSettingsSaving(key);
    setSettingsError('');
    try {
      const res = await SetLeagueSettingOverride(key, value.trim(), 'commissioner roster-limit control');
      if (!res.ok) {
        setSettingsError(res.error);
        return;
      }
      await loadRosterLimitSettings();
    } finally {
      setSettingsSaving('');
    }
  }

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
    try {
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
    } catch (e) {
      // A thrown IPC call (bridge down, panic) must not leave the modal stuck "previewing…"
      // forever — fall into the rejected terminal state so the operator can read it and close.
      if (gen !== stageGen.current) return;
      setPending({
        ...p,
        previewing: false,
        previewOK: false,
        detail: `Couldn't preview this action — the engine was unreachable (${e instanceof Error ? e.message : String(e)}).`,
      });
    }
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
      await loadPhase();
      await loadFranchises();
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

  function stageTradeDeadline() {
    // An empty input CLEARS the deadline server-side (parseTradeDeadline/AppendTradeDeadline both
    // treat '' as "no deadline"); a filled datetime-local value converts to RFC3339 UTC.
    const tradeDeadline = tradeDeadlineInput ? new Date(tradeDeadlineInput).toISOString() : '';
    void stage({
      kind: 'SET_TRADE_DEADLINE',
      title: 'Calendar · trade deadline (§14)',
      subject: tradeDeadline ? `Set trade deadline → ${new Date(tradeDeadline).toLocaleString()}` : 'Clear the trade deadline',
      meta: `Current phase: ${phase}`,
      note: 'Sets or clears the §14 trade deadline. Once passed, every TRADE is rejected until the commissioner clears it. Persists across phase transitions and rollovers.',
      destructive: false,
      request: main.TransactionRequest.createFrom({ kind: 'SET_TRADE_DEADLINE', tradeDeadline, note: tradeDeadlineNote }),
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
    <div
      style={{
        border: '1px solid var(--hairline)',
        background: 'var(--surface-canvas)',
        padding: 32,
        color: 'var(--text-primary)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <h2 style={{ margin: 0, fontSize: 18, fontWeight: 700 }}>League Controls</h2>
        <span
          className="twr-banner twr-banner--caution"
          style={{ display: 'inline-flex', alignItems: 'center', gap: 8, fontWeight: 700 }}
        >
          <span
            aria-hidden
            style={{ width: 7, height: 7, borderRadius: '50%', background: 'var(--amber-base)' }}
          />{' '}
          {phase}
        </span>
      </div>
      <p style={{ margin: '8px 0 0', maxWidth: 560, fontSize: 13, color: 'var(--text-secondary)' }}>
        Commissioner calendar and off-common-path powers. Every action is checked with the engine
        before it commits.
      </p>

      {/* Calendar section */}
      <h3
        style={{
          margin: '28px 0 0',
          fontSize: 11,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.08em',
          color: 'var(--text-secondary)',
        }}
      >
        Season calendar
      </h3>
      <div
        style={{
          marginTop: 12,
          maxWidth: 1120,
          display: 'grid',
          gridTemplateColumns: 'repeat(5, 1fr)',
          gap: 16,
        }}
      >
        <Card title="Roster limits — IR / taxi">
          <p style={{ margin: 0, fontSize: 11, color: 'var(--text-tertiary)' }}>0 turns the mechanic off.</p>
          <label style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
            Taxi squad slots
            <input
              type="number"
              min={0}
              className="twr-input"
              style={{ width: '100%', marginTop: 4 }}
              value={taxiSlots}
              onChange={(e) => setTaxiSlots(e.target.value)}
            />
          </label>
          <Action onClick={() => void saveRosterLimitSetting('taxiSquad', taxiSlots)}>
            {settingsSaving === 'taxiSquad' ? 'Saving…' : 'Save taxi slots'}
          </Action>
          <label style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
            IR slots
            <input
              type="number"
              min={0}
              className="twr-input"
              style={{ width: '100%', marginTop: 4 }}
              value={irSlots}
              onChange={(e) => setIrSlots(e.target.value)}
            />
          </label>
          <Action onClick={() => void saveRosterLimitSetting('injuredReserve', irSlots)}>
            {settingsSaving === 'injuredReserve' ? 'Saving…' : 'Save IR slots'}
          </Action>
          {settingsError && (
            <p style={{ margin: 0, fontSize: 11, color: 'var(--red-base)' }}>{settingsError}</p>
          )}
        </Card>
        <Card title="Advance phase">
          <Select value={toPhase} onChange={setToPhase} options={PHASES} />
          <TextInput value={phaseNote} onChange={setPhaseNote} placeholder="Note (optional)" />
          <Action onClick={stageAdvancePhase}>Advance phase…</Action>
        </Card>
        <Card title="Roll season (§14)">
          <p style={{ margin: 0, fontSize: 11.5, color: 'var(--text-secondary)' }}>
            Requires the current phase to be PLAYOFFS.
          </p>
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
        <Card title="Trade deadline (§14)">
          <input
            type="datetime-local"
            className="twr-input"
            style={{ width: '100%' }}
            value={tradeDeadlineInput}
            onChange={(e) => setTradeDeadlineInput(e.target.value)}
          />
          <p style={{ margin: 0, fontSize: 11, color: 'var(--text-tertiary)' }}>Leave blank to clear the deadline.</p>
          <TextInput value={tradeDeadlineNote} onChange={setTradeDeadlineNote} placeholder="Note (optional)" />
          <Action onClick={stageTradeDeadline}>{tradeDeadlineInput ? 'Set deadline…' : 'Clear deadline…'}</Action>
        </Card>
      </div>

      {/* Commissioner (destructive) section — red divider */}
      <div style={{ marginTop: 32, borderTop: '1px solid var(--red-muted)', paddingTop: 20 }}>
        <h3
          style={{
            margin: 0,
            fontSize: 11,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--red-base)',
          }}
        >
          Commissioner powers — irreversible
        </h3>
        <div
          style={{
            marginTop: 12,
            maxWidth: 860,
            display: 'grid',
            gridTemplateColumns: 'repeat(3, 1fr)',
            gap: 16,
          }}
        >
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
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 10,
        border: '1px solid var(--hairline)',
        background: 'var(--surface-sunken)',
        padding: 16,
      }}
    >
      <div style={{ fontSize: 13, fontWeight: 700, color: 'var(--text-primary)' }}>{title}</div>
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
      className="twr-input"
      style={{ width: '100%' }}
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
      className="twr-select"
      style={{ width: '100%' }}
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
      className={`twr-btn${destructive ? ' twr-btn--danger' : ''}`}
      style={{ marginTop: 'auto', textTransform: 'none', letterSpacing: 'normal', fontSize: 12.5 }}
    >
      {children}
    </button>
  );
}
