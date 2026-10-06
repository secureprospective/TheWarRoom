import { homeRoute, isNode, routeFor, type Route } from './nodes';
export function parseRoute(hash: string): Route {
  const parts = /^#\/([^/]+)\/([^/]+)$/.exec(hash);
  return parts && isNode(parts[1]) ? routeFor(parts[1], parts[2]) ?? homeRoute : homeRoute;
}
export function formatRoute(route: Route): string { return `#/${route.node}/${route.workspace}`; }
