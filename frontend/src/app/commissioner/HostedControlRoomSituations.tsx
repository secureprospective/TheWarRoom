import { useState } from 'react';
import { HQAct } from '../shell/HQAct';
import { situations } from '../control/room/guide';
import { resolveSituationStep, type HostedControlTab, type SituationRoute } from './hostedControlRoomData';
import type { ActDefinition } from './actTable';

export function HostedControlRoomSituations({ onTab, onTool, onUnavailable }: {
  onTab(tab: HostedControlTab): void; onTool(definition: ActDefinition): void;
  onUnavailable(words: string): void;
}) {
  const [category, setCategory] = useState('ALL');
  const [selected, setSelected] = useState(0);
  const categories = ['ALL', ...new Set(situations.map(story => story[0]))];
  const story = situations[selected];
  const follow = (route: SituationRoute) => {
    if (route.kind === 'tab') onTab(route.tab);
    else if (route.kind === 'tool') onTool(route.definition);
    else onUnavailable(route.kind === 'unknown'
      ? `This situation route is not registered: ${route.route}` : route.words);
  };
  return <div className="cr-col hosted-situations">
    <div className="cr-chips" aria-label="Situation categories">
      {categories.map(value => <HQAct key={value} className="cr-chip" selected={category === value}
        label={`${value} situations`} onAction={() => {
          setCategory(value);
          setSelected(value === 'ALL' ? 0 : situations.findIndex(item => item[0] === value));
        }}>{value}<span className="c">
          {value === 'ALL' ? situations.length : situations.filter(item => item[0] === value).length}
        </span></HQAct>)}
    </div>
    <div className="cr-sit hosted-situation-list" aria-label="Situations">
      {situations.map((item, index) => category === 'ALL' || item[0] === category ?
        <HQAct key={item[1]} selected={selected === index} onAction={() => setSelected(index)}
          label={item[1]} attributes={{ 'data-situation': String(index) }}>
          <span className="cat">{item[0]}</span><b>{item[1]}</b>
          <span className="fs">{item[2].length} steps · first: {item[2][0]![0]}</span>
        </HQAct> : null)}
    </div>
    {story && <section className="cr-panel hosted-situation-detail" aria-label={story[1]}>
      <div className="cr-ph">{story[0]} <span className="sp" />{story[1]}</div>
      <div className="cr-pb">
        <ol className="hosted-situation-steps">
          {story[2].map(([words, command, mark], stepIndex) => {
            const route = resolveSituationStep(command, mark);
            return <li key={`${command}:${words}`}>
              <span>{words}</span>
              <code>{command}</code>
              <span className={`cr-st ${mark === 'B' ? 'set' : mark === 'R' ? 'plan' : 'cr-np'}`}>
                {mark === 'B' ? 'BUILT' : mark === 'R' ? 'RULED · NOT BUILT' : 'GAP'}
              </span>
              <HQAct className="cr-btn sm" disabled={mark !== 'B'} label={`Open ${words}`}
                attributes={{ 'data-step': String(stepIndex), 'data-route': command }}
                onAction={() => follow(route)}>OPEN ›</HQAct>
            </li>;
          })}
        </ol>
        <p className="cr-note">Still up to you: {story[3]}</p>
        <p className="cr-note ok">Check afterwards: {story[4]}</p>
      </div>
    </section>}
  </div>;
}
