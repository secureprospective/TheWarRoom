import type { ReactNode } from 'react';
import { HQAct } from './HQAct';
import { nodes, nodeKeys, type Node } from './nodes';

type Leaf = { slug: string; label: string };
export type RailLeaves = Partial<Record<Node, readonly Leaf[]>>;

/** Shared workspace navigation for local and hosted rails; routing stays with each shell. */
export function RailNodes({ node, workspace, open, leaves, order = nodeKeys, beforeLeaves, afterLeaves,
  mark, container = true }: {
  node: Node; workspace: string; open: (node: Node, workspace?: string) => void;
  leaves?: RailLeaves; order?: readonly Node[];
  beforeLeaves?: (node: Node) => ReactNode; afterLeaves?: (node: Node) => ReactNode;
  mark?: (node: Node) => ReactNode; container?: boolean;
}) {
  const content = <>
    <div className="gl">LEAGUE</div>
    {order.map((key, index) => {
      const active = node === key;
      return <div className={`node${active ? ' open' : ''}`} key={key}>
        <HQAct className="nb" label={nodes[key].label} active={active} onAction={() => open(key)}>
          <span className="k">{index + 1}</span>{nodes[key].label}{mark?.(key)}<span className="car">›</span>
        </HQAct>
        {active && beforeLeaves?.(key)}
        {active && (leaves?.[key] ?? []).map((leaf, i, all) => <HQAct key={leaf.slug}
          label={leaf.label} className={`leaf${workspace === leaf.slug ? ' sel' : ''}`}
          onAction={() => open(key, leaf.slug)}>
          <span className="tr">{i === all.length - 1 ? '└' : '├'}</span>{leaf.label}
        </HQAct>)}
        {active && afterLeaves?.(key)}
      </div>;
    })}
  </>;
  return container ? <nav className="rail-body" aria-label="Workspaces">{content}</nav> : content;
}
