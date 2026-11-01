import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { RuleInput } from '../commissioner/RuleInput';
import { draftEditor } from '../commissioner/draftEditor';
import { surfaceActions } from '../commissioner/surfaceActions';
import { migration } from './migration';

// The same catalogue inputs used for later rule changes answer the import's remaining questions.
export default function MoveMFL() {
  const s = migration.use();
  useEffect(() => surfaceActions.register('move-mfl', args => {
    if (args.action === 'clear') { migration.input(''); return; }
    if (args.action === 'check') return migration.prepare();
    if (args.action === 'pick') return migration.prepare(args.value);
    if (args.action === 'open') return migration.importLeague();
  }), []);
  const blocked = migration.blocked();
  const edit = () => {
    draftEditor.handler = migration.edit;
  };
  return <details className="move-mfl">
    <summary>Move my MFL league</summary>
    <section className="league-form" aria-label="Move my MFL league" aria-busy={s.pending}
      onFocusCapture={edit}>
      <p>Bring your league’s settings, teams, players and picks into the app. Nothing on MFL changes.</p>
      <fieldset disabled={s.pending || Boolean(s.done)}>
        <legend>Read your league</legend>
        <label>MFL league ID<input inputMode="numeric" autoComplete="off" value={s.id}
          onChange={e => migration.input(e.target.value)} /></label>
        <label>Commissioner’s MFL API key (only if needed)<input type="password" autoComplete="off"
          value={s.key} onChange={e => migration.key(e.target.value)} /></label>
        <small>Saved in the OS keyring for this MFL league only. Leave blank to use its saved key.</small>
        <Act verb="commissioner.surface" variant="button" label="Prepare MFL league"
          disabled={!s.id.trim() || s.pending || Boolean(s.done)}
          args={{ screen: 'move-mfl', action: 'check', value: '' }}>Prepare</Act>
      </fieldset>
      {s.pending && <p role="status">Reading and checking the league…</p>}
      {s.error && <p role="alert">{s.error}</p>}
      {s.prep && <>
        <h4>{s.prep.name} · {s.prep.season}</h4>
        <p>{s.prep.franchises} teams · {s.prep.players} players · {s.prep.picks} picks</p>
        {s.prep.problems.length > 0 && <div role="alert"><h5>Fix these before importing</h5>
          <ul>{s.prep.problems.map(p => <li key={p}>{p}</li>)}</ul></div>}
        {s.prep.ready && <fieldset disabled={s.pending || Boolean(s.done)}>
          <legend>Fill the rest from</legend>
          <label>Starting settings<select value={s.preset} onChange={e => {
            void surfaceActions.run({ screen: 'move-mfl', action: 'pick', value: e.target.value });
          }}>
            <option value="legacy_nfl">Legacy NFL</option><option value="default32">32-Team Default</option>
            <option value="mirror16">16-Team Mirror</option>
          </select></label>
          <p>MFL’s settings win. This bundle only fills settings MFL does not hold;
            every setting can change later.
          </p>
          {s.prep.notes.length > 0 && <ul>{s.prep.notes.map(n => <li key={n}>{n}</li>)}</ul>}
          {s.prep.waiting.length > 0 && <h5>Settings still waiting for you</h5>}
          {s.prep.waiting.map(f => <RuleInput key={f.field} schema={f.schema} label={f.label}
            draft={s.drafts[f.field]} path={[f.field]}
            onChange={draft => migration.answer(f.field, draft)} />)}
          <label>Reason for importing<textarea value={s.reason}
            onChange={e => migration.reason(e.target.value)} /></label>
          <p>Confirm once to import the league and these answers together. The new league opens when recorded.
          </p>
          {blocked && <p>{blocked}</p>}
          <Act verb="commissioner.surface" variant="button" label="Confirm import" disabled={Boolean(blocked)}
            args={{ screen: 'move-mfl', action: 'open', value: '' }}>Confirm import</Act>
        </fieldset>}
      </>}
      {s.done && <><p role="status">Imported. The league is recorded in the app.</p>
        <Act verb="commissioner.surface" variant="button" label="Move another league"
          args={{ screen: 'move-mfl', action: 'clear', value: '' }}>Move another league</Act></>}
    </section>
  </details>;
}
