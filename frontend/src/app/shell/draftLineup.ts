import { useEffect, useRef, useState } from 'react';
import {
  POSITIONS, type LineupCheck, type LineupReading, type Snapshot, type Position,
} from '../data/contract';
import type { LineupDraft } from './state';

function positionOrder(position?: Position): number {
  const index = position === undefined ? -1 : POSITIONS.indexOf(position);
  return index < 0 ? POSITIONS.length : index;
}
export function draftReading(
  snapshot: Snapshot, saved: LineupReading, draft: LineupDraft, check?: LineupCheck,
): LineupReading {
  const directory = new Map(snapshot.players.value.map((p) => [p.id, p]));
  const active = snapshot.rosters.value.find((r) => r.franchiseId === draft.franchiseId)?.players
    .filter((p) => p.rosterStatus === 'ROSTER').map((p) => p.id) ?? [];
  const players = (ids: string[]) => ids.map((id) => ({ id, position: directory.get(id)?.position })).sort(
    (a, b) => positionOrder(a.position) - positionOrder(b.position) ||
      (directory.get(a.id)?.name ?? a.id).localeCompare(directory.get(b.id)?.name ?? b.id) ||
      a.id.localeCompare(b.id),
  );
  return { ...saved, starters: players(draft.starters),
    bench: players(active.filter((id) => !draft.starters.includes(id))), check: check ?? saved.check };
}
export function useDraftCheck(
  read: (franchiseId: string, starters: string[]) => Promise<LineupCheck>,
  draft: LineupDraft | null,
  saved?: LineupCheck,
) {
  const counter = useRef(0);
  const [result, setResult] = useState<{ check?: LineupCheck; error?: string }>({});
  useEffect(() => {
    const request = ++counter.current;
    if (!draft) {
      setResult({});
      return;
    }
    void read(draft.franchiseId, draft.starters).then((check) => {
      if (counter.current === request) setResult({ check });
    }).catch((cause) => {
      if (counter.current === request) setResult((previous) => ({ ...previous, error: String(cause) }));
    });
    return () => { ++counter.current; };
  }, [read, draft]);
  return { ...result, check: result.check ?? saved };
}
