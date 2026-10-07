import { URGENCIES } from './contract';
import type { Alert, AlertReading, Unavailable } from './contract';
import { array, choice, object, text, timestamp } from './parseValues';
import { provenance } from './parse';
import { nodes, routeFor, type Node } from '../shell/nodes';

function alert(value: unknown, path: string): Alert {
  const r = object(value, path, [
    'kind', 'urgency', 'title', 'detail', 'node', 'workspace', 'at', 'provenance',
  ]);
  const node = choice(r.node, `${path}.node`, Object.keys(nodes) as Node[]);
  const workspace = text(r.workspace, `${path}.workspace`);
  if (!routeFor(node, workspace)) throw new Error(`${path}.workspace: unknown workspace for ${node}`);
  const at = timestamp(r.at, `${path}.at`);
  if (!at) throw new Error(`${path}.at: expected timestamp`);
  return {
    kind: text(r.kind, `${path}.kind`),
    urgency: choice(r.urgency, `${path}.urgency`, URGENCIES),
    title: text(r.title, `${path}.title`),
    detail: text(r.detail, `${path}.detail`),
    node,
    workspace,
    at,
    provenance: provenance(r.provenance, `${path}.provenance`),
  };
}
function unavailable(value: unknown, path: string): Unavailable {
  const r = object(value, path, ['kind', 'note']);
  return { kind: text(r.kind, `${path}.kind`), note: text(r.note, `${path}.note`) };
}
export function parseAlerts(value: unknown): AlertReading {
  const r = object(value, 'alerts', ['alerts', 'unavailable']);
  return {
    alerts: array(r.alerts, 'alerts.alerts', alert),
    unavailable: array(r.unavailable, 'alerts.unavailable', unavailable),
  };
}
