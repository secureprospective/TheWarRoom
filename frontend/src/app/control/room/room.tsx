import { Act } from '../../commands/Act';
import { createContext, useContext, useEffect, type ReactNode } from 'react';
import type { leagueState } from '../../commissioner/proxy';
import type { RoomData } from './roomReads';

export const tabs = [
  ['jobs', 'JOBS'],
  ['sit', 'SITUATIONS'],
  ['rules', 'RULES'],
  ['season', 'SEASON DESK'],
  ['fix', 'FIX'],
  ['picks', 'PICKS'],
  ['people', 'PEOPLE'],
  ['league', 'LEAGUE'],
  ['server', 'SERVER'],
] as const;
export type Tab = (typeof tabs)[number][0];
export type Card = { title: string; sub: string; body: ReactNode; wide?: boolean };
export type LeagueReading = ReturnType<typeof leagueState.peek>;

/** What a branch may ask of the room. */
export type Room = {
  league: string;
  data: RoomData;
  state: LeagueReading;
  tab: Tab;
  go(tab: Tab, then?: () => void): void;
  inspect(node: ReactNode): void;
  card(card: Card | null): void;
  /** Open the command's existing form in the editor tray, or say it is not built. */
  run(command: string, initial?: Record<string, string>): void;
  /** Open the commissioner rule-change form on one rule. */
  rule(field: string): void;
  undo(entry: string): void;
  toast(words: string): void;
  closeTray(): void;
};
export const RoomContext = createContext<Room | null>(null);
export function useRoom(): Room {
  const room = useContext(RoomContext);
  if (!room) throw new Error('Control Room context missing');
  return room;
}
/** A branch's inspector when nothing in it is chosen. */
export function useDefaultInspector(node: ReactNode, deps: unknown[] = []) {
  const room = useRoom();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    room.inspect(node);
  }, deps);
}

/** The honest words for data the app does not hold yet. */
export const notYet = 'Not available yet';
export function NotYet({ what }: { what?: string }) {
  return (
    <span className="cr-na" title={what}>
      {notYet}
      {what ? ` · ${what}` : ''}
    </span>
  );
}

/** The mock's desk(): a panel with key values, a mini list and the command buttons. */
export function Desk({
  title,
  sub,
  kv,
  acts,
  children,
}: {
  title: string;
  sub: ReactNode;
  kv: [string, ReactNode][];
  acts: [string, string][];
  children?: ReactNode;
}) {
  const room = useRoom();
  return (
    <div className="cr-panel" aria-label={title}>
      <div className="cr-ph">
        {title}
        <span className="sp" />
        <span className="ct">{sub}</span>
      </div>
      <div className="cr-pb" style={{ display: 'flex', flexDirection: 'column', flex: 1 }}>
        <div className="cr-kv">
          {kv.map(([k, v]) => (
            <Pair key={k} k={k} v={v} />
          ))}
        </div>
        {children && <div className="cr-mini">{children}</div>}
        <div className="cr-btns">
          {acts.map(([label, command]) => (
            <Act
              key={label}
              className={`cr-btn${label.startsWith('RUN ROLL') ? ' g3' : ''}`}
              verb="commissioner.surface"
              variant="surface"
              label={label}
              args={{ screen: 'control-room', action: 'open', value: command }}
              attributes={{ 'data-act': String(command) }}
            >
              {label}
            </Act>
          ))}
        </div>
      </div>
    </div>
  );
}
function Pair({ k, v }: { k: string; v: ReactNode }) {
  return (
    <>
      <span>{k}</span>
      <b>{v}</b>
    </>
  );
}
export function Kv({ rows }: { rows: [string, ReactNode][] }) {
  return (
    <div className="cr-kv">
      {rows.map(([k, v]) => (
        <Pair key={k} k={k} v={v} />
      ))}
    </div>
  );
}
/** Month and day plus time, in the viewer's zone. */
export function when(iso: string | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1900) return '—';
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

/** A rule's value as the rules reading words it; settings congruence: every desk value comes from here. */
export function ruleValue(room: Room, field: string): string {
  const row = room.state.rules?.sections.flatMap((s) => s.rows).find((r) => r.field === field);
  return row ? row.value : '—';
}
