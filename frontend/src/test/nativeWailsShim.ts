// Component tests default to the desktop transport; browser transport tests remove this marker.
if (typeof window !== 'undefined' && !('go' in window)) {
  Object.defineProperty(window, 'go', { configurable: true, value: {} });
}
export {};
