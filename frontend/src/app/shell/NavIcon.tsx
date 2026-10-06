import type { Node } from './nodes';
const icons = {
  home: <><path d="M2 7.5 8 2.5l6 5V14H2z"/><path d="M6.5 14v-4h3v4"/></>,
  war: <><circle cx="8" cy="8" r="5.5"/><path d="M8 1v3M8 12v3M1 8h3M12 8h3"/></>,
  hq: <><path d="M8 1.5 13.5 3.5v4c0 3.5-2.5 6-5.5 7-3-1-5.5-3.5-5.5-7v-4z"/></>,
  trade: <><path d="M2 5h11l-3-3M14 11H3l3 3"/></>,
  pulse: <><path d="M1 8h3l2-5 3 10 2-5h4"/></>,
  control: <><path d="M3 2v12M8 2v12M13 2v12"/><path d="M1.5 5h3M6.5 10h3M11.5 6h3"/></>,
};
export function NavIcon({ node }: { node: Node }) {
  return <svg viewBox="0 0 16 16" aria-hidden="true">{icons[node]}</svg>;
}
