import { useInspectorStore } from '../../store/inspector';
import { main } from '../../../wailsjs/go/models';

// InspectorContent is the B-4a Contextual Inspector body — the per-player score anatomy the shell's
// <Inspector> renders as `inspector={…}`. Session-B anatomy: a score-dominant hero, the six layer
// bars that build it, and a terminal contract/cap block. Data comes from the inspector store
// (GetPlayerScore → the persisted breakdown); this component is pure presentation over that slice.
//
// Honesty rulings baked in:
//   - No hero HUE banding: the Session-C ≥90/80/70/55 bands assume a normalized 0-100 score; the
//     engine's AdjustedScore is a raw magnitude (BasePoints × multipliers), so a threshold on it
//     would mis-signal. The hero renders achromatic (restraint doctrine); quality shows through the
//     per-layer boost/penalty bars and the cap tier, which ARE defined scales.
//   - Multiplier bars are centered at 1.0 — fill RIGHT (green) for a boost (>1), LEFT (amber) for a
//     penalty (<1). That >1/<1 semantic is real; it never invents a scale.
//   - FilmRaw is never shown (DEBUG-only engine field); Film uses FilmEffective.

export function InspectorContent() {
  const player = useInspectorStore((s) => s.player);
  const loading = useInspectorStore((s) => s.loading);
  const error = useInspectorStore((s) => s.error);
  const notFound = useInspectorStore((s) => s.notFound);
  const label = useInspectorStore((s) => s.label);
  const warning = useInspectorStore((s) => s.warning);

  if (loading) return <Centered text="Loading breakdown…" />;
  if (error) return <Banner kind="warn" text={error} />;
  if (notFound)
    return <Centered text="No score for this player under the active config. Re-score the league?" />;
  if (!player) return <Centered text="Select a player" />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Hero player={player} />
      {warning && <Banner kind="caution" text={warning} />}

      <Section title="Score layers">
        <NumberLine label="Base (L2 proxy)" value={player.basePoints.toFixed(1)} sub="fantasy pts" />
        <MultBar label="Age (L3)" mult={player.agePull} />
        <MultBar label="Film" mult={player.filmEffective} />
        <MultBar label="Athleticism (RAS)" mult={player.rasEffective} />
        <MultBar label="Breakout" mult={player.breakoutEffective} />
        <MultBar label="Cap" mult={player.capMultiplier} />
      </Section>

      <Section title="Composite">
        <NumberLine label="L4 scouting (combined)" value={fmtMult(player.l4Combined)} />
        <NumberLine label="Scouting-adjusted" value={player.scoutingAdjusted.toFixed(1)} />
        <NumberLine label="Adjusted score" value={player.adjustedScore.toFixed(1)} strong />
      </Section>

      <Section title="Contract & cap">
        <NumberLine label="Salary" value={player.salary > 0 ? `$${player.salary.toFixed(1)}M` : '—'} />
        <ContractRow player={player} />
        <NumberLine
          label="Cap efficiency"
          value={player.capEffOK ? `${player.capEff.toFixed(2)} / $M` : '—'}
        />
      </Section>

      {label && (
        <div
          style={{
            fontFamily: 'var(--mono)',
            fontSize: 10,
            color: 'var(--text-tertiary)',
            borderTop: '1px solid var(--hairline)',
            paddingTop: 8,
          }}
        >
          {label}
        </div>
      )}
    </div>
  );
}

// Hero — the score-dominant header: position badge · name · franchise, with the AdjustedScore as the
// oversized numeric anchor (achromatic per the honesty ruling above).
function Hero({ player }: { player: main.PlayerScoreDTO }) {
  return (
    <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12 }}>
      <div style={{ minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          {player.position && (
            <span
              className="twr-chip"
              style={{ fontSize: 10, padding: '1px 6px', textTransform: 'none' }}
            >
              {player.position}
            </span>
          )}
          <span style={{ fontSize: 10, fontFamily: 'var(--mono)', color: 'var(--text-tertiary)' }}>
            {/* '—' not 'Free agent': an empty franchise can also mean the player isn't in the roster
                store yet (sync lag) — don't assert free-agency we can't confirm (GLM L5). */}
            {player.franchiseID || '—'}
          </span>
        </div>
        <div style={{ marginTop: 4, fontSize: 16, fontWeight: 700, color: 'var(--text-primary)' }}>
          {player.name}
        </div>
      </div>
      <div style={{ textAlign: 'right', flexShrink: 0 }}>
        <div
          style={{
            fontFamily: 'var(--mono)',
            fontVariantNumeric: 'tabular-nums',
            fontSize: 26,
            fontWeight: 700,
            lineHeight: 1,
            color: 'var(--text-primary)',
          }}
        >
          {player.adjustedScore.toFixed(1)}
        </div>
        <div style={{ fontSize: 9.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--text-tertiary)' }}>
          Adjusted
        </div>
      </div>
    </div>
  );
}

// MultBar renders a multiplier layer as a bar centered at 1.0: a boost (>1) fills right in green, a
// penalty (<1) fills left in amber. Deviation is clamped to ±0.5 for the visual half-width; the exact
// factor is always printed, so the bar is a glance and the number is the truth.
function MultBar({ label, mult }: { label: string; mult: number }) {
  const finite = Number.isFinite(mult);
  const dev = finite ? Math.max(-0.5, Math.min(0.5, mult - 1)) : 0; // clamp for the bar geometry only
  const pct = (Math.abs(dev) / 0.5) * 50; // 0..50% of the half-track
  const boost = mult > 1; // exactly 1.0 is neutral (aligns with the cap-tier dot rule)
  // >1 boost green · <1 penalty amber · exactly 1.0 (or non-finite) neutral — never contradict the
  // cap-tier dot for the Cap row (GLM L4/L6).
  const color = !finite || mult === 1 ? 'var(--text-disabled)' : boost ? 'var(--green-base)' : 'var(--amber-base)';
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '104px 1fr 48px', alignItems: 'center', gap: 8 }}>
      <span style={{ fontSize: 11.5, color: 'var(--text-secondary)' }}>{label}</span>
      <div
        style={{
          position: 'relative',
          height: 6,
          background: 'var(--surface-sunken)',
          border: '1px solid var(--hairline)',
        }}
      >
        {/* center baseline (1.0) */}
        <span aria-hidden style={{ position: 'absolute', left: '50%', top: 0, bottom: 0, width: 1, background: 'var(--text-disabled)' }} />
        <span
          aria-hidden
          style={{
            position: 'absolute',
            top: 0,
            bottom: 0,
            background: color,
            width: `${pct}%`,
            ...(boost ? { left: '50%' } : { right: '50%' }),
          }}
        />
      </div>
      <span
        style={{
          textAlign: 'right',
          fontFamily: 'var(--mono)',
          fontVariantNumeric: 'tabular-nums',
          fontSize: 11.5,
          color: 'var(--text-primary)',
        }}
      >
        {finite ? fmtMult(mult) : '—'}
      </span>
    </div>
  );
}

// ContractRow shows the cap tier with a boost/penalty dot driven by the cap MULTIPLIER (the honest
// direction), plus the veteran flag — never guessing Hot/Cold polarity.
function ContractRow({ player }: { player: main.PlayerScoreDTO }) {
  const boost = player.capMultiplier >= 1;
  const dotColor = player.capMultiplier === 1 ? 'var(--text-disabled)' : boost ? 'var(--green-base)' : 'var(--amber-base)';
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
      <span style={{ fontSize: 11.5, color: 'var(--text-secondary)' }}>Cap tier</span>
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
        {player.isVeteran && (
          <span style={{ fontSize: 9.5, fontFamily: 'var(--mono)', color: 'var(--text-tertiary)', textTransform: 'uppercase' }}>
            vet
          </span>
        )}
        <span aria-hidden style={{ width: 7, height: 7, borderRadius: '50%', background: dotColor }} />
        <span style={{ fontSize: 11.5, color: 'var(--text-primary)' }}>{player.capTier || '—'}</span>
      </span>
    </div>
  );
}

// fmtMult prints a multiplier as ×N.NN — the compact form the bars/composite share.
function fmtMult(m: number): string {
  return `×${m.toFixed(2)}`;
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div
        style={{
          fontSize: 10,
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.08em',
          color: 'var(--text-tertiary)',
        }}
      >
        {title}
      </div>
      {children}
    </div>
  );
}

function NumberLine({ label, value, sub, strong }: { label: string; value: string; sub?: string; strong?: boolean }) {
  return (
    <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: 8 }}>
      <span style={{ fontSize: 11.5, color: 'var(--text-secondary)' }}>{label}</span>
      <span style={{ display: 'inline-flex', alignItems: 'baseline', gap: 5 }}>
        {sub && <span style={{ fontSize: 10, color: 'var(--text-tertiary)' }}>{sub}</span>}
        <span
          style={{
            fontFamily: 'var(--mono)',
            fontVariantNumeric: 'tabular-nums',
            fontSize: strong ? 13.5 : 12,
            fontWeight: strong ? 700 : 500,
            color: 'var(--text-primary)',
          }}
        >
          {value}
        </span>
      </span>
    </div>
  );
}

function Centered({ text }: { text: string }) {
  return (
    <div
      style={{
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        textAlign: 'center',
        color: 'var(--text-tertiary)',
        fontFamily: 'var(--mono)',
        fontSize: 12,
      }}
    >
      {text}
    </div>
  );
}

function Banner({ kind, text }: { kind: 'warn' | 'caution'; text: string }) {
  return (
    <div className={`twr-banner twr-banner--${kind}`} style={{ display: 'block', fontWeight: 500 }}>
      {text}
    </div>
  );
}
