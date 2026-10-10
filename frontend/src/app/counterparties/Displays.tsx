import type { Snapshot } from '../data/contract';
import { Act } from '../commands/Act';
import type { TeamEdge, TeamShare } from './contract';

export const number = (v: number | null | undefined, digits = 2) =>
  v == null ? '—' : v.toFixed(digits);
export const percent = (v: number | null) => (v == null ? '—' : `${(v * 100).toFixed(0)}%`);
export function teamIdentity(snapshot: Snapshot | undefined, fid: string, fallback: string) {
  const identity = snapshot?.franchises.value.find((f) => f.id === fid);
  return { name: identity?.name ?? fallback, matched: !!identity };
}
export function TeamLink({ snapshot, fid, name }: { snapshot?: Snapshot; fid: string; name: string }) {
  const identity = teamIdentity(snapshot, fid, name);
  return (
    <span className="counter-team">
      <Act
        verb="market"
        args={{ field: 'inspect', value: `team:${fid}` }}
        label={`Inspect ${identity.name}`}
      >
        {identity.name}
      </Act>
      {!identity.matched && <small>Archive identity · not matched to current league</small>}
    </span>
  );
}
export function EdgeBar({ edge, label }: { edge: TeamEdge; label: string }) {
  if (edge.mean == null) return <span>—</span>;
  if (edge.lo == null || edge.hi == null) return <span>{number(edge.mean)} · too few for a range</span>;
  const span = Math.max(Math.abs(edge.lo), Math.abs(edge.hi), 0.01);
  const x = (v: number) => 150 + (v / span) * 135;
  const description = `${label}: ${number(edge.mean)}, likely ${number(edge.lo)} to ${number(edge.hi)}`;
  return (
    <div className="counter-edge" data-tone={edge.tone}>
      <span>
        {number(edge.mean)} · likely {number(edge.lo)} to {number(edge.hi)}
      </span>
      <svg
        className="market-range"
        viewBox="0 0 300 26"
        role="img"
        aria-label={`${description}; zero marked`}
      >
        <line className="axis" x1="12" x2="288" y1="13" y2="13" />
        <line className="reference" x1="150" x2="150" y1="2" y2="24" />
        <line className="range" x1={x(edge.lo)} x2={x(edge.hi)} y1="13" y2="13" />
        <circle cx={x(edge.mean)} cy="13" r="4" />
      </svg>
    </div>
  );
}
export function Share({ share, label }: { share: TeamShare; label: string }) {
  return <span data-tone={share.tone} title={`${label}: ${share.k} of ${share.n}`}>{percent(share.p)}</span>;
}
