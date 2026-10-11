import { useWorkAnchor } from '../shell/workScroll';
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { localDate } from '../clock/labels';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { FranchisePicker } from '../shell/FranchisePicker';
import type { Snapshot } from '../data/contract';
import type { Scope, Readings, Reading, PlayerContract, ContractsProvider, Year, Row } from './contract';
import { contractsProvider } from './provider';
import './contracts.css';
import { useContractsFilters } from './state';
import { whatIfProvider, type WhatIf, type WhatIfProvider } from './whatif';
import ReleasePlans from './ReleasePlans';
import ContractActs, { ContractMovePanel } from './ContractActs';
import { WhatIfPanel, WhatIfActs } from './WhatIfPanel';

type Result<T> = { reading?: T; error?: string };
export function useContractRead<T>(
  key: string,
  read: () => Promise<T>,
  provider: Pick<ContractsProvider, 'onChange'>,
  retain = false,
): Result<T> {
  const [result, setResult] = useState<Result<T> & { key: string }>({ key });
  useEffect(() => {
    let sequence = 0;
    let active = true;
    const refresh = () => {
      const request = ++sequence;
      setResult(previous => retain && previous.key === key && previous.reading ? previous : { key });
      void read().then(
        (reading) => {
          if (active && request === sequence) setResult({ key, reading });
        },
        (cause) => {
          if (active && request === sequence) setResult({ key, error: String(cause) });
        },
      );
    };
    refresh();
    const unsubscribe = provider.onChange(refresh);
    return () => {
      active = false;
      sequence += 1;
      unsubscribe();
    };
    // key describes every argument; callers memoize the provider.
  }, [key, provider]);
  return result.key === key ? { reading: result.reading, error: result.error } : {};
}

export function ContractReading<T>({
  title,
  endpoint,
  reading,
  error,
  children,
  metadata = true,
}: {
  title: string;
  endpoint?: string;
  reading?: Reading<T>;
  error?: string;
  children?: ReactNode;
  metadata?: boolean;
}) {
  return (
    <section className="contract-reading" aria-label={title} data-endpoint={endpoint}>
      <h4>{title}</h4>
      {error ? (
        <p role="alert">
          Failed · {error}. Retry by reopening this workspace or refreshing from MFL.
        </p>
      ) : !reading ? (
        <p role="status">Reading the MFL mirror… No figures yet.</p>
      ) : reading.status !== 'ok' ? (
        <p role={reading.status === 'failed' ? 'alert' : 'status'}>
          {reading.status === 'not_wired'
            ? 'Not wired'
            : reading.status === 'thin'
              ? 'Too few to judge'
              : 'Failed'}{' '}
          · {reading.absenceReason}
        </p>
      ) : (
        children
      )}
      {reading && metadata && (
        <>
          <p className="contract-source">
            Source: {reading.provenance.source} · As of:{' '}
            {reading.provenance.freshness.fetchedAt
              ? localDate(reading.provenance.freshness.fetchedAt) : 'Not recorded'}
          </p>
          <details>
            <summary>How this is counted</summary>
            <p>{reading.howCounted}</p>
            <p>{reading.provenance.freshness.note}</p>
            <p>
              Method: {reading.method} · Build: {reading.buildStamp}
            </p>
          </details>
        </>
      )}
    </section>
  );
}
function Years({ years }: { years: Year[] }) {
  return (
    <>
      <table className="contract-years" aria-label="Year commitments">
        <thead>
          <tr><th>Year</th><th>Amount</th><th>Tag</th></tr>
        </thead>
        <tbody>
          {years.map((year) => (
            <tr key={year.season}>
              <th scope="row">{year.season}</th>
              <td className="contract-money">{year.cost.text}</td>
              <td>{year.tag}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="contract-footnote">
        {[...new Set(years.map((year) => year.reason))].join(' · ')}
      </p>
    </>
  );
}
function ContractGroup({ title, rows, mine, whatif }: {
  title: string; rows: Row[]; mine: boolean; whatif?: WhatIf;
}) {
  return (
    <section className="contract-group" aria-label={`${title} (${rows.length})`}>
      <h5>{title} ({rows.length})</h5>
      <table className="contract-table" data-team-column={!mine}>
        <thead>
          <tr>
            <th>Player</th><th>Annual salary</th><th>Expires / status</th>
            {!mine && <th>Team</th>}
            {mine && <th className="contract-whatif-column">What-if</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id} data-mine={mine}>
              <td>
                <Act
                  verb="inspector.open"
                  args={{ subject: { kind: 'player', id: row.id, franchiseId: row.franchiseId } }}
                  label={`Inspect ${row.name} contract`}
                >
                  {row.name}
                </Act>{' '}
                <small className="contract-position">{row.position || 'Not held'}</small>

              </td>
              <td className="contract-money">{row.salary.text}</td>
              <td>
                {row.expiration || 'Not held'}{row.status ? ` · ${row.status}` : ''}
              </td>
              {!mine && <td className="contract-team" title={row.team}>{row.team}</td>}
              {mine && <td className="contract-whatif-column">
                <WhatIfActs mine={row.franchiseId} player={row.id} value={whatif} menu />
              </td>}
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 && <p>No contracts match in this group.</p>}
    </section>
  );
}
export function ContractPanel({
  id,
  provider: supplied,
  mine: suppliedMine,
  whatIf: suppliedWhatIf,
}: {
  id: string;
  mine?: string;
  whatIf?: WhatIfProvider;
  provider?: ContractsProvider;
}) {
  const shell = commands.use();
  const mine = suppliedMine ?? shell.franchiseId ?? '';
  const provider = useMemo(() => supplied ?? contractsProvider(), [supplied]);
  const result = useContractRead<Reading<PlayerContract>>(id, () => provider.player(id), provider);
  const simulation = useMemo(() => suppliedWhatIf ?? whatIfProvider(), [suppliedWhatIf]);
  const whatif = useContractRead<Reading<WhatIf>>(mine, () => simulation.query(mine), simulation, true);
  return (
    <div className="contracts">
      <ContractReading title="Contract years" endpoint="C-06.5" {...result}>
        {result.reading?.value && (
          <>
            <p>
              {result.reading.value.player.name} ·{' '}
              {result.reading.value.player.status || 'Contract status not supplied'}
            </p>
            {result.reading.value.player.info && (
              <p>MFL contract info: {result.reading.value.player.info}</p>
            )}
            <Years years={result.reading.value.years} />
            {mine === result.reading.value.player.franchiseId &&
              <WhatIfActs mine={mine} player={id} value={whatif.reading?.value} />}
          </>
        )}
      </ContractReading>
    </div>
  );
}
export default function Contracts({
  snapshot,
  provider: supplied,
  whatIf: suppliedWhatIf,
}: {
  snapshot: Snapshot;
  provider?: ContractsProvider;
  whatIf?: WhatIfProvider;
}) {
  const s = commands.use();
  const provider = useMemo(() => supplied ?? contractsProvider(), [supplied]);
  const simulation = useMemo(() => suppliedWhatIf ?? whatIfProvider(), [suppliedWhatIf]);
  const whatif = useContractRead<Reading<WhatIf>>(
    s.franchiseId ?? '', () => simulation.query(s.franchiseId ?? ''), simulation, true,
  );
  const root = useRef<HTMLDivElement>(null);
  useWorkAnchor(root, '.contract-table tbody tr');
  const filters = useContractsFilters();
  const scope = { ...filters, mine: s.franchiseId ?? '' };
  const key = JSON.stringify(scope);
  const { reading: readings, error } = useContractRead<Readings>(
    key,
    () => provider.contracts(scope),
    provider,
  );
  const [options, setOptions] = useState<{
    teams: { id: string; name: string }[];
    positions: string[];
  }>({ teams: [], positions: [] });
  useEffect(() => {
    if (readings?.table.value) setOptions(readings.table.value);
  }, [readings]);
  if (!s.franchiseId) return <FranchisePicker snapshot={snapshot} />;
  const cap = readings?.cap.value;
  const table = readings?.table.value;
  return (
    <div ref={root} className="contracts">
      <WhatIfPanel mine={s.franchiseId} {...whatif} />
      <ContractMovePanel mine={s.franchiseId} />
      {whatif.reading?.value?.moves.length ? <p className="whatif-status" role="status">
        {whatif.reading.value.moves.at(-1)?.status}</p> : null}
      <ContractReading title="My franchise cap" reading={readings?.cap} error={error}>
        {cap && (
          <>
            <p>
              {cap.team} · {cap.season}
            </p>
            <dl className="kv">
              <dt>Cap</dt>
              <dd>{cap.cap?.text ?? 'Unavailable'}</dd>
              <dt>Used</dt>
              <dd>{cap.used.text}</dd>
              <dt>Room</dt>
              <dd>{cap.room?.text ?? 'Unavailable'}</dd>
            </dl>
            <p>{cap.capSource}</p>
            {!cap.cap && <p>Cap configuration unavailable; room cannot be calculated.</p>}
            <h5>Committed by future season · gross salary</h5>
            {cap.future.length ? (
              <Years years={cap.future} />
            ) : (
              <p>No future commitments in the held mirror.</p>
            )}
          </>
        )}
      </ContractReading>
      <section className="contract-controls" aria-label="Contract filters and sorting">
        <label>
          Team
          <select
            value={filters.team}
            onChange={(e) =>
              commands.dispatch('contracts.set', { field: 'team', value: e.target.value })
            }
          >
            <option value="">All teams</option>
            {options.teams.map((team) => (
              <option key={team.id} value={team.id}>
                {team.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Position
          <select
            value={filters.position}
            onChange={(e) =>
              commands.dispatch('contracts.set', { field: 'position', value: e.target.value })
            }
          >
            <option value="">All positions</option>
            {options.positions.map((pos) => (
              <option key={pos} value={pos}>
                {pos}
              </option>
            ))}
          </select>
        </label>
        <label>
          Sort
          <select
            value={filters.sort}
            onChange={(e) =>
              commands.dispatch('contracts.set', {
                field: 'sort',
                value: e.target.value as Scope['sort'],
              })
            }
          >
            <option value="name">Player</option>
            <option value="team">Team</option>
            <option value="position">Position</option>
            <option value="salary">Salary</option>
            <option value="expiration">Expiration</option>
          </select>
        </label>
        <Act verb="contracts.set" args={{ field: 'desc', value: !filters.desc }}>
          {filters.desc ? 'Descending' : 'Ascending'}
        </Act>
        <p>My team first.</p>
      </section>
      <ContractReading
        title="Every contract in the league"
        endpoint="M-078"
        reading={readings?.table}
        error={error}
      >
        {table &&
          (table.rows.length === 0 ? (
            <p>No contracts match these filters.</p>
          ) : (
            <div className="contract-table-container">
              <ContractGroup
                title={`My team · ${cap?.team ??
                  options.teams.find((team) => team.id === s.franchiseId)?.name ?? s.franchiseId}`}
                rows={table.rows.filter((row) => row.franchiseId === s.franchiseId)}
                mine whatif={whatif.reading?.value}
              />
              <ContractGroup title="Rest of the league"
                rows={table.rows.filter((row) => row.franchiseId !== s.franchiseId)} mine={false} />
            </div>
          ))}
        {table && (
          <p className="contract-source">
            Names and positions: {table.directorySource || 'Directory unavailable'} · As of:{' '}
            {table.directoryAsOf ? localDate(table.directoryAsOf) : 'Not recorded'}
          </p>
        )}
        <p>Open any player to see what each contract year costs in the Inspector.</p>
      </ContractReading>
      <ContractReading
        title="MFL salary adjustments"
        endpoint="M-031"
        reading={readings?.adjustments}
        error={error}
      >
        <p>MFL adjustments, not proven itemised dead cap.</p>
        {readings?.adjustments.value?.rows.length === 0 && (
          <p>No MFL salary adjustments in the held mirror.</p>
        )}
        <ul className="contract-adjustments">
          {readings?.adjustments.value?.rows.map((a) => (
            <li key={a.id}>
              <b>{a.team}</b>
              <span className="contract-money">{a.amount.text}</span>
              <p>{a.description || 'No description supplied by MFL'}</p>
            </li>
          ))}
        </ul>
      </ContractReading>
      <ContractReading
        title="Can't-cut list"
        endpoint="M-025"
        reading={readings?.cantCut}
        error={error}
      />
      {!supplied && <><ReleasePlans mine={s.franchiseId ?? ''} />
        <ContractActs mine={s.franchiseId ?? ''} /></>}
      <p>What-if models the charge; Make this a plan keeps a checked move. Nothing is submitted.</p>
    </div>
  );
}
