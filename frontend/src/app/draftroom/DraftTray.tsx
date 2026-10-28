import { useState } from 'react';
import { ChangeBar } from '../commissioner/ChangeBar';
import { useRequestKey } from '../commissioner/forms';
import { PreviewReview } from '../commissioner/PreviewReview';
import SideTray from '../commissioner/SideTray';
import { useChangePreview } from '../commissioner/useChangePreview';
import type { Draft } from './draft';
import { checkpointWords, franchiseName, pickWords, playerLine } from './draftWords';
import { changeOf, missing, noInputs, trayValue, wordsFor, type Inputs, type TrayAction }
  from './trayAction';

function Choose<K extends keyof Inputs>({ label, field, inputs, set, options }: {
  label: string; field: K; inputs: Inputs; set(next: Inputs): void;
  options: { value: string; text: string }[];
}) {
  return (
    <label>{label}
      <select value={inputs[field]} onChange={e => set({ ...inputs, [field]: e.target.value })}>
        <option value="">Choose…</option>
        {options.map(o => <option key={o.value} value={o.value}>{o.text}</option>)}
      </select>
    </label>
  );
}

function PickLine({ draft, action }: { draft: Draft; action: TrayAction }) {
  const pick = 'ref' in action ? draft.picks.find(p => p.round === action.ref.round &&
    p.original === action.ref.original && p.extra === action.ref.extra) : undefined;
  if (!pick) return null;
  const held = pick.player === '' ? '' : `: ${draft.players[pick.player]?.name ?? pick.player}`;
  return <p>Round {pick.round}, pick {pick.slot}, {franchiseName(draft, pick.owner)} —{' '}
    {pickWords(draft, pick)}{held}</p>;
}

// DraftForm is the body of every live-draft change: Go's preview of exactly what would be sent, the inputs
// the change needs, and the shared confirm. Go decides who may act and how firmly to confirm.
export function DraftForm({ action, draft }: { action: TrayAction; draft: Draft }) {
  const [inputs, setInputs] = useState<Inputs>(noInputs);
  const [reason, setReason] = useState('');
  const words = wordsFor(action, draft);
  const change = changeOf(action, draft, inputs, reason);
  const request = useRequestKey(change);
  const { preview, wait, safeguard } = useChangePreview(change, request);
  const rookies = draft.available.map(p => ({ value: p.id, text: playerLine(p) }));
  const kind = action.kind;
  const who = draft.startup ? 'player' : 'rookie';
  return (
    <section className="league-form" aria-label={words.title}>
      <PreviewReview preview={preview} confirm={
        <ChangeBar form={change?.command ?? kind} change={change} request={request} reason={reason}
          onReason={setReason} safeguard={safeguard} confirmLabel={words.confirm}
          blocked={missing(action, draft, inputs) || wait} />
      }>
        <PickLine draft={draft} action={action} />
        {(kind === 'assign' || kind === 'insert' || kind === 'change') &&
          <Choose field="player"
            label={kind === 'change' ? `The ${who} to give this pick` : draft.startup ? 'Player' : 'Rookie'}
            inputs={inputs} set={setInputs} options={rookies} />}
        {kind === 'insert' && <Choose label="Franchise that gets the extra pick" field="franchise"
          inputs={inputs} set={setInputs}
          options={draft.owners.map(o => ({ value: o.id, text: o.name }))} />}
        {kind === 'set' && (
          <label>Minutes the pick on the clock has left
            <input type="number" min={1} step={1} value={inputs.minutes}
              onChange={e => setInputs({ ...inputs, minutes: e.target.value })} />
          </label>
        )}
        {kind === 'restore' && <Choose label="Restore the draft to how it stood after" field="checkpoint"
          inputs={inputs} set={setInputs}
          options={draft.checkpoints.map(c => ({ value: c.entry, text: checkpointWords(c) }))} />}
      </PreviewReview>
    </section>
  );
}

// DraftTray opens a live-draft change in the one side tray every commissioner change opens in.
export default function DraftTray({ action, draft, opener, background, onClose }: {
  action: TrayAction; draft: Draft; opener: HTMLElement; background: HTMLElement | null; onClose(): void;
}) {
  const words = wordsFor(action, draft);
  return (
    <SideTray title={words.title} screen="draftroom" opener={opener} background={background}
      onClose={onClose} closeLabel={words.close}>
      <DraftForm key={`${draft.league}:${trayValue(action)}`} action={action} draft={draft} />
    </SideTray>
  );
}
