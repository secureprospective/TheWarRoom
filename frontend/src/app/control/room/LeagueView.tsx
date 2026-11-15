import { useRoomSurface } from './useRoomSurface';
import { Act } from '../../commands/Act';
import { useEffect, useState } from 'react';
import { NoteForm } from '../../commissioner/NoteForm';
import '../../commissioner/commissioner.css';
import { readLedgerFiltered, type Ledger } from '../../commissioner/proxy';
import { Desk, NotYet, useDefaultInspector, useRoom, when } from './room';

/** The ledger chips. Go filters by kind (TargetCommissionerLedgerFiltered); a chip with no Go kind is
 *  shown disabled, never faked with a word match. */
const CHIPS: [string, string | null][] = [
  ['ALL', ''],
  ['RULES', 'rules'],
  ['FIXES', 'corrections'],
  ['TRADES', null],
  ['PEOPLE', 'people'],
  ['VIA AI', null],
  ['SYSTEM', null],
];

export function LeagueView({
  readFiltered = readLedgerFiltered,
}: {
  readFiltered?: typeof readLedgerFiltered;
}) {
  const room = useRoom();
  const { people, ledger } = room.state;
  const { data } = room;
  const [kind, setKind] = useState<'notice' | 'newsletter' | 'note'>('notice');
  const [chip, setChip] = useState('');
  const [filtered, setFiltered] = useState<Ledger>();
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    setFiltered(undefined);
    setError('');
    if (!chip) return;
    readFiltered(room.league, { person: '', kind: chip, from: '', before: '' }).then(
      (r) => {
        if (active) setFiltered(r);
      },
      (e) => {
        if (active) setError(String(e));
      },
    );
    return () => {
      active = false;
    };
  }, [chip, room.league, readFiltered, ledger?.revision]);
  useRoomSurface('room-league', ({ action, value }) => {
    if (action === 'group') setChip(value);
    else if (value === 'notice' || value === 'newsletter' || value === 'note') setKind(value);
  });
  const shown = (chip ? filtered : ledger)?.entries.slice(0, 22) ?? [];
  useDefaultInspector(
    <>
      <h3>League</h3>
      <div className="sub">
        Talking to the league, the record of every change, the data feeds, and the leagues on this server.
      </div>
    </>,
  );
  return (
    <>
      <div className="cr-col" style={{ width: 440, flex: 'none' }}>
        <div className="cr-panel">
          <div className="cr-ph">TELL THE LEAGUE</div>
          <div
            className="cr-pb"
            style={{ display: 'flex', flexDirection: 'column', gap: 8, overflow: 'auto' }}
          >
            <div className="cr-seg">
              {(['notice', 'newsletter', 'note'] as const).map((k) => (
                <Act
                  key={k}
                  active={kind === k}
                  verb="commissioner.surface"
                  variant="surface"
                  label={k === 'note' ? 'LEAGUE NOTE' : k.toUpperCase()}
                  args={{ screen: 'room-league', action: 'view', value: k }}
                >
                  {k === 'note' ? 'LEAGUE NOTE' : k.toUpperCase()}
                </Act>
              ))}
            </div>
            {kind === 'newsletter' ? (
              <NotYet what="the newsletter is not built (CM-052)" />
            ) : people ? (
              <NoteForm key={`${people.league}:${kind}`} people={people} kind={kind} />
            ) : (
              <p role="status">Reading the people…</p>
            )}
          </div>
        </div>
        <Desk
          title="DATA & ENGINE"
          sub="feed health lives in Server"
          kv={[
            ['Last refresh', <NotYet key="r" />],
            ['Engine parameters', paramWords(room.data)],
          ]}
          acts={[
            ['REFRESH NOW', 'data.refresh'],
            ['CLEAR A CACHE', 'cache.clear'],
            ['ENGINE (C ONLY)', 'engine.set'],
          ]}
        />
        <Desk
          title="LEAGUES"
          sub={data.directory?.name ?? room.state.rules?.name ?? '—'}
          kv={[
            ['Source', data.directory?.source || '—'],
            ['Season', String(data.directory?.rosterSeason ?? '—')],
          ]}
          acts={[
            ['NEW LEAGUE (PRESET)', 'commish.league.create'],
            ['MIGRATE AN MFL LEAGUE ›', 'league.migrate.prepare'],
            ['SEED A HELD LEAGUE', 'commish.league.seed_held'],
          ]}
        />
        <Desk
          title="RULEBOOK"
          sub="what owners read"
          kv={[
            ['Official rulebook', data.houseRules?.source || '—'],
            ['House rules', `${data.houseRules?.rules.length ?? 0} entries`],
          ]}
          acts={[
            ['ADD A HOUSE RULE', 'house.texts'],
            ['REMOVE ONE', 'house.texts'],
          ]}
        />
      </div>
      <div className="cr-col" style={{ flex: 1 }}>
        <div className="cr-panel" style={{ flex: 1 }}>
          <div className="cr-ph">
            LEDGER · every change, who, why
            <span className="sp" />
            <span className="ct">owners see this, minus role grants and invites</span>
          </div>
          <div className="cr-pb cr-led">
            <div className="cr-chips" style={{ display: 'flex', border: 0, padding: '0 0 8px' }}>
              {CHIPS.map(([label, k]) => (
                <Act
                  key={label}
                  className="cr-chip"
                  active={k === chip}
                  disabled={k === null}
                  style={k === null ? { opacity: 0.4 } : undefined}
                  verb="commissioner.surface"
                  variant="surface"
                  label={label}
                  args={{ screen: 'room-league', action: 'group', value: k ?? '' }}
                >
                  {label}
                </Act>
              ))}
            </div>
            {error && <div className="cr-note">{error}</div>}
            {shown.map((l) => (
              <div key={l.id}>
                <span className="t">{when(l.when)}</span>
                <span className="a">{l.who}</span>
                <span>
                  {l.what}
                  {l.why ? ` · ${l.why}` : ''}
                </span>
              </div>
            ))}
            {!ledger && <div className="cr-note">{room.state.error || 'Reading the ledger…'}</div>}
          </div>
        </div>
      </div>
    </>
  );
}

/** The mock's summary of the engine parameters (GetParams): how many there are and how many are changed. */
export function paramWords(data: ReturnType<typeof useRoom>['data']): string {
  if (!data.params) return data.errors.params ?? '—';
  const all = data.params.params ?? [];
  if (!all.length) return 'none defined';
  const changed = all.filter((p) => p.overridden).length;
  return `${all.length} defined${changed ? ` · ${changed} changed` : ''}`;
}
