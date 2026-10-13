import { stateLabel } from '../shell/moveLabels';
import { scrollWorkIntoView } from '../shell/workScroll';
import { WhatIfFigures } from './WhatIfFigures';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { RailStrip } from '../shell/EnvelopeRail';
import { localDate } from '../clock/labels';
import {
  contractActAction, reconcileContractMove, keepLabel, onContractActsChange, useContractMove,
  type ContractActPlan,
  type ContractActRequest,
} from './contractActs';
import '../pool/pool.css';

export function Figures({ plan }: { plan: ContractActPlan }) {
  return <div className="contract-act-figures"><p>My team's cap this season and gross commitments in
    future seasons — future salaries are inferred.</p>
    <WhatIfFigures figures={plan.teamFigures} label="Team cap before and after this Act" /></div>;

}
export function ContractActCard({ plan, mine, revealing = false }: {
  plan: ContractActPlan; mine: string; revealing?: boolean;
}) {
  const card = useRef<HTMLElement>(null);
  const reveal = useContractMove(state => state.reveal);
  useEffect(() => {
    if (!revealing || reveal !== plan.id || !card.current) return;
    scrollWorkIntoView(card.current);
    card.current.focus({ preventScroll: true });
    useContractMove.setState({ reveal: '' });
  }, [revealing, reveal, plan.id]);
  const finished = ['withdrawn', 'landed', 'failed'].includes(plan.receipt?.state ?? '');
  const post = <pre className="bid-post" aria-label="Contract ProBoards post">{plan.post}</pre>;
  const action = (kind: ContractActRequest['action'], label: string) =>
    <Act verb="contracts.set" variant="text" label={label}
    args={{ field: 'contract-act', value: { mine, id: plan.id, action: kind } }}>{label}</Act>;
  return <article ref={card} tabIndex={-1} className="bid-plan contract-act-card" aria-label={plan.label}>
    <h5>{plan.label}</h5>
    {!finished && <p className="bid-line"><span>{plan.price}</span></p>}
    {plan.receipt && <RailStrip receipt={{ ...plan.receipt, audit: plan.receipt.audit.map(entry =>
      entry.to === 'ready' ? { ...entry, note: `Ready: ${plan.reasons.filter(reason =>
        reason.startsWith('Not checked:')).length} things not checked` } : entry) }} />}
    {!finished && <><ul className="bid-note">{plan.reasons.map(reason => <li key={reason}>{reason}</li>)}</ul>
      <Figures plan={plan} />
      <p className="bid-note">Post this to the commissioner on ProBoards. Nothing is posted by the app.</p>
      {post}
      <div className="bid-actions">
        {plan.canHandOff && action('handoff', 'Copy request and hand off')}
        {plan.canPost && action('posted', 'I posted it')}
        {plan.canWithdraw && action('withdraw', ['bid_pending', 'not_verified']
          .includes(plan.receipt?.state ?? '')
          ? 'Commissioner did not process it' : plan.canPost ? "I didn't post it" : 'Remove plan')}
        {plan.canRebase && action('rebase', "Reprice from today's MFL mirror")}
      </div>
      <details className="bid-details"><summary>How this is counted</summary><p>{plan.how}</p></details></>}
    <details className="bid-details"><summary>History</summary>{finished && post}
      <ol>{plan.receipt?.audit.map((entry, i) =>
        <li key={i}><time dateTime={entry.at}>{localDate(entry.at)}</time> {entry.note}</li>)}</ol></details>
  </article>;
}

export function ContractActMenu({ mine, player }: { mine: string; player: string }) {
  return <Act verb="contracts.set" label="Contract Acts…"
    args={{ field: 'contract-open', value: { mine, player } }}>
    Contract Acts…</Act>;
}
export function ContractMovePanel({ mine }: { mine: string }) {
  const target = useContractMove();
  return target.mine === mine && target.player
    ? <ContractMoveForm key={`${mine}:${target.player}`} mine={mine} player={target.player} /> : null;
}
export function ContractMovePreview({ plan }: { plan: ContractActPlan }) {
  return <><p><span>{plan.price}</span></p>
    <ul>{plan.reasons.map(reason => <li key={reason}>{reason}</li>)}</ul>
    <Figures plan={plan} />
    <details><summary>How this is counted</summary><p>{plan.how}</p></details></>;
}
export function ContractMoveFeedback({ plan }: { plan: ContractActPlan }) {
  return <div className="whatif-plan-line" role="status">
        <p>Plan: {stateLabel(plan.receipt?.state ?? 'draft', plan.intent)}</p>
        {plan.receipt?.state !== 'ready' && <ul>{plan.reasons.map(reason =>
          <li key={reason}>{reason}</li>)}</ul>}
        <Act verb="contracts.set" label="See it in My moves"
          args={{ field: 'contract-reveal', value: plan.id }}>See it in My moves</Act>
      </div>;
}
// Switching Act keeps the owner's terms where the new Act allows them, else its nearest allowed ones.
export function switchAct(request: ContractActRequest, intent: string, options: ContractActPlan[]) {
  const option = options.find(option => option.intent === intent);
  const pick = (value: number | undefined, allowed: number[] = []) =>
    allowed.length === 0 || allowed.includes(value ?? 0) ? value : Math.min(Math.max(value ?? 1,
      allowed[0]), allowed[allowed.length - 1]);
  return { ...request, intent, source: option?.figures[0]?.season,
    destination: option?.figures[option.figures.length - 1]?.season,
    years: pick(request.years, option?.years), level: pick(request.level, option?.levels.map(l => l.value)) };
}
// The Act, tender level and term pickers, laid out from the choices Go sends with each Act.
export function ContractActChoices({ request, options, onChange }: {
  request: ContractActRequest; options: ContractActPlan[]; onChange: (request: ContractActRequest) => void;
}) {
  const chosen = options.find(option => option.intent === request.intent);
  return <>
    <label>Act<select value={request.intent}
      onChange={e => onChange(switchAct(request, e.target.value, options))}>
      {options.map(option => <option key={option.intent} value={option.intent}>{option.act}</option>)}
    </select></label>
    {Boolean(chosen?.levels.length) && <label>Tender level<select value={request.level}
      onChange={e => onChange({ ...request, level: Number(e.target.value) })}>
      {chosen?.levels.map(l => <option key={l.value} value={l.value}>{l.label}</option>)}
    </select></label>}
    {Boolean(chosen?.years.length) && <label>Years<select value={request.years}
      onChange={e => onChange({ ...request, years: Number(e.target.value) })}>
      {chosen?.years.map(v => <option key={v}>{v}</option>)}
    </select></label>}
  </>;
}
function ContractMoveForm({ mine, player }: { mine: string; player: string }) {
  const form = useRef<HTMLElement>(null);
  useLayoutEffect(() => {
    if (!form.current) return;
    scrollWorkIntoView(form.current);
    form.current.focus({ preventScroll: true });
  }, [mine, player]);
  const [request, setRequest] = useState<ContractActRequest>({
    mine, player, action: 'preview', intent: 'rfa.tender', level: 1, years: 1,
    move: '1', source: 0, destination: 0,
  });
  const [options, setOptions] = useState<ContractActPlan[]>([]);
  const [plan, setPlan] = useState<ContractActPlan>();
  const [error, setError] = useState('');
  const move = useContractMove();
  const { kept, saving, error: keepError } = move;
  const keep = keepLabel(request, move);
  useEffect(() => {
    let active = true, sequence = 0;
    const load = () => {
      const current = ++sequence;
      void contractActAction({ mine, action: 'read' }).then(reading => {
        if (active && current === sequence) reconcileContractMove(mine, reading.value ?? []);
      }, cause => { if (active && current === sequence)
        setError(cause instanceof Error ? cause.message : String(cause)); });
    };
    load();
    const off = onContractActsChange(load), movesOff = commands.onMovesChange(load);
    return () => { active = false; off(); movesOff(); };
  }, [mine]);
  useEffect(() => {
    let active = true;
    setOptions([]); setPlan(undefined); setError('');
    void contractActAction({ mine, player, action: 'options' }).then(r => {
      if (!active) return;
      const choices = r.value ?? []; setOptions(choices);
      if (choices[0]) setRequest(current => ({ ...current, intent: choices[0].intent }));
    }, e => { if (active) setError(e instanceof Error ? e.message : String(e)); });
    return () => { active = false; };
  }, [mine, player]);
  useEffect(() => {
    if (!options.some(option => option.intent === request.intent)) return;
    let active = true;
    setPlan(undefined); useContractMove.setState({ error: '' }); setError('');
    void contractActAction({ ...request, mine, player }).then(
      r => { if (active) { setPlan(r.value?.[0]);
        const label = r.value?.[0]?.label.replace('Extend him', 'Extend').replace(' · ', ' ') ?? '';
        useContractMove.setState({ label }); } },
      e => { if (active) setError(e instanceof Error ? e.message : String(e)); },
    );
    return () => { active = false; };
  }, [request, mine, player, options]);
  return <section ref={form} tabIndex={-1} className="contract-reading contract-move"
    aria-label="Contract move">
    <header className="contract-move-head"><h4>Contract move</h4>
    <Act verb="contracts.set" args={{ field: 'contract-close', value: '' }}
      variant="icon" label="Close contract move">Close</Act></header>
    <p className="contract-move-subject">{options[0]?.subject ?? plan?.subject}</p>
    <fieldset disabled={saving} className="contract-act-form">
      <ContractActChoices request={request} options={options} onChange={setRequest} />
      {request.intent === 'contract.restructure' && <>
        <label>Source season<input type="number" value={request.source || ''}
          onChange={e => setRequest({ ...request, source: Number(e.target.value) })}
        /></label>
        <label>Destination season<input type="number" value={request.destination || ''}
          onChange={e => setRequest({ ...request, destination: Number(e.target.value) })}
        /></label>
        <label>Move ($M)<input value={request.move}
          onChange={e => setRequest({ ...request, move: e.target.value })}
        /></label>
      </>}
      {error && <p role="alert">{error}</p>}
      {!error && options.length === 0 && <p>No eligible Acts found.</p>}
      {plan && <><ContractMovePreview plan={plan} />
        <div className="bid-actions"><Act verb="contracts.set" variant="text"
          disabled={keep !== 'Keep plan'} label={keep}
          args={{ field: 'contract-keep', value: { ...request, mine, player } }}>{keep}</Act></div>
        {keepError && <p role="alert">{keepError}</p>}</>}
      {kept && <ContractMoveFeedback plan={kept} />}
    </fieldset>
  </section>;
}

export default function ContractActs({ mine, revealing = false }: { mine: string; revealing?: boolean }) {
  const [plans, setPlans] = useState<ContractActPlan[]>();
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true, sequence = 0;
    setPlans(undefined); setError('');
    const load = () => {
      const current = ++sequence;
      if (!mine) return;
      void contractActAction({ mine, action: 'read' }).then(r => {
        if (active && sequence === current) { setPlans(r.value); setError(''); }
      }, e => { if (active && sequence === current) setError(e instanceof Error ? e.message : String(e)); });
    };
    load(); const off = onContractActsChange(load), movesOff = commands.onMovesChange(load);
    return () => { active = false; off(); movesOff(); };
  }, [mine]);
  return <section className="release-plans" aria-label="Contract Act plans" data-endpoint="C-06.5">
    <h4>Contract Act plans</h4>{error && <p role="alert">{error}</p>}
    {plans?.length === 0 && <p>Use Contract Acts… in one of my players' menus to keep a draft.</p>}
    {plans?.map(plan => <ContractActCard key={plan.id} mine={mine} plan={plan} revealing={revealing} />)}
  </section>;
}
