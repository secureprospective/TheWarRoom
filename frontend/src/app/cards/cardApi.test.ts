import { expectTypeOf, it } from 'vitest';
import type { CardProps, VerdictSlot } from './Card';
import type { PlayerCardProps } from './PlayerCard';
import type { LabelledSignal } from '../look/channels';

it('forbids free frame styling and unlabelled or speculative slots at compile time', () => {
  expectTypeOf<CardProps>().not.toHaveProperty('className');
  expectTypeOf<CardProps>().not.toHaveProperty('style');
  expectTypeOf<{ signal: 'red' }>().not.toMatchTypeOf<LabelledSignal>();
  expectTypeOf<{ signal: 'green'; label: 'Buy' }>().not.toMatchTypeOf<VerdictSlot>();
});

it('forbids density subscriptions through card props', () => {
  expectTypeOf<CardProps>().not.toHaveProperty('density');
  expectTypeOf<PlayerCardProps>().not.toHaveProperty('density');
  // These assignments must stop compiling if density returns to either API.
  const card: 'density' extends keyof CardProps ? false : true = true;
  const player: 'density' extends keyof PlayerCardProps ? false : true = true;
  expectTypeOf(card).toEqualTypeOf<true>();
  expectTypeOf(player).toEqualTypeOf<true>();
});
