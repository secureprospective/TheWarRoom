// @vitest-environment jsdom
import { act } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import AppLeagueMount from './AppLeagueMount';
import { FRANCHISE_KEY } from './franchise';

const counts = vi.hoisted(() => ({ mounts: 0, unmounts: 0 }));
vi.mock('./leaguePicker', async real => ({ ...await real<typeof import('./leaguePicker')>(),
  readAppLeaguePeople: async (league: string) => ({ league, yourFranchise: 'one',
    seats: [{ franchise: 'one', name: 'One' }, { franchise: 'two', name: 'Two' }] }) }));
vi.mock('./WarRoomNative', async () => {
  const { useEffect, useState } = await import('react');
  const { HQAct } = await import('./HQAct');
  return { default: ({ mine }: { mine: string }) => {
    const [draft, setDraft] = useState(0);
    useEffect(() => { counts.mounts++; return () => { counts.unmounts++; }; }, []);
    return <HQAct onAction={() => setDraft(n => n + 1)}>{mine} draft {draft}</HQAct>;
  } };
});
vi.mock('./NativeFranchiseHQ', () => ({ default: () => <p>HQ</p> }));

let close: (() => Promise<void>) | undefined;
afterEach(async () => { await close?.(); localStorage.clear(); vi.unstubAllGlobals(); });
it('retains War Room on revisit, but drops it on franchise change and disposal', async () => {
  vi.stubGlobal('IS_REACT_ACT_ENVIRONMENT', true);
  const host = document.createElement('div'); document.body.append(host);
  const root = createRoot(host);
  close = async () => { await act(async () => root.unmount()); host.remove(); };
  const render = async (view: 'war' | 'hq') => {
    await act(async () => { root.render(<AppLeagueMount league={{ id: 'native.war', name: 'League',
      teams: 2, app: true }} view={view} leave={() => {}} />); });
  };
  await render('hq');
  expect(counts.mounts).toBe(0);
  await render('war');
  await act(async () => host.querySelector('button')!.click());
  await render('hq');
  expect(host.querySelector('[hidden]')?.textContent).toContain('one draft 1');
  await render('war');
  expect(host.textContent).toContain('one draft 1');
  expect(counts.mounts).toBe(1);
  const select = host.querySelector('select')!;
  await act(async () => {
    select.value = 'two'; select.dispatchEvent(new Event('change', { bubbles: true }));
  });
  expect(localStorage.getItem(`${FRANCHISE_KEY}:native.war`)).toBe('two');
  expect(host.textContent).toContain('two draft 0');
  expect(counts.mounts).toBe(2); expect(counts.unmounts).toBe(1);
});
