import { useEffect, useMemo, useRef, useState } from 'react';
import { PreviewTransaction, ExecuteTransaction } from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { useTransactionsStore } from '../../store/transactions';
import { ConfirmModal, type Pending } from './ConfirmModal';
import { money, initials, Empty } from './format';

// TradeBuilder is the M4 slice-3 multi-franchise/multi-leg TRADE surface (design doc §"TRADE gets
// its own builder"). Unlike the subject-centric single-player workspace, a trade spans several
// franchises: the operator browses any roster, ADDS players to a cart, sets each leg's DESTINATION
// franchise, then previews → confirms → commits ONE atomic N-leg swap (the engine's Trade request,
// every leg lands or none). No mflID is typed (D2): names come from the server, the id rides hidden
// on each leg. The whole trade goes through the same D5 quote → D4 re-send-intent commit path the
// single-move ops use, reusing ConfirmModal. Every list is guarded `?? []` (D9).
//
// B-2 restyle: Session-C token contract; the browsed roster migrated onto the shared .twr-board*
// grammar (roster reuses M1/M2's row grammar per the wireframe), rail + cart on tokens/controls.

// TradeLeg is one player's move as staged in the cart: the id crosses the wire, the rest is display.
type TradeLeg = {
  mflID: string;
  name: string;
  position: string;
  fromFranchiseID: string;
  toFranchiseID: string; // '' until the operator picks a destination
};

// Browsed-roster board template (Player·Pos·Cap·Add).
const ROSTER_COLS = '1fr 46px 84px 74px';

export function TradeBuilder() {
  const franchises = useTransactionsStore((s) => s.franchises);
  const roster = useTransactionsStore((s) => s.roster);
  const phase = useTransactionsStore((s) => s.legalOpsPhase);
  const loadFranchises = useTransactionsStore((s) => s.loadFranchises);
  const loadLegalOps = useTransactionsStore((s) => s.loadLegalOps);
  const loadRoster = useTransactionsStore((s) => s.loadRoster);
  const clearRoster = useTransactionsStore((s) => s.clearRoster);

  const [browseFr, setBrowseFr] = useState<string | null>(null);
  const [legs, setLegs] = useState<TradeLeg[]>([]);
  const [tradeLegal, setTradeLegal] = useState<boolean | null>(null);
  const [filter, setFilter] = useState('');
  // Rationale is REQUIRED (Alpha-scope panel lock — every trade must carry the commissioner's
  // reason); PicksNote is optional free-text (no pick-ownership ledger yet, deliberately
  // unvalidated on the server too).
  const [rationale, setRationale] = useState('');
  const [picksNote, setPicksNote] = useState('');

  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  // Generation token (same discipline as the single-move workspace, GLM L3): a PreviewTransaction
  // that resolves after the operator dismissed/restaged the modal is discarded, never reopening it.
  const stageGen = useRef(0);

  useEffect(() => {
    void (async () => {
      await Promise.all([loadFranchises(), loadLegalOps()]);
      // tradeLegal stays local + three-state (null = still loading, so the "not legal" screen
      // never flashes before the fetch resolves) — read the store's freshly-loaded legalOps
      // once the fetch settles, matching the original ops.ok-gated computation exactly.
      setTradeLegal(useTransactionsStore.getState().legalOps.includes('TRADE'));
    })();
  }, [loadFranchises, loadLegalOps]);

  async function pickFranchise(id: string) {
    setBrowseFr(id);
    // Clear the roster FIRST so RosterPicker shows "Loading…" (no Add buttons) during the fetch. Without
    // this, the rail highlights the new franchise while the OLD roster is still on screen with live Add
    // buttons — and addLeg would stamp the new franchise onto a player who belongs to the old one (GLM M1).
    clearRoster();
    await loadRoster(id);
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

  // A trade is stageable once it has at least one leg, every leg has a destination that isn't the
  // player's own franchise (an unset destination or a no-op self-move would be rejected in-tx anyway),
  // and a Rationale is present (the engine rejects an empty one, so gate it here too — no wasted
  // preview round-trip on a request that will just bounce).
  const stageable =
    legs.length > 0 &&
    legs.every((l) => l.toFranchiseID && l.toFranchiseID !== l.fromFranchiseID) &&
    rationale.trim() !== '';

  function cancel() {
    stageGen.current++;
    setPending(null);
  }

  async function stage() {
    if (!stageable) return;
    const req = main.TransactionRequest.createFrom({
      kind: 'TRADE',
      moves: legs.map((l) => ({ mflID: l.mflID, toFranchiseID: l.toFranchiseID })),
      rationale,
      picksNote,
    });
    const gen = ++stageGen.current;
    const franchiseCount = new Set(legs.flatMap((l) => [l.fromFranchiseID, l.toFranchiseID])).size;
    const base: Pending = {
      kind: 'TRADE',
      title: 'Trade · atomic swap',
      subject: `${legs.length}-leg trade`,
      meta: `${franchiseCount} franchises · ${phase}`,
      note:
        'Every leg lands together or the whole trade rolls back. The engine re-checks each move on confirm; rosters and caps refresh after it commits.' +
        ` Rationale: ${rationale}` +
        (picksNote ? ` · Picks: ${picksNote}` : ''),
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
      setRationale('');
      setPicksNote('');
      if (browseFr) await loadRoster(browseFr);
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
      <div
        style={{
          display: 'flex',
          height: 'calc(100vh - 190px)',
          minHeight: 520,
          alignItems: 'center',
          justifyContent: 'center',
          border: '1px solid var(--hairline)',
          background: 'var(--surface-canvas)',
          fontSize: 13,
          color: 'var(--text-tertiary)',
        }}
      >
        Trades are not legal in the current phase ({phase}).
      </div>
    );
  }

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
      {/* Left rail — browse any franchise's roster */}
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
              abbr={f.franchiseID}
              label={f.name || f.franchiseID}
              count={f.playerCount}
              active={browseFr === f.franchiseID}
              onClick={() => void pickFranchise(f.franchiseID)}
            />
          ))}
        </div>
      </nav>

      {/* Center — the browsed roster; click Add to stage a leg */}
      <section style={{ display: 'flex', minWidth: 0, flex: 1, flexDirection: 'column' }}>
        <RosterPicker
          roster={roster}
          browseFr={browseFr}
          label={browseFr ? frName(browseFr) : ''}
          inCart={inCart}
          onAdd={addLeg}
        />
      </section>

      {/* Right — the trade cart */}
      <aside
        style={{
          display: 'flex',
          width: 360,
          flex: 'none',
          flexDirection: 'column',
          borderLeft: '1px solid var(--hairline)',
          background: 'var(--surface-tile)',
        }}
      >
        <div style={{ borderBottom: '1px solid var(--hairline)', padding: '18px 18px 14px' }}>
          <h2 style={{ margin: 0, fontSize: 16, fontWeight: 700 }}>Trade builder</h2>
          <p style={{ margin: '2px 0 0', fontSize: 11.5, color: 'var(--text-secondary)' }}>
            {legs.length === 0
              ? 'Add players from any roster to build a swap.'
              : `${legs.length} leg(s) staged · ${phase}`}
          </p>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: '16px 18px' }}>
          {legs.length === 0 ? (
            <p style={{ margin: 0, fontSize: 12, color: 'var(--text-tertiary)' }}>
              Pick a franchise on the left, then click a player to add a trade leg. Set each player's
              destination franchise here.
            </p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
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

        <div style={{ borderTop: '1px solid var(--hairline)', padding: '14px 18px' }}>
          <label style={{ display: 'block', marginBottom: 10 }}>
            <span style={{ display: 'block', marginBottom: 4, fontSize: 11, color: 'var(--text-secondary)' }}>
              Rationale <span style={{ color: 'var(--amber-base)' }}>(required)</span>
            </span>
            <textarea
              className="twr-input"
              style={{ width: '100%', minHeight: 52, resize: 'vertical', fontSize: 12 }}
              placeholder="Why is this trade happening?"
              value={rationale}
              onChange={(e) => setRationale(e.target.value)}
            />
          </label>
          <label style={{ display: 'block', marginBottom: 10 }}>
            <span style={{ display: 'block', marginBottom: 4, fontSize: 11, color: 'var(--text-secondary)' }}>
              Draft picks involved <span style={{ color: 'var(--text-tertiary)' }}>(optional, free text)</span>
            </span>
            <textarea
              className="twr-input"
              style={{ width: '100%', minHeight: 40, resize: 'vertical', fontSize: 12 }}
              placeholder="e.g. 2027 1st (Franchise X) to Franchise Y"
              value={picksNote}
              onChange={(e) => setPicksNote(e.target.value)}
            />
          </label>
          <button
            type="button"
            className="twr-btn"
            style={{ width: '100%', padding: '10px 12px' }}
            disabled={!stageable}
            onClick={() => void stage()}
          >
            Review trade…
          </button>
          {legs.length > 0 && legs.every((l) => l.toFranchiseID && l.toFranchiseID !== l.fromFranchiseID) && !rationale.trim() && (
            <p style={{ margin: '8px 0 0', fontSize: 11, color: 'var(--amber-base)' }}>
              A rationale is required to continue.
            </p>
          )}
          {legs.length > 0 && !legs.every((l) => l.toFranchiseID && l.toFranchiseID !== l.fromFranchiseID) && (
            <p style={{ margin: '8px 0 0', fontSize: 11, color: 'var(--amber-base)' }}>
              Set a destination for every leg to continue.
            </p>
          )}
        </div>
      </aside>

      <ConfirmModal pending={pending} busy={busy} onConfirm={() => void confirm()} onCancel={cancel} />
    </div>
  );
}

// RailRow — a franchise entry in the browse rail (neutral selection axis, per Session-C).
function RailRow({
  abbr,
  label,
  count,
  active,
  onClick,
}: {
  abbr: string;
  label: string;
  count?: number;
  active: boolean;
  onClick: () => void;
}) {
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
      <span
        style={{
          width: 32,
          fontSize: 11,
          fontWeight: 700,
          fontFamily: 'var(--mono)',
          color: active ? 'var(--text-primary)' : 'var(--text-secondary)',
        }}
      >
        {abbr}
      </span>
      <span
        style={{
          flex: 1,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          fontSize: 12.5,
          color: 'var(--text-primary)',
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
      <div style={{ borderBottom: '1px solid var(--hairline)', padding: '16px 22px 14px' }}>
        <h1 style={{ margin: 0, fontSize: 21, fontWeight: 700, letterSpacing: '-0.01em' }}>
          {label || roster.franchiseID}
        </h1>
        <div style={{ marginTop: 8, fontSize: 13, color: 'var(--text-secondary)' }}>
          Cap used{' '}
          <b style={{ fontFamily: 'var(--mono)', fontVariantNumeric: 'tabular-nums', color: 'var(--amber-base)' }}>
            {money(roster.capUsed)}
          </b>
        </div>
        {roster.warning && (
          <div style={{ marginTop: 4, fontSize: 11, color: 'var(--amber-base)' }}>{roster.warning}</div>
        )}
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        <div className="twr-board" style={{ ['--twr-cols' as string]: ROSTER_COLS }}>
          <div className="twr-board__sub">
            <span>Player</span>
            <span>Pos</span>
            <span className="twr-r">Cap</span>
            <span className="twr-r">Trade</span>
          </div>
          {players.map((p) => {
            const staged = inCart.has(p.mflID);
            return (
              <div key={p.mflID} className="twr-board__row">
                <span className="twr-c-name">{p.name}</span>
                <span className="twr-c-pos">{p.position}</span>
                <span className="twr-c-num twr-r" style={{ color: 'var(--amber-base)' }}>
                  {money(p.capSalary)}
                </span>
                <span className="twr-r">
                  <button
                    type="button"
                    className="twr-btn"
                    style={{ padding: '3px 8px', fontSize: 10 }}
                    disabled={staged}
                    onClick={() => onAdd(p)}
                  >
                    {staged ? 'Staged' : 'Add →'}
                  </button>
                </span>
              </div>
            );
          })}
        </div>
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
    <div style={{ border: '1px solid var(--hairline)', background: 'var(--surface-raised)', padding: 10 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <div
          style={{
            display: 'grid',
            height: 32,
            width: 32,
            flex: 'none',
            placeItems: 'center',
            background: 'var(--surface-overlay)',
            fontSize: 12,
            fontWeight: 700,
            fontFamily: 'var(--mono)',
            color: 'var(--text-secondary)',
          }}
        >
          {initials(leg.name)}
        </div>
        <div style={{ minWidth: 0, flex: 1 }}>
          <div style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: 13, fontWeight: 600 }}>
            {leg.name}
          </div>
          <div style={{ fontSize: 11, color: 'var(--text-secondary)' }}>
            {leg.position} · from {frName(leg.fromFranchiseID)}
          </div>
        </div>
        <button
          type="button"
          onClick={onRemove}
          aria-label="Remove leg"
          className="twr-iconbtn"
          style={{ flex: 'none' }}
        >
          ×
        </button>
      </div>
      <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>→</span>
        <select
          className="twr-select"
          style={{ minWidth: 0, flex: 1 }}
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
