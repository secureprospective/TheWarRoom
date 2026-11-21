import { useEffect, useMemo, useState } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { HQAct } from '../shell/HQAct';
import { onHostedTopics } from '../data/hostedReads';
import { createReadQuery } from '../data/query';
import { hostedCommandAvailable, type NativeRequest } from '../commands/native';
import { HostedCommissionerActTray } from './CommissionerActTray';
import { actTable, type ActDefinition } from './actTable';
import { hostedWorkflow } from './api';
import { parseHostedRoster, type HostedRoster } from './hostedReadings';
import type { HostedLeagueRoom, HostedPort } from './hostedRoom';

function createHostedPicksQuery(league: string, port: HostedPort = hostedWorkflow) {
  return createReadQuery('commissioner.TargetNativeAssets', {
    initial: (): { picks?: HostedRoster } => ({}),
    read: async (scope: string): Promise<HostedRoster> => {
      const response = await port({ command: 'native.assets.read', mode: 'read',
        arguments: { league: scope } } as NativeRequest);
      const picks = parseHostedRoster(response.data);
      if (picks.league !== league || scope !== league)
        throw new Error('Picks belong to another league.');
      return picks;
    },
    accept: (picks: HostedRoster) => ({ picks }),
  });
}

function nameOf(reading: HostedRoster, franchise: string): string {
  return reading.franchises.find(team => team.value === franchise)?.label ?? franchise;
}

const movePick = actTable.find((definition): definition is ActDefinition =>
  definition.command === 'correct.pick');

export function HostedControlRoomPicks({ league, room, port }: {
  league: string; room: HostedLeagueRoom; port?: HostedPort;
}) {
  const query = useMemo(() => createHostedPicksQuery(league, port), [league, port]);
  const state = query.use();
  const roomState = room.use();
  const [year, setYear] = useState('next');
  const [round, setRound] = useState('all');
  const [trade, setTrade] = useState('all');
  const [team, setTeam] = useState('');
  const [definition, setDefinition] = useState<ActDefinition>();

  useEffect(() => {
    const unsubscribe = onHostedTopics(topics => {
      if (!topics.some(topic => topic.league === league && topic.kind === 'league')) return;
      query.invalidate();
      void query.load([league], { overlap: 'after' });
    });
    void query.load([league]);
    return () => { unsubscribe(); query.dispose(); };
  }, [league, query]);

  const refresh = () => { query.invalidate(); void query.load([league]); };
  const reading = state.picks;
  const seasons = [...new Set(reading?.picks.map(pick => pick.season) ?? [])].sort((a, b) => a - b);
  const defaultYear = seasons.find(season => season > (reading?.season ?? 0)) ?? seasons.at(-1);
  const selectedYear = year === 'next' ? String(defaultYear ?? 'all') : year;
  const rounds = [...new Set(reading?.picks.map(pick => pick.round) ?? [])].sort((a, b) => a - b);
  const search = team.trim().toLowerCase();
  const rows = (reading?.picks ?? []).filter(pick => {
    if (selectedYear !== 'all' && pick.season !== Number(selectedYear)) return false;
    if (round !== 'all' && pick.round !== Number(round)) return false;
    if (trade === 'traded' && (pick.owner === pick.original || pick.correction || pick.compensatory))
      return false;
    if (search) {
      const names = `${nameOf(reading!, pick.original)} ${nameOf(reading!, pick.owner)} `
        + `${pick.original} ${pick.owner}`;
      if (!names.toLowerCase().includes(search)) return false;
    }
    return true;
  }).sort((a, b) => a.season - b.season || a.round - b.round || a.slot - b.slot);
  const unavailableActions = 'Grant, swap, condition, forfeit, void, restore and create are not on the '
    + 'server app yet. Existing draft commands operate on the rookie draft, not future pick ownership.';

  return <section className="cr-col hosted-picks" data-testid="hosted-control-picks"
    aria-label="Picks">
    <div className="cr-panel">
      <div className="cr-ph">PICKS <span className="sp" />
        <span className="ct">{reading
          ? `${rows.length} of ${reading.picks.length} picks` : 'Reading picks…'}</span>
        <HQAct disabled={state.pending} onAction={refresh}>Refresh picks</HQAct>
      </div>
      <div className="cr-pb hosted-pick-filters">
        <label>Year
          <select aria-label="Filter picks by year" value={selectedYear}
            onChange={event => setYear(event.target.value)}>
            <option value="all">All years</option>
            {seasons.map(season => <option key={season} value={season}>{season}</option>)}
          </select>
        </label>
        <label>Round
          <select aria-label="Filter picks by round" value={round}
            onChange={event => setRound(event.target.value)}>
            <option value="all">All rounds</option>
            {rounds.map(value => <option key={value} value={value}>Round {value}</option>)}
          </select>
        </label>
        <label>Trade status
          <select aria-label="Filter picks by trade status" value={trade}
            onChange={event => setTrade(event.target.value)}>
            <option value="all">All picks</option>
            <option value="traded">Traded</option>
            <option value="conditioned" disabled>Conditioned · not on server app yet</option>
          </select>
        </label>
        <label>Team search
          <input aria-label="Search picks by team" type="search" value={team}
            onChange={event => setTeam(event.target.value)} />
        </label>
      </div>
      {state.error && <FailureAlert
        message={state.error}
        prefix="Picks unavailable. "
        code={state.errorCode}
        className="cr-note"
        league={league}
        screen="Control Room · Picks"
        route="control/picks"
        port={port}
      />}
      {!reading && !state.error && <p role="status" className="cr-note">Reading the league's picks…</p>}
      {reading && <div className="ct-wrap">
        <table className="cr-t">
          <thead><tr><th>YEAR</th><th>ROUND</th><th>ORIGINAL</th><th>OWNED BY</th>
            <th>CONDITION</th><th>HOW</th><th>STATUS</th></tr></thead>
          <tbody>{rows.map(pick => {
            const how = pick.correction ? 'Correction' : pick.compensatory ? 'Compensatory'
              : pick.owner !== pick.original ? 'Traded' : 'Original';
            const status = pick.player ? 'USED' : pick.skipped ? 'SKIPPED' : 'ACTIVE';
            return <tr key={`${pick.season}:${pick.round}:${pick.original}:${pick.slot}`}>
              <td>{pick.season}</td><td>R{pick.round}</td><td>{nameOf(reading, pick.original)}</td>
              <td>{nameOf(reading, pick.owner)}</td><td>Not on the server app yet</td>
              <td>{how}</td><td>{status}</td>
            </tr>;
          })}</tbody>
        </table>
        {rows.length === 0 && <p className="cr-note">No picks match these filters.</p>}
      </div>}
    </div>
    <div className="cr-panel hosted-pick-actions">
      <div className="cr-ph">PICK ACTIONS</div>
      <div className="cr-pb">
        {movePick && hostedCommandAvailable(movePick.command) ? <HQAct
          disabled={!room.canChange() || Boolean(roomState.review) || roomState.pending}
          onAction={() => setDefinition(movePick)}>Correct a pick owner</HQAct>
          : <p role="status">Changing pick ownership is not on the server app yet.</p>}
        <p className="cr-note" role="status">{unavailableActions}</p>
      </div>
      {definition && <>
        <HQAct disabled={roomState.pending} onAction={() => { room.cancel(); setDefinition(undefined); }}>
          Close pick correction
        </HQAct>
        <HostedCommissionerActTray definition={definition} league={league} room={room} port={port} />
      </>}
    </div>
  </section>;
}
