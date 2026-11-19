import type { ReactElement, ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it, vi } from 'vitest';
import { parsePeople } from '../../commissioner/people';
import { EffectivePermissions } from './PeopleView';
import { openTeam } from './openers';
import type { Room } from './room';

const permissions = ['approve-stat-fixes', 'rule-on-trades', 'edit-lineups', 'move-picks',
  'change-rules', 'run-rollover'] as const;

function peopleResponse() {
  return parsePeople({
    league: 'native.permissions', name: 'Permissions', revision: 4,
    seats: [{ franchise: 'franchise.01', name: 'Vikings', abbreviation: 'MIN', owner: 'acct.owner' }],
    roles: [{ person: 'acct.owner', role: 'commissioner' }], invites: [], dotCount: 0,
    dotStatus: 'Ready', votes: [], succession: 'Commissioner active',
    actors: [{ person: 'acct.owner', effectivePermissions: [...permissions] }],
    yourPermissions: [...permissions], yourRole: 'commissioner', yourFranchise: 'franchise.01',
    mayChangeRules: true, mayGrantRoles: true, mayPostNotices: true,
  });
}

function markup(node: ReactNode): string {
  if (!node) throw new Error('Expected an inspector node');
  return renderToStaticMarkup(node as ReactElement);
}

it('renders effective permissions from the server People response in both Control Room views', () => {
  const people = peopleResponse();
  const panel = markup(<EffectivePermissions people={people} />);
  expect(panel).toContain('Vikings');
  expect(panel).toContain('approve-stat-fixes');
  expect(panel).toContain('run-rollover');

  const inspect = vi.fn();
  const room = { state: { people }, data: { errors: {}, signIns: [{
    principal: 'acct.owner', authenticator: true, lastSeen: '2026-11-19T10:00:00Z',
  }] }, inspect } as unknown as Room;
  openTeam(room, 'franchise.01');
  const team = markup(inspect.mock.calls[0]?.[0]);
  expect(team).toContain('Effective deputy permissions');
  expect(team).toContain('change-rules');
  expect(team).toContain('Authenticator');
  expect(team).toContain('Enabled');
});

it('keeps the placeholder when the People response omits private actor permissions', () => {
  const people = { ...peopleResponse(), actors: [] };
  const panel = markup(<EffectivePermissions people={people} />);
  expect(panel).toContain('Not available yet');
  expect(panel).toContain('server does not include actor permissions for this role');
});

it('keeps authenticator status unavailable when the local sign-in read has no matching fact', () => {
  const inspect = vi.fn();
  const room = { state: { people: peopleResponse() }, data: { errors: { signIns: 'sign-in read failed' } },
    inspect } as unknown as Room;
  openTeam(room, 'franchise.01');
  const team = markup(inspect.mock.calls[0]?.[0]);
  expect(team).toContain('Not available yet');
  expect(team).toContain('sign-in read failed');
});
