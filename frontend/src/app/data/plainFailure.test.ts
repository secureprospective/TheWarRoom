import { expect, it } from 'vitest';
import { ADVICE, plainFailure } from './plainFailure';

it('reduces an offline failure to its plain sentence', () => {
  expect(plainFailure(`Error: site news: Can't reach MFL.${ADVICE}`)).toBe(`Can't reach MFL.${ADVICE}`);
  expect(plainFailure('Error: League unavailable')).toBe('League unavailable');
});
