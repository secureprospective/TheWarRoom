import { useEffect, useState } from 'react';
import type { control } from '../../../wailsjs/go/models';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { dataState } from '../datasources/state';

function PickSetting({ param, busy }: { param: control.ParamView; busy: boolean }) {
  const [value, setValue] = useState(String(param.value));
  useEffect(() => setValue(String(param.value)), [param.value]);
  const n = Number(value);
  const valid = value.trim() !== '' && Number.isFinite(n) && n >= param.min && n <= param.max;
  const id = `setting-${param.key}`;
  return <div className="league-pick-setting">
    <label htmlFor={id}>{param.label}</label>
    <p id={`${id}-help`}>{param.description}. From {param.min} to {param.max}; default {param.default}.</p>
    <input id={id} type="number" min={param.min} max={param.max} step="any" value={value}
      aria-describedby={`${id}-help`} disabled={busy || !commands.canDispatch('data.parameter')}
      onChange={event => setValue(event.target.value)} />
    <Act variant="button" verb="data.parameter" label={`Apply ${param.label}`}
      disabled={busy || !valid || n === param.value}
      args={{ kind: 'parameter', key: param.key, position: '', value: n }}>Apply</Act>
    <Act variant="button" verb="data.parameter" label={`Reset ${param.label} to default`}
      disabled={busy || !param.overridden}
      args={{ kind: 'parameter', key: param.key, position: '', value: null }}>Reset to default</Act>
  </div>;
}

// These are league settings, not a change to the approved player valuation model.
export default function PickSettings() {
  const s = dataState.use();
  useEffect(() => { void dataState.loadParams(); }, []);
  const params = s.params?.params.filter(p => p.key === 'power.pick_weight' || p.key === 'pickvalue.discount');
  return <section aria-label="Future draft picks" aria-busy={s.pending || Boolean(s.busy)}>
    <h3>Future draft picks</h3>
    <p>Picks add to the franchise power rankings. They do not count in this season’s view or change player values.</p>
    {s.error && <p role="alert">{s.error}</p>}
    {s.result && <p role="status">{s.result}</p>}
    {!params && !s.error && <p role="status">Reading pick settings…</p>}
    {params?.map(param => <PickSetting key={param.key} param={param} busy={Boolean(s.busy)} />)}
  </section>;
}
