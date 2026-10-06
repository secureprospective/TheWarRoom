import type { ShellState } from './state';

type Source = {
  read(): ShellState;
  subscribe(listener: (state: ShellState, previous: ShellState) => void): () => void;
};
type Root = Pick<HTMLElement, 'setAttribute' | 'querySelectorAll'>;

// CSS owns density; React's render projection deliberately excludes it.
export function connectDensity(source: Source, root: Root) {
  const apply = (state: ShellState) => {
    root.setAttribute('data-density', state.density);
    for (const button of root.querySelectorAll('.act-density[data-d]')) {
      button.setAttribute('aria-pressed', String(button.getAttribute('data-d') === state.density));
    }
  };
  apply(source.read());
  return source.subscribe((state, previous) => {
    if (state.density !== previous.density) apply(state);
  });
}
