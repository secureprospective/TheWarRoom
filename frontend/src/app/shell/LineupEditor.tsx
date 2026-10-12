import './lineup.css';
import { useMemo, useEffect, type ReactNode } from 'react';
import type { Snapshot, LineupReading } from '../data/contract';
import { PlayerCard } from '../cards/PlayerCard';
import { SignalChip } from '../look/Slots';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { MatrixRoster } from './MatrixRoster';
import type { PlayerSubject } from './state';
import { useDraftCheck, draftReading } from './draftLineup';
import { changedPlayers, editReason } from '../commands/lineupEdit';
import { LineupPlan } from './LineupPlan';
import { lineupGroups, lineupText, lineupVerdict } from './roster';
import { provenanceSlot } from '../cards/playerModel';
import type { franchiseRoster } from './roster';

export default function LineupEditor({ snapshot, roster, franchiseId, selected, cap, reading, error }: {
  snapshot: Snapshot;
  reading?: LineupReading;
  error?: string;
  roster: ReturnType<typeof franchiseRoster>;
  franchiseId: string;
  selected: PlayerSubject | null;
  cap: ReactNode;
}) {
  const { draft: held, receiptId } = commands.useLineupEdit();
  const draft = held?.franchiseId === franchiseId ? held : null;
  const { check, error: checkError } = useDraftCheck(commands.checkLineup, draft, reading?.check);
  const edited = useMemo(() => draft && reading ? draftReading(snapshot, reading, draft, check) : reading,
    [snapshot, reading, draft, check]);
  const active = new Set(roster.groups[0].players.map((p) => p.id));
  const baseline = reading?.starters.map((p) => p.id) ?? [];
  const changes = draft ? changedPlayers(draft.starters, baseline) : [];
  const pending = commands.useMoves((s) => s.drafting[`lineup:${franchiseId}`]);
  const reason = editReason(reading, error);
  const unchanged = !changes.length ? 'No changes from saved lineup' : undefined;
  const saveReason = pending ? 'Checking and saving plan…' : reason ?? unchanged;
  useEffect(() => {
    const read = () => { void commands.loadMoves(franchiseId); };
    read();
    return commands.onMovesChange(read);
  }, [franchiseId]);
  const groups = useMemo(() => edited ? lineupGroups(snapshot, roster, edited) : roster.groups,
    [snapshot, roster, edited]);
  return (
    <>
      <div className="lineup-actions">
        <h4>Roster</h4>
        {!draft && <Act verb="lineup.edit" args={{ franchiseId }} disabled={Boolean(reason)}
          label={reason ?? 'Edit lineup'}>Edit lineup</Act>}
        {draft && <>
          <Act verb="lineup.reset" args={{}} />
          <Act verb="lineup.cancel" args={{}} />
          <Act verb="lineup.draft" args={{}} disabled={Boolean(saveReason)}
            label={saveReason ?? 'Check and save plan'}>Check and save plan</Act>
        </>}
      </div>
      {draft && <p>Click a starter to bench him. Click a bench player to start him.</p>}
      <p className="lineup-reading" role="status">
        {draft && edited
          ? `Draft · ${lineupVerdict(edited)}` +
            ` · ${changes.length} ${changes.length === 1 ? 'player' : 'players'} changed from saved`
          : lineupText(reading, error)}{' '}
        {reading && <>
          <SignalChip {...provenanceSlot([['Lineup', reading.provenance]])} />{' '}
          <SignalChip {...provenanceSlot([['Rules', reading.rulesSource]])} />
        </>}
      </p>
      {checkError && <p role="alert">Lineup check unavailable · {checkError}</p>}
      <LineupPlan receiptId={receiptId} franchiseId={franchiseId} snapshot={snapshot} />
      {cap}
      {groups.map((group) => (
        <section key={group.label}>
          <h5>{group.label} · {group.players.length} <SignalChip {...roster.provenance} /></h5>
          <MatrixRoster cards={group.cards} franchiseId={franchiseId} selected={selected} />
          <div className="roster-cards" aria-label={`${group.label} roster`}>
            {group.cards.map(({ model, provenance }) => {
              const toggleable = Boolean(draft) &&
                (group.label === 'Starters' || (group.label === 'Bench' && active.has(model.id)));
              const action = group.label === 'Starters' ? 'Bench' : 'Start';
              const label = toggleable ? `${action} ${model.name}` : `Inspect ${model.name}`;
              return (
                <Act
                  key={model.id}
                  verb={toggleable ? 'lineup.toggle' : 'inspector.open'}
                  args={toggleable
                    ? { playerId: model.id } : { subject: { kind: 'player', id: model.id, franchiseId } }}
                  active={selected?.id === model.id && selected.franchiseId === franchiseId}
                  label={label}
                >
                  {toggleable && (
                    <span className="lineup-affordance" data-changed={changes.includes(model.id)}>
                      {group.label === 'Starters' ? 'Starting · click to bench' : 'Bench · click to start'}
                      {changes.includes(model.id) && ' · Changed from saved'}
                    </span>
                  )}
                  <PlayerCard
                    snapshot={snapshot}
                    franchiseId={franchiseId}
                    playerId={model.id}
                    asOf={roster.asOf}
                    provenance={provenance}
                    model={model}
                  />
                </Act>
              );
            })}
          </div>
          {!group.players.length && <p className="not-wired">No players</p>}
        </section>
      ))}
    </>
  );
}
