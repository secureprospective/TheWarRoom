import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { array, integer, object, text } from '../data/parseValues';
import { parsePeople } from '../commissioner/people';
import { browserStorage, type SettingsStorage } from './franchise';

export const LEAGUE_KEY = 'thewarroom.target.selected-league';
export const MIRROR_LEAGUE_ID = 'mirror';

export type LeagueChoice = { id: string; name: string; teams: number; app: boolean };

export function parseLeagues(value: unknown): LeagueChoice[] {
  const leagues = array(value, 'Leagues', (entry, path) => {
    const row = object(entry, path, ['id', 'name', 'teams', 'app']);
    if (typeof row.app !== 'boolean') throw new Error(`${path}.app: expected boolean`);
    return {
      id: text(row.id, `${path}.id`),
      name: text(row.name, `${path}.name`),
      teams: integer(row.teams, `${path}.teams`),
      app: row.app,
    };
  });
  if (leagues.filter((league) => !league.app).length > 1)
    throw new Error('Leagues: only one reference league is expected');
  return leagues;
}

export function resolveLeagueId(leagues: LeagueChoice[], stored: string | null): string {
  return leagues.some((league) => league.id === stored) ? stored! : leagues[0]?.id ?? '';
}

export function chooseLeague(
  leagues: LeagueChoice[],
  id: string,
  storage?: SettingsStorage,
): string {
  if (!leagues.some((league) => league.id === id)) throw new Error('unknown league');
  storage?.setItem(LEAGUE_KEY, id);
  return id;
}

export async function readLeagues(): Promise<LeagueChoice[]> {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('The league list requires the desktop app.');
  const { targetLeagues } = await import('../data/api');
  return parseLeagues(await targetLeagues());
}

export type LeagueState = {
  leagues: LeagueChoice[];
  selectedId: string;
  selected?: LeagueChoice;
  loading: boolean;
  error?: string;
  select: (id: string) => void;
};

export function useLeagues(
  read: () => Promise<LeagueChoice[]> = readLeagues,
  storage: SettingsStorage | undefined = browserStorage(),
): LeagueState {
  const [leagues, setLeagues] = useState<LeagueChoice[]>([]);
  const [selectedId, setSelectedId] = useState<string>(
    () => storage?.getItem(LEAGUE_KEY) ?? MIRROR_LEAGUE_ID,
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  useEffect(() => {
    let active = true;
    setLoading(true);
    read().then(
      (choices) => {
        if (!active) return;
        setLeagues(choices);
        setSelectedId((current) => resolveLeagueId(choices, current));
        setLoading(false);
      },
      (cause) => {
        if (!active) return;
        setError(`League list failed: ${cause instanceof Error ? cause.message : String(cause)}`);
        setLoading(false);
      },
    );
    return () => {
      active = false;
    };
  }, [read]);
  const select = useCallback(
    (id: string) => {
      try {
        setSelectedId(chooseLeague(leagues, id, storage));
        setError(undefined);
      } catch (cause) {
        setError(`League selection failed: ${cause instanceof Error ? cause.message : String(cause)}`);
      }
    },
    [leagues, storage],
  );
  const selected = leagues.find((league) => league.id === selectedId);
  return useMemo(
    () => ({ leagues, selectedId, selected, loading, error, select }),
    [leagues, selectedId, selected, loading, error, select],
  );
}

export const LeagueContext = createContext<LeagueState | undefined>(undefined);

export function RailLeaguePicker() {
  const state = useContext(LeagueContext);
  if (!state) return null;
  return <div className="twr-app"><LeaguePicker leagues={state.leagues} selected={state.selected}
    onSelect={state.select} loading={state.loading} error={state.error} /></div>;
}

export async function readAppLeaguePeople(league: string) {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('This league requires the desktop app.');
  const { nativeBindings } = await import('../commissioner/api');
  const people = parsePeople(await nativeBindings().TargetCommissionerPeople(league));
  if (people.league !== league) throw new Error('People belong to another league');
  return people;
}

export async function appLeagueFranchise(league: string): Promise<string> {
  return (await readAppLeaguePeople(league)).yourFranchise;
}

export function leagueInitials(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  const initials = words.length > 1 ? words[0][0] + words[words.length - 1][0] : words[0]?.slice(0, 2);
  return (initials ?? '??').toUpperCase();
}

export function LeaguePicker({
  leagues,
  selected,
  onSelect,
  loading,
  error,
}: {
  leagues: LeagueChoice[];
  selected?: LeagueChoice;
  onSelect: (id: string) => void;
  loading?: boolean;
  error?: string;
}) {
  const [open, setOpen] = useState(false);
  const name = selected?.name ?? (loading ? 'Opening league…' : 'Choose a league');
  return (
    <div className="league">
      <div className="crest">
        <i>{leagueInitials(name)}</i>
        <div>
          <b>{name}</b>
          <small>
            {selected ? `${selected.teams} teams` : ''}
            {selected?.app ? ' · app' : selected ? ' · dynasty IDP' : ''}
          </small>
        </div>
      </div>
      <button type="button" className="pick" aria-expanded={open}
        onClick={() => setOpen(!open)}>Switch league</button>
      {open && <ul aria-label="Leagues">
        {leagues.map((league) => <li key={league.id}>
          <button type="button" onClick={() => { onSelect(league.id); setOpen(false); }}>
            {league.name} — {league.teams} teams
            <small>{league.app ? league.id : 'Synced from MFL'}</small>
          </button>
        </li>)}
      </ul>}
      {error && <p role="alert">{error}</p>}
    </div>
  );
}

export function LeagueChoices({
  leagues,
  selectedId,
  onSelect,
}: {
  leagues: LeagueChoice[];
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  return (
    <section className="league-choices" aria-labelledby="league-choices-title">
      <h4 id="league-choices-title">Leagues</h4>
      {leagues.length === 0 ? (
        <p role="status">No leagues available.</p>
      ) : (
        <ul>
          {leagues.map((league) => (
            <li key={league.id}>
              <button type="button" onClick={() => onSelect(league.id)}
                aria-current={selectedId === league.id ? 'true' : undefined}>
                {league.name} — {league.teams} teams
                <small>{league.app ? league.id : 'Synced from MFL'}</small>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
