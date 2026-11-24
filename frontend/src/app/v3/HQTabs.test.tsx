import { renderToStaticMarkup } from 'react-dom/server';
import { expect, it } from 'vitest';
import { HQTabs } from './HQLineupV3';

const tabs = (mflMoves?: boolean) => renderToStaticMarkup(<HQTabs tab="lineup-and-roster" navigate={() => {}}
  badge={null} kind="OWNER" moves={0} mflMoves={mflMoves} />);

it('shows My MFL moves only where the screen asks for it', () => {
  expect(tabs()).not.toContain('My MFL moves');
  expect(tabs(false)).not.toContain('My MFL moves');
  expect(tabs(true)).toContain('My MFL moves');
});
