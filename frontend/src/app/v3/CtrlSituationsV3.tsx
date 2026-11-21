// Desktop v3 Control Room, Situations tab, after TheWarRoom-mock-v3-FINAL.html (screens.js
// control.situations): a 4-column grid of situation guide cards with category filter chips and direct
// step routes via resolveSituationStep and the hosted room's actions (`c`).
import { useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { situations, type Situation, type Step } from '../control/room/guide';
import { resolveSituationStep, type SituationRoute } from '../commissioner/hostedControlRoomData';
import type { HostedControlRoomState } from '../commissioner/HostedControlRoom';
import type { ControlBodyProps } from './ControlRoomV3';

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;

function Panel({ title, badge, children, foot, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; tag?: ReactNode }) {
  return <section className="pc panel sz-L">
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

/** Category filter chips with situation counts, starting with ALL. */
export function situationCategories(
  list: readonly Situation[] = situations,
): { category: string; count: number }[] {
  const names = ['ALL', ...new Set(list.map(story => story[0]))];
  return names.map(category => ({
    category,
    count: category === 'ALL' ? list.length : list.filter(item => item[0] === category).length,
  }));
}

/** Situations matching the selected category chip. */
export function filterSituations(
  category: string,
  list: readonly Situation[] = situations,
): readonly Situation[] {
  return category === 'ALL' ? list : list.filter(story => story[0] === category);
}

/** Dispatches a resolved situation step to the hosted room's tab, tool, or route message. */
export function followSituationRoute(
  route: SituationRoute,
  c: Pick<HostedControlRoomState, 'navigateTo' | 'openTool' | 'showRoute'>,
): void {
  if (route.kind === 'tab') c.navigateTo(route.tab);
  else if (route.kind === 'tool') c.openTool(route.definition);
  else c.showRoute(route.kind === 'unknown'
    ? `This situation route is not registered: ${route.route}` : route.words);
}

const markTagClass = (mark: Step[2]) =>
  mark === 'B' ? 'tag gr' : mark === 'R' ? 'tag am' : 'tag o';

const markLabel = (mark: Step[2]) =>
  mark === 'B' ? 'BUILT' : mark === 'R' ? 'RULED · NOT BUILT' : 'GAP';

export function CtrlSituationsV3({ c, badge }: ControlBodyProps) {
  const [category, setCategory] = useState('ALL');
  const categories = situationCategories();
  const visible = filterSituations(category);

  return <div className="wbody scr scr-ctrl"><div className="lay"
    style={{ gridTemplateColumns: 'minmax(0,1fr)' }}>
    <Panel title="Situations" badge={badge}
      tag={<span className="tag o">{visible.length}</span>}
      foot="guide cards · open any built step to jump to its tab or tool">
      <div className="fbar" aria-label="Situation categories">
        {categories.map(({ category: value, count }) => <HQAct key={value} className="chip"
          selected={category === value} label={`${value} situations`}
          onAction={() => setCategory(value)}>
          {value} <span className="m">{count}</span>
        </HQAct>)}
      </div>
      {c.routeWords && <St><span role="status">{c.routeWords}</span></St>}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4,minmax(0,1fr))',
        gap: '10px', padding: '10px' }} aria-label="Situations">
        {visible.map(story => <section key={story[1]} className="pc panel sz-L"
          aria-label={story[1]}>
          <div className="ph">{badge}<b className="pt">{story[1]}</b><span className="sp" />
            <span className="tag">{story[0]}</span></div>
          <div className="pb">
            {story[2].map(([words, command, mark], stepIndex) => {
              const route = resolveSituationStep(command, mark);
              return <div key={`${command}:${words}`} className="row"
                style={{ height: 'auto', minHeight: 30, padding: '6px 10px', whiteSpace: 'normal',
                  gridTemplateColumns: 'minmax(0,1fr) auto auto' }}>
                <div>
                  <div>{words}</div>
                  <code className="m">{command}</code>
                </div>
                <span className={markTagClass(mark)}>{markLabel(mark)}</span>
                <span className="acts">
                  <HQAct className="btn sm" disabled={mark !== 'B'} label={`Open ${words}`}
                    attributes={{ 'data-step': String(stepIndex), 'data-route': command }}
                    onAction={() => followSituationRoute(route, c)}>OPEN ›</HQAct>
                </span>
              </div>;
            })}
            <div className="m" style={{ padding: '6px 10px', whiteSpace: 'normal' }}>
              Still up to you: {story[3]}</div>
            <div className="m" style={{ padding: '0 10px 8px', whiteSpace: 'normal' }}>
              Check afterwards: {story[4]}</div>
          </div>
        </section>)}
      </div>
    </Panel>
  </div></div>;
}
