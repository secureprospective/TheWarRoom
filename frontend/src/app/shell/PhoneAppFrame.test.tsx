// @vitest-environment jsdom
import { act, useState } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import type { ReactNode } from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import type { HomeBrief } from './homeReads';
import { answerPhoneQuestion } from './phoneAsk';
import { PhoneAppFrame, type PhoneTab } from './PhoneAppFrame';
import type { LeagueChoice } from './leaguePicker';

const league: LeagueChoice = { id: 'one', name: 'Test League', app: true };
const another: LeagueChoice = { id: 'two', name: 'Second League', app: true };
const brief = { results: { week: 8, games: [{ home: 'mine', away: 'other', homeName: 'My Team',
  awayName: 'Rivals' }], scores: [{ franchise: 'mine', points: 28.4 },
  { franchise: 'other', provisional: 24.1 }] }, at: {}, errors: {} } as unknown as HomeBrief;
let root: Root | undefined;
let host: HTMLDivElement | undefined;
afterEach(async () => {
  if (root) await act(async () => root!.unmount());
  host?.remove(); root = undefined; host = undefined;
  Reflect.deleteProperty(window, 'matchMedia');
  Reflect.deleteProperty(window, 'innerWidth');
});
function setViewport(width: number) {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width });
  Object.defineProperty(window, 'matchMedia', { configurable: true, value: vi.fn((media: string) => ({
    media, matches: width < 700, addEventListener: vi.fn(), removeEventListener: vi.fn(),
  })) });
}
async function renderFrame(width: number, node: ReactNode) {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  setViewport(width);
  host = document.createElement('div'); document.body.append(host); root = createRoot(host);
  await act(async () => root!.render(node));
}
async function click(element: Element) {
  await act(async () => (element as HTMLButtonElement).click());
}
function tabButtons() {
  return [...host!.querySelectorAll('[role="tab"]')] as HTMLButtonElement[];
}

function FrameHarness({ badges }: { badges?: Partial<Record<PhoneTab, number>> }) {
  const [tab, setTab] = useState<PhoneTab>('today');
  return <PhoneAppFrame activeTab={tab} leagues={[league]} selectedLeagueId={league.id}
    weekLabel="WK 8" badges={badges} onSelectTab={setTab} onSelectLeague={() => {}}
    homeReadSource={<p>Private Home read holder</p>}><p>Desktop route content</p></PhoneAppFrame>;
}

it('lands on the phone frame with five switchable placeholder tabs and live data slots', async () => {
  await renderFrame(390, <FrameHarness />);
  expect(host!.querySelector('.phone-app__league')?.textContent).toContain('Test League');
  expect(host!.querySelector('.phone-app__league')?.textContent).toContain('WK 8');
  expect(host!.querySelector('.phone-app__bar')?.textContent).not.toContain('Mock');
  expect(host!.querySelector('.phone-app__ask')?.textContent).toBe('Ask');
  expect(tabButtons().map(button => button.textContent?.trim())).toEqual([
    'Today', 'Live', 'Team', 'Moves', 'League',
  ]);
  expect(host!.querySelector('.phone-app__slot h1')?.textContent).toBe('Today');
  expect(host!.textContent).not.toContain('Desktop route content');
  expect(host!.querySelector('.phone-app__data')?.hasAttribute('hidden')).toBe(true);
  expect(host!.querySelectorAll('.phone-app__badge')).toHaveLength(0);
  await click(tabButtons()[1]);
  expect(host!.querySelector('.phone-app__slot h1')?.textContent).toBe('Live');
  expect(tabButtons()[1].getAttribute('aria-selected')).toBe('true');
  await act(async () => tabButtons()[1].dispatchEvent(new KeyboardEvent('keydown', {
    key: 'ArrowRight', bubbles: true,
  })));
  expect(host!.querySelector('.phone-app__slot h1')?.textContent).toBe('Team');
  expect(document.activeElement).toBe(tabButtons()[2]);
});

it('opens the admitted-league picker, switches with a toast, and dismisses sheets accessibly', async () => {
  const onSelectLeague = vi.fn();
  await renderFrame(430, <PhoneAppFrame activeTab="today" leagues={[league, another]}
    selectedLeagueId={league.id} weekLabel="WK 8" onSelectTab={() => {}}
    onSelectLeague={onSelectLeague} homeReadSource={null}><p>Desktop</p></PhoneAppFrame>);
  await click(host!.querySelector('.phone-app__league')!);
  expect(host!.querySelector('[role="dialog"]')?.getAttribute('aria-modal')).toBe('true');
  expect(host!.querySelector('#phone-app-leagues-title')?.textContent).toBe('Your leagues');
  const rows = [...host!.querySelectorAll('.phone-app__league-row')];
  expect(rows[0].getAttribute('aria-current')).toBe('true');
  await click(rows[1]);
  expect(onSelectLeague).toHaveBeenCalledWith('two');
  expect(host!.querySelector('.phone-app__toast')?.textContent).toBe('Switched to Second League');
  expect(host!.querySelector('[role="dialog"]')).toBeNull();

  await click(host!.querySelector('.phone-app__ask')!);
  expect(host!.querySelector('#phone-app-ask-title')?.textContent).toBe('Ask');
  await act(async () => document.dispatchEvent(new KeyboardEvent('keydown', {
    key: 'Escape', bubbles: true,
  })));
  expect(host!.querySelector('[role="dialog"]')).toBeNull();
  await click(host!.querySelector('.phone-app__league')!);
  await click(host!.querySelector('.phone-app__scrim')!);
  expect(host!.querySelector('[role="dialog"]')).toBeNull();

  await click(host!.querySelector('.phone-app__ask')!);
  const handle = host!.querySelector('.phone-app__grab')!;
  await act(async () => {
    handle.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, clientY: 10 }));
    handle.dispatchEvent(new MouseEvent('pointerup', { bubbles: true, clientY: 80 }));
  });
  expect(host!.querySelector('[role="dialog"]')).not.toBeNull();
  await act(async () => {
    handle.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, clientY: 10 }));
    handle.dispatchEvent(new MouseEvent('pointerup', { bubbles: true, clientY: 81 }));
  });
  expect(host!.querySelector('[role="dialog"]')).toBeNull();
});

it('keeps account actions at the foot of the league sheet with the existing controls', async () => {
  const onAuthenticator = vi.fn(), onSignOut = vi.fn();
  await renderFrame(390, <PhoneAppFrame activeTab="today" leagues={[league]}
    selectedLeagueId={league.id} accountName="owner" onSelectTab={() => {}}
    onAuthenticator={onAuthenticator} onSignOut={onSignOut} onSelectLeague={() => {}}>
    <p>Desktop</p>
  </PhoneAppFrame>);
  await click(host!.querySelector('.phone-app__league')!);
  const list = host!.querySelector('.phone-app__league-list')!;
  const account = host!.querySelector('.phone-app__account')!;
  expect(list.nextElementSibling).toBe(account);
  expect(account.querySelector('.phone-app__account-name')?.textContent).toContain('owner');
  const rows = [...account.querySelectorAll('button')];
  expect(rows.map(button => button.textContent)).toEqual(['Authenticator', 'Sign out']);
  expect(rows.every(button => button.classList.contains('phone-app__league-row'))).toBe(true);
  await click(rows[0]);
  expect(onAuthenticator).toHaveBeenCalledOnce();

  await click(host!.querySelector('.phone-app__league')!);
  const signOut = [...host!.querySelectorAll('.phone-app__account button')]
    .find(button => button.textContent === 'Sign out')!;
  await click(signOut);
  expect(onSignOut).toHaveBeenCalledOnce();
});

it('keeps Ask on the app shell and answers only from the existing Today score reading', async () => {
  await renderFrame(390, <PhoneAppFrame activeTab="live" leagues={[league]}
    selectedLeagueId={league.id} askBrief={brief} askMine="mine" onSelectTab={() => {}}
    onSelectLeague={() => {}}><p>Desktop</p></PhoneAppFrame>);
  await click(host!.querySelector('.phone-app__ask')!);
  const suggestion = host!.querySelector('.phone-app__suggestions button')!;
  await click(suggestion);
  expect(host!.querySelector('.phone-app__answer')?.textContent).toContain('You are ahead of Rivals');
  expect(answerPhoneQuestion('Who is injured?', brief, 'mine')).toMatchObject({ label: 'NOT ANSWERED',
    text: expect.stringContaining('Nothing was sent or saved') });
  expect(answerPhoneQuestion('Am I winning?', undefined, 'mine')).toMatchObject({
    label: 'READING UNAVAILABLE', text: expect.stringContaining('No score reading') });
  await click(host!.querySelector('.phone-app__close')!);
  expect(host!.querySelector('[role="dialog"]')).toBeNull();
  expect(document.activeElement).toBe(host!.querySelector('.phone-app__ask'));
});

it('renders only the original desktop route above the phone query', async () => {
  await renderFrame(1280, <FrameHarness />);
  expect(host!.textContent).toContain('Desktop route content');
  expect(host!.querySelector('.phone-app')).toBeNull();
  expect(host!.querySelector('.phone-app__data')).toBeNull();
});

it('renders only supplied badges and a source-backed live indicator', async () => {
  await renderFrame(430, <PhoneAppFrame activeTab="today" leagues={[league]}
    selectedLeagueId={league.id} badges={{ today: 2 }} live onSelectTab={() => {}}
    onSelectLeague={() => {}}><p>Desktop</p></PhoneAppFrame>);
  expect(host!.querySelector('.phone-app__badge')?.textContent).toBe('2');
  expect(host!.querySelector('.phone-app__live-dot')?.getAttribute('aria-label')).toBe('Games live');
});
