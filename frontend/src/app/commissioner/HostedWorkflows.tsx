import { useEffect, useState } from 'react';
import { Act } from '../commands/Act';
import type { NativeRequest } from '../commands/native';
import { ActFields } from './ActFields';
import { actChange, type ActValues } from './actBody';
import { actTable, type ActDefinition, type Field } from './actTable';
import type { ActReadings } from './actReadings';
import { surfaceActions } from './surfaceActions';
import { newKey } from './forms';
import type { HostedRoom, HostedReading } from './hostedRoom';

export type HostedTask = 'bids' | 'waivers' | 'trades' | 'results' | 'rollover';
const field = (path: string, label: string, kind: Field['kind']): Field => ({
  path,
  label,
  kind,
});
const owner = (
  command: string,
  label: string,
  root: string,
  fields: Field[],
  privateReading = false,
): ActDefinition => ({
  control: '',
  command,
  label,
  root,
  fields,
  private: privateReading,
});
const ownerActs = [
  owner(
    'bid.place',
    'Place a private bid',
    'bid',
    [
      field('player', 'Free agent', 'player'),
      field('salaryCents', 'Annual salary (dollars)', 'money'),
      field('years', 'Years', 'number'),
    ],
    true,
  ),
  owner(
    'bid.raise',
    'Raise your bid',
    'bid',
    [
      field('bid', 'Your bid', 'bid'),
      field('salaryCents', 'New annual salary (dollars)', 'money'),
      field('years', 'Years', 'number'),
    ],
    true,
  ),
  owner('bid.withdraw', 'Withdraw your bid', 'bid', [field('bid', 'Your bid', 'bid')], true),
  owner('roster.cut', 'Release a player to waivers', 'waiver', [
    field('player', 'Your player', 'player'),
  ]),
  owner(
    'waivers.claim',
    'Claim a player',
    'waiver',
    [field('window', 'Waiver player', 'window')],
    true,
  ),
  ...['accept', 'decline', 'withdraw', 'appeal'].map((action) =>
    owner(`trade.${action}`, `${action[0].toUpperCase()}${action.slice(1)} a trade`, 'trade', [
      field('id', 'Trade', 'trade'),
    ]),
  ),
];
const commissionerCommands: Record<HostedTask, string[]> = {
  bids: ['bids.process', 'bids.award.override'],
  waivers: ['waivers.process', 'waivers.order.set', 'waivers.order.auto', 'waivers.award.override'],
  trades: ['trade.review', 'trade.recuse', 'trade.appeal.rule', 'trade.override'],
  results: [
    'commish.score.franchise',
    'commish.standings.record.adjust',
    'commish.standings.override.clear',
  ],
  rollover: ['season.rollover'],
};
export function HostedWorkflows({
  task,
  room,
  reading,
  franchise,
  commissioner,
}: {
  task: HostedTask;
  room: HostedRoom;
  reading: HostedReading;
  franchise: string;
  commissioner: boolean;
}) {
  const { markets: m, roster } = reading;
  const roomState = room.use();
  const [selected, setSelected] = useState('');
  const [values, setValues] = useState<ActValues>({});
  const [reason, setReason] = useState('');
  const [partner, setPartner] = useState('');
  const [assets, setAssets] = useState<string[]>([]);
  const disabled = !room.canChange();
  const definitions = [
    ...ownerActs.filter((a) =>
      task === 'bids'
        ? a.root === 'bid'
        : task === 'waivers'
          ? a.root === 'waiver'
          : task === 'trades'
            ? a.root === 'trade'
            : false,
    ),
    ...(commissioner ? actTable.filter((a) => commissionerCommands[task].includes(a.command)) : []),
  ];
  const definition = definitions.find((a) => a.command === selected);
  const own = roster.players.filter((p) => p.franchise === franchise);
  const marketCommand = [
    'bids.process',
    'bids.award.override',
    'waivers.process',
    'waivers.award.override',
    'trade.review',
    'trade.appeal.rule',
    'trade.override',
  ].includes(selected);
  const options: ActReadings['options'] = {
    franchise: roster.franchises,
    player: task === 'bids' ? roster.pool : own.map((p) => ({ value: p.id, label: p.name })),
    bid: m.bids.players.flatMap((p) =>
      p.own ? [{ value: p.own.id, label: `${p.name}: ${p.own.words}` }] : [],
    ),
    bidPlayer: m.bids.players.map((p) => ({ value: p.player, label: p.name })),
    window: m.waivers.players.map((p) => ({
      value: p.window,
      label: `${p.name}: ${p.words}`,
    })),
    claim: m.waivers.claims.map((c) => ({
      value: c.id,
      label: `${m.waivers.players.find((p) => p.player === c.player)?.name || 'Waiver player'}: ${c.words}`,
    })),
    trade: m.trades.offers.map((t, i) => ({
      value: t.id,
      label: `Trade ${i + 1}: ${t.words}`,
    })),
    ...(marketCommand ? roomState.market?.options : {}),
  };
  const revision =
    task === 'bids'
      ? m.bids.revision
      : task === 'waivers'
        ? m.waivers.revision
        : task === 'trades'
          ? m.trades.revision
          : task === 'results'
            ? m.standings.revision
            : roster.revision;
  const change =
    definition && (!marketCommand || roomState.market)
      ? actChange(definition, values, { league: reading.lineup.league, revision }, reason)
      : null;
  const review = (
    command: string,
    args: object,
    summary: string,
    visibility: 'league' | 'private' = 'league',
  ) => {
    void room.review(
      {
        command,
        mode: 'preview',
        arguments: {
          league: reading.lineup.league,
          command,
          args,
          reason: reason.trim(),
          expected: revision,
          request: newKey(),
          effective: 'now',
          visibility,
        },
      } as NativeRequest,
      summary,
    );
  };
  const teamName = (id: string) =>
    roster.franchises.find((f) => f.value === id)?.label || 'League team';
  const destination = (owner: string) => (owner === franchise ? teamName(partner) : 'Your team');
  const tradeAssets = [
    ...roster.players
      .filter((p) => p.franchise === franchise || p.franchise === partner)
      .map((p) => ({
        value: JSON.stringify({
          from: p.franchise,
          to: p.franchise === franchise ? partner : franchise,
          player: p.id,
        }),
        label: `${p.name}: ${teamName(p.franchise)} → ${destination(p.franchise)}`,
      })),
    ...roster.picks
      .filter((p) => p.owner === franchise || p.owner === partner)
      .map((p) => ({
        value: JSON.stringify({
          from: p.owner,
          to: p.owner === franchise ? partner : franchise,
          season: p.season,
          round: p.round,
          original: p.original,
        }),
        label: `${p.season} round ${p.round}, originally ${teamName(p.original)} → ${destination(p.owner)}`,
      })),
  ];
  useEffect(() =>
    surfaceActions.register('hosted-workflows', ({ value }) => {
      if (value === 'offer' && partner && assets.length)
        review(
          'trade.offer',
          {
            trade: {
              franchise,
              parties: [franchise, partner],
              assets: assets.map((v) => JSON.parse(v) as object),
            },
          },
          'Offer this trade',
        );
      if (value !== 'workflow' || !change || !definition || !reason.trim()) return;
      const args = change.args as Record<string, object>;
      review(
        definition.command,
        ownerActs.includes(definition)
          ? {
              ...args,
              [definition.root]: { ...args[definition.root], franchise },
            }
          : args,
        definition.label,
        definition.private ? 'private' : 'league',
      );
    }),
  );
  return (
    <section aria-label={`${task} workflow`}>
      {task === 'bids' && (
        <>
          <h3>Free-agent bids</h3>
          <p>Blind bids stay private until decided. Not even the commissioner can peek.</p>
          <p>Active bid limit: {m.bids.limit}</p>
          {m.bids.notes.map((n) => (
            <p key={n}>{n}</p>
          ))}
          {!m.bids.players.length && <p>No bids are visible to your franchise.</p>}
          {m.bids.players.map((p) => (
            <section key={p.player}>
              <h4>{p.name}</h4>
              <p>{p.words}</p>
              <p>{p.own?.words || 'You have no bid'}</p>
              {p.leader && <p>Leading bid: {p.leader.words}</p>}
              <p>{p.closes}</p>
            </section>
          ))}
          {m.bids.awards.map((a) => (
            <p key={a}>{a}</p>
          ))}
        </>
      )}
      {task === 'waivers' && (
        <>
          <h3>Waivers</h3>
          <p>
            {m.waivers.method} · {m.waivers.words}
          </p>
          {!m.waivers.players.length && (
            <p>No players are on waivers. Releasing a roster player opens a window.</p>
          )}
          {m.waivers.players.map((p) => (
            <p key={p.window}>
              {p.name}: {p.words}
            </p>
          ))}
          <h4>Your private claims</h4>
          {m.waivers.claims.map((c) => (
            <p key={c.id}>
              {m.waivers.players.find((p) => p.player === c.player)?.name || 'Waiver player'}:{' '}
              {c.words}
            </p>
          ))}
          <p>The server processes closed windows. A claim is not a roster award.</p>
        </>
      )}
      {task === 'trades' && (
        <>
          <h3>Trades</h3>
          {m.trades.notes.map((n) => (
            <p key={n}>{n}</p>
          ))}
          {!m.trades.offers.length && <p>No trades have been offered.</p>}
          {m.trades.offers.map((t, i) => (
            <section key={t.id}>
              <h4>Trade {i + 1}</h4>
              <p>{t.words}</p>
              <p>Expires {t.expires}</p>
              {[...t.details, ...t.reasons].map((d, i) => (
                <p key={i}>{d}</p>
              ))}
            </section>
          ))}
          <fieldset disabled={disabled}>
            <legend>Offer a trade</legend>
            <label>
              Other franchise
              <select
                value={partner}
                onChange={(e) => {
                  setPartner(e.target.value);
                  setAssets([]);
                  room.cancel();
                }}
              >
                <option value="">Choose a team</option>
                {roster.franchises
                  .filter((f) => f.value !== franchise)
                  .map((f) => (
                    <option key={f.value} value={f.value}>
                      {f.label}
                    </option>
                  ))}
              </select>
            </label>
            {partner &&
              tradeAssets.map((a) => (
                <label key={a.value}>
                  <input
                    type="checkbox"
                    checked={assets.includes(a.value)}
                    onChange={(e) => {
                      setAssets(
                        e.target.checked
                          ? [...assets, a.value]
                          : assets.filter((v) => v !== a.value),
                      );
                      room.cancel();
                    }}
                  />
                  {a.label}
                </label>
              ))}
            <Act
              variant="button"
              verb="commissioner.surface"
              args={{
                screen: 'hosted-workflows',
                action: 'check',
                value: 'offer',
              }}
              label="Check trade offer"
              disabled={disabled || !partner || !assets.length}
            >
              Check trade offer
            </Act>
          </fieldset>
        </>
      )}
      {task === 'results' && (
        <>
          <h3>Week {reading.lineup.week} results</h3>
          <p>{m.results.words}</p>
          {m.results.games.map((g, i) => (
            <section key={i}>
              <h4>{g.names}</h4>
              <p>{g.words}</p>
              {g.why.map((w) => (
                <p key={w}>{w}</p>
              ))}
            </section>
          ))}
          <h3>Computed standings</h3>
          <p>{m.standings.words}</p>
          <p>{m.standings.criteria.join(' · ')}</p>
          {m.standings.missing.map((w) => (
            <p key={w}>{w}</p>
          ))}
          <Standings rows={m.standings.computed} />
          {m.standings.override && (
            <>
              <h3>Commissioner standings</h3>
              <p>{m.standings.override.words}</p>
              <Standings rows={m.standings.override.rows} />
            </>
          )}
          <p>
            Choose a week in Set lineup, then refresh. Corrections keep the computed result
            alongside the ruling.
          </p>
        </>
      )}
      {task === 'rollover' && (
        <>
          <h3>Move to the next season</h3>
          <p>
            Current roster season: {roster.season}. Check the full plan before confirming.
            Contracts, pending decisions, picks and waiting rules move together in one recorded
            change.
          </p>
          <p>{reading.rules.rollover}</p>
          {!commissioner && <p>Only your commissioner can move the league into its next season.</p>}
        </>
      )}
      {definitions.length > 0 && (
        <fieldset disabled={disabled}>
          <legend>
            {commissioner ? 'Owner actions and commissioner rulings' : 'Your actions'}
          </legend>
          <label>
            Action
            <select
              value={selected}
              onChange={(e) => {
                setSelected(e.target.value);
                setValues({});
                room.cancel();
                if (
                  [
                    'bids.process',
                    'bids.award.override',
                    'waivers.process',
                    'waivers.award.override',
                    'trade.review',
                    'trade.appeal.rule',
                    'trade.override',
                  ].includes(e.target.value)
                )
                  void room.market(e.target.value);
              }}
            >
              <option value="">Choose an action</option>
              {definitions.map((a) => (
                <option key={a.command} value={a.command}>
                  {a.label}
                </option>
              ))}
            </select>
          </label>
          {definition && (
            <ActFields
              fields={definition.fields}
              values={values}
              held={{ options }}
              onChange={(key, value) => {
                setValues({ ...values, [key]: value });
                room.cancel();
              }}
            />
          )}
          <label>
            Reason for this change
            <textarea
              value={reason}
              onChange={(e) => {
                setReason(e.target.value);
                room.cancel();
              }}
            />
          </label>
          <Act
            variant="button"
            verb="commissioner.surface"
            args={{
              screen: 'hosted-workflows',
              action: 'check',
              value: 'workflow',
            }}
            label="Check this change"
            disabled={disabled || !change || !reason.trim()}
          >
            Check this change
          </Act>
        </fieldset>
      )}
    </section>
  );
}
function Standings({ rows }: { rows: HostedReading['markets']['standings']['computed'] }) {
  return (
    <table>
      <thead>
        <tr>
          <th>Place</th>
          <th>Team</th>
          <th>Record</th>
          <th>Points</th>
          <th>Why</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={r.name}>
            <td>{r.rank}</td>
            <td>{r.name}</td>
            <td>{r.record}</td>
            <td>{r.points}</td>
            <td>{r.why.join(' · ')}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
