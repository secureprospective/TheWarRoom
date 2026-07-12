import { useEffect, useState } from 'react';
import { GetCurrentPhase } from '../../../wailsjs/go/main/App';

// LeagueControls is the commissioner off-common-path surface (design D6): pure calendar ops
// (ADVANCE_PHASE / ROLLOVER_SEASON / SET_SIGNING_WINDOW) live here, away from the per-player
// workspace. Slice 1 ships the stub + the current phase; the controls themselves port over from
// the dev panel in a later slice (D8 phased cutover — the dev panel still carries them until then).

export function LeagueControls() {
  const [phase, setPhase] = useState('…');
  useEffect(() => {
    void (async () => {
      const r = await GetCurrentPhase();
      setPhase(r.ok ? r.phase : `? (${r.detail})`);
    })();
  }, []);

  return (
    <div className="border border-[#29344a] bg-[#0e1420] p-8 text-[#e2e8f0]">
      <h2 className="text-[18px] font-bold">League Controls</h2>
      <div className="mt-3 inline-flex items-center gap-2 border border-[rgba(240,180,41,0.35)] bg-[rgba(240,180,41,0.14)] px-2.5 py-1 text-[11px] font-bold uppercase tracking-[0.07em] text-[#f0b429]">
        <span className="h-[7px] w-[7px] rounded-full bg-[#f0b429]" /> {phase}
      </div>
      <p className="mt-5 max-w-[520px] text-[13px] text-[#93a1b8]">
        Commissioner calendar controls — advance the season phase, roll the season over, and open or
        close the free-agency signing window — move here from the dev panel in a later slice. For now
        run them from the <b className="text-[#e2e8f0]">B7a: Transactions (dev)</b> tab.
      </p>
    </div>
  );
}
