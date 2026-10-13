import { useEffect } from 'react';
import type { Snapshot } from '../data/contract';
import { Act } from '../commands/Act';
import { ArchiveReading } from '../market/ReadingView';
import type { TeamPanelReading } from './contract';
import { counterState } from './state';
import { EdgeBar, TeamLink, number, percent, teamIdentity } from './Displays';

export function TeamReading({
  reading,
  snapshot,
}: {
  reading?: TeamPanelReading;
  snapshot?: Snapshot;
}) {
  const v = reading?.value;

  const identity = v && teamIdentity(snapshot, v.fid, v.name);
  return (
    <ArchiveReading
      reading={reading}
      countText={v?.countText}
      title={identity?.name ?? 'Team trade history'}
    >
      {v && (
        <>
          <p>{v.asOf}</p>
          <p className="counter-legend">{v.words.legend}</p>
          {!identity?.matched && (
            <p>Archive identity · not matched to current league; no identity guessed.</p>
          )}
          {v.empty ? (
            <p>{v.note}</p>
          ) : (
            <>
              <p>{v.dayText}</p>
              <p>{v.sinceText}</p>
              <EdgeBar edge={v.day} label={v.words.day} />
              <EdgeBar edge={v.since} label={v.words.since} />
              <section data-endpoint="L-27">
                <h5>Seasons and standing</h5>
                <p>{v.seasonSay}</p>
                <div className="counter-seasons">
                  {v.seasons.map((c) => (
                    <section
                      key={c.season}
                      className="counter-season"
                      data-flagged={c.flagged}
                      title={c.title}
                    >
                      <h5>{c.label}</h5>
                      <p>{c.mostlyText}</p>
                      <div className="counter-bars" aria-label={c.title}>
                        <label>
                          For now <meter min="0" max={Math.max(c.n, 1)} value={c.now} /> {c.nowText}
                        </label>
                        <label>
                          For later <meter min="0" max={Math.max(c.n, 1)} value={c.later} />{' '}
                          {c.laterText}
                        </label>
                        <label>
                          Like for like <meter min="0" max={Math.max(c.n, 1)} value={c.swap} />{' '}
                          {c.swapText}
                        </label>
                      </div>
                      <p>Net on the day {number(c.net)}</p>
                      {c.flagged && <p>Against its standing · {c.againstText} trade sides</p>}
                    </section>
                  ))}
                </div>
              </section>
              <section>
                <h5>What its trades did</h5>
                {v.posture.map((p) => (
                  <p key={p.id} title={p.title}>
                    {p.label}: {percent(p.p)}{' '}
                    {p.tone === 'down' ? v.words.down : p.tone === 'up' ? v.words.up : ''}
                  </p>
                ))}
                {v.asTier.map((t) => (
                  <div className="market-row" key={t.tier}>
                    <b>{t.label}</b>
                    <span>{t.text}</span>
                    <EdgeBar edge={t.edge} label={t.label} />
                  </div>
                ))}
              </section>
              <section>
                <h5>Trade averages</h5>
                <p>{v.words.picks}: {number(v.behaviour.picks.mean)} a trade</p>
                <p>{v.words.age}: {number(v.behaviour.age.mean, 1)} years a trade</p>
                <p>{v.words.partners}: {v.partnersText}</p>
              </section>
              <section data-endpoint="L-28">
                <h5>Assets sent and taken</h5>
                <p>{v.kindSay}</p>
                {v.kinds.map((k) => (
                  <div className="market-row" key={k.kind}>
                    <b>{k.name}</b>
                    <span>{k.text}</span>
                    <label>
                      Sent · {number(k.outW)} forecast wins
                      <meter min="0" max={v.flowScale} value={k.outW} />
                    </label>
                    <label>
                      Taken · {number(k.inW)} forecast wins
                      <meter min="0" max={v.flowScale} value={k.inW} />
                    </label>
                  </div>
                ))}
              </section>
              <section data-endpoint="L-29">
                <h5>Trade partners</h5>
                {v.partners[0] && (
                  <p>{v.words.partner}: <TeamLink snapshot={snapshot}
                    fid={v.partners[0].fid} name={v.partners[0].name} />{' '}
                    {v.partners[0].tradesText}</p>
                )}
                {v.partners.map((p) => (
                  <div className="market-row" key={p.fid}>
                    <TeamLink snapshot={snapshot} fid={p.fid} name={p.name} />
                    <span>
                      {p.tradesText} · {p.netText}
                    </span>
                    <EdgeBar edge={p.net} label="Average expected wins with this partner" />
                  </div>
                ))}
                <p>{v.partnersSay}</p>
              </section>
            </>
          )}
        </>
      )}
    </ArchiveReading>
  );
}
export default function TeamPanel({ fid, snapshot }: { fid: string; snapshot?: Snapshot }) {
  const s = counterState.use();
  useEffect(() => {
    void counterState.team(fid);
  }, [fid, s.applied]);
  return (
    <div className="market counterparties">
      {s.teamPending[fid] && (
        <p role="status">Updating team history; any figures still describe its previous scope.</p>
      )}
      <TeamReading reading={s.panels[fid]} snapshot={snapshot} />
      {s.panels[fid]?.status === 'failed' && (
        <Act variant="button" verb="market" args={{ field: 'counterpart-retry', value: '' }}>
          Retry team history
        </Act>
      )}
    </div>
  );
}
