import { useEffect, useState } from 'react';
import { FixtureProvider } from '../data/provider';
import type { Snapshot } from '../data/contract';
import { POSITIONS } from '../data/contract';
import { Card, DENSITIES } from '../cards/Card';
import { PlayerCard } from '../cards/PlayerCard';
import { GRAVITIES, URGENCIES, SIGNALS } from '../look/channels';
import { Glyph, TIERS, POSTURES, PRIVATE_MARKERS } from '../look/Glyph';
import { Countdown } from '../look/Slots';
import { specimenPlayers } from './selection';
import '../look/look.css';
import './specimen.css';

export default function Specimen() {
  const [snapshot, setSnapshot] = useState<Snapshot>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    let active = true;
    new FixtureProvider()
      .snapshot()
      .then((value) => {
        if (active) setSnapshot(value);
      })
      .catch((cause) => {
        if (active) setError(`Fixture failed: ${String(cause)}`);
      });
    return () => {
      active = false;
    };
  }, []);
  const swatches = [
    ...SIGNALS.map((name) => ({ token: `signal-${name}`, label: name })),
    ...[0, 1, 2, 3, 4, 'line', 'line-hi', 'tx1', 'tx2', 'tx3'].map((name) => ({
      token: `surface-${name}`,
      label: `surface ${name}`,
    })),
    ...['wordmark', 'crest'].map((name) => ({
      token: `brand-${name}`,
      label: `brand ${name}`,
    })),
  ];
  return (
    <main className="twr-app specimen">
      <h1>Ring 0 look specimen · fixture only</h1>
      <section>
        <h2>Tokens</h2>
        <div className="specimen-swatches">
          {swatches.map(({ token, label }) => (
            <div key={token}>
              <span
                className="specimen-swatch"
                style={{ background: `var(--${token})` }}
              />
              {label}
            </div>
          ))}
        </div>
      </section>
      <section>
        <h2>Gravity · neutral consequence</h2>
        <div className="specimen-frames">
          {GRAVITIES.map((gravity) => (
            <Card
              key={gravity}
              gravity={gravity}
              density="tactical"
              header={<h3 className="card-title">{gravity}</h3>}
            >
              Neutral frame
            </Card>
          ))}
        </div>
      </section>
      <section>
        <h2>Urgency · countdown channel</h2>
        <div className="specimen-line">
          {URGENCIES.map((urgency) => (
            <Countdown
              key={urgency}
              urgency={urgency}
              label={`${urgency} · countdown treatment`}
            />
          ))}
        </div>
      </section>
      <section>
        <h2>Classification · monochrome</h2>
        <div className="specimen-line">
          {[...POSITIONS, ...TIERS, ...POSTURES, ...PRIVATE_MARKERS].map((name) => (
            <span className="classification-label" key={name}>
              <Glyph name={name} />
              {name}
            </span>
          ))}
        </div>
      </section>
      {error && <p role="alert">{error}</p>}
      {!snapshot && !error && <p role="status">Loading league fixture…</p>}
      {snapshot && (
        <section>
          <h2>Real rostered players · as of 2026-10-06 UTC</h2>
          {DENSITIES.map((density) => (
            <section key={density}>
              <h3>{density}</h3>
              <div className={`specimen-players specimen-${density}`}>
                {specimenPlayers(snapshot).map(({ franchiseId, playerId, reason }) => (
                  <div key={`${reason}/${playerId}`}>
                    <p className="specimen-reason">
                      {reason} · {playerId}
                    </p>
                    <PlayerCard
                      snapshot={snapshot}
                      franchiseId={franchiseId}
                      playerId={playerId}
                      asOf={new Date('2026-10-06T00:00:00Z')}
                      density={density}
                    />
                  </div>
                ))}
              </div>
            </section>
          ))}
        </section>
      )}
    </main>
  );
}
