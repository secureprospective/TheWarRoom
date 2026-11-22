export function createMFLKeyField() {
  let input: HTMLInputElement | null = null;
  let focusOnMount = false;
  const listeners = new Set<() => void>();
  const notify = () => listeners.forEach((listener) => listener());
  return {
    mount: (field: HTMLInputElement | null) => {
      if (input) {
        input.removeEventListener('input', notify);
        input.value = '';
      }
      input = field;
      input?.addEventListener('input', notify);
      if (input && focusOnMount) {
        input.focus();
        focusOnMount = false;
      }
      notify();
    },
    empty: () => !input?.value,
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => { listeners.delete(listener); };
    },
    take: (): string => {
      const text = input?.value ?? '';
      if (input) input.value = '';
      notify();
      return text;
    },
    focus: () => {
      focusOnMount = !input;
      input?.focus();
    },
  };
}
