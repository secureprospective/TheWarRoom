import { expectTypeOf, it } from 'vitest';
import type { CardProps, VerdictSlot } from './Card';
import type { LabelledSignal } from '../look/channels';

it('forbids free frame styling and unlabelled or speculative slots at compile time', () => {
  expectTypeOf<CardProps>().not.toHaveProperty('className');
  expectTypeOf<CardProps>().not.toHaveProperty('style');
  expectTypeOf<{ signal: 'red' }>().not.toMatchTypeOf<LabelledSignal>();
  expectTypeOf<{ signal: 'green'; label: 'Buy' }>().not.toMatchTypeOf<VerdictSlot>();
});
