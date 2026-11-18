import { useEffect, useState, type ReactNode } from 'react';
import type { ThinHomeEntry } from './ThinHome';
import { HQAct } from './HQAct';
import './PhoneShell.css';

export const PHONE_QUERY = '(max-width: 699px)';
export type PhoneTab = 'today' | 'live' | 'team' | 'moves' | 'league';
const tabs: { id: PhoneTab; label: string }[] = [
  { id: 'today', label: 'Today' }, { id: 'live', label: 'Live' },
  { id: 'team', label: 'Team' }, { id: 'moves', label: 'Moves' },
  { id: 'league', label: 'League' },
];

export function phoneTabEntry(tab: PhoneTab): ThinHomeEntry {
  switch (tab) {
    case 'today': return 'home';
    case 'live': return 'pulse';
    case 'team': return 'hq';
    case 'moves': return 'war';
    case 'league': return 'league';
  }
}

function phoneViewport(): boolean {
  if (typeof window === 'undefined') return false;
  return typeof window.matchMedia === 'function'
    ? window.matchMedia(PHONE_QUERY).matches : window.innerWidth < 700;
}

function usePhoneViewport(enabled: boolean): boolean {
  const [phone, setPhone] = useState(() => enabled && phoneViewport());
  useEffect(() => {
    if (!enabled) return;
    if (typeof window.matchMedia !== 'function') {
      if (typeof window.addEventListener !== 'function') return;
      const update = () => setPhone(window.innerWidth < 700);
      window.addEventListener('resize', update);
      return () => window.removeEventListener('resize', update);
    }
    const query = window.matchMedia(PHONE_QUERY);
    const update = (event: MediaQueryListEvent) => setPhone(event.matches);
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  }, [enabled]);
  return phone;
}

export function PhoneShell({ enabled = true, leagueName, activeTab, commissioner = false,
  onSelect, onControlRoom, children }: { enabled?: boolean; leagueName: string; activeTab: PhoneTab;
  commissioner?: boolean; onSelect: (tab: PhoneTab) => void; onControlRoom: () => void;
  children: ReactNode }) {
  const phone = usePhoneViewport(enabled);
  if (!enabled || !phone) return <>{children}</>;
  return <div className="phone-shell">
    <header className="phone-shell__appbar">
      <span className="phone-shell__league-name">{leagueName || 'Your league'}</span>
      {commissioner && activeTab === 'league' && <HQAct className="phone-shell__control"
        onAction={onControlRoom}>Control Room</HQAct>}
    </header>
    <div className="phone-shell__content">{children}</div>
    <nav className="phone-shell__tabs" aria-label="Phone tabs">
      {tabs.map(tab => <HQAct key={tab.id} className="phone-shell__tab"
        currentPage={activeTab === tab.id ? 'page' : undefined} onAction={() => onSelect(tab.id)}
        label={tab.label}>{tab.label}</HQAct>)}
    </nav>
  </div>;
}
