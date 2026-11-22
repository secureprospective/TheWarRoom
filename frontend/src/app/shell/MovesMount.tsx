import { lazy, Suspense } from 'react';
import type { Snapshot } from '../data/contract';
import type { PlayerSubject } from './state';

let playerAct: typeof import('./PlayerAct').default | undefined;
let myMoves: typeof import('./MyMoves').default | undefined;
const loadPlayerAct = () => import('./PlayerAct').then((module) => {
  playerAct = module.default;
  return module;
});
const loadMyMoves = () => import('./MyMoves').then((module) => {
  myMoves = module.default;
  return module;
});
const PlayerAct = lazy(loadPlayerAct);
const MyMoves = lazy(loadMyMoves);

export function createMovesPrefetch(
  load: () => Promise<unknown> = () => Promise.all([loadPlayerAct(), loadMyMoves()]),
) {
  let scheduled = false;
  return () => {
    if (scheduled) return;
    scheduled = true;
    const prefetch = () => {
      return load().catch((cause) => {
        console.error('Moves prefetch failed:', cause);
      });
    };
    if (typeof requestIdleCallback === 'function') requestIdleCallback(prefetch);
    else setTimeout(prefetch, 0);
  };
}

export function MovesMount({ snapshot }: { snapshot: Snapshot }) {
  const View = myMoves ?? MyMoves;
  return (
    <Suspense fallback={<p role="status">Opening My moves…</p>}>
      <View snapshot={snapshot} />
    </Suspense>
  );
}

export function PlayerActMount({ subject }: { subject: PlayerSubject }) {
  const View = playerAct ?? PlayerAct;
  return (
    <Suspense fallback={<p role="status">Act loading…</p>}>
      <View subject={subject} />
    </Suspense>
  );
}
