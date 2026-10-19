import { Act } from '../commands/Act';

// TrayLauncher is the only thing a commissioner screen carries for a change: a control that opens the
// change in the side tray. The form, its preview and its confirm live in the tray, never beside the list.
// `label` is the full name a screen reader hears; `text` is what the control shows.
export function TrayLauncher({ tray, label, text = label }: { tray: string; label: string; text?: string }) {
  return (
    <Act verb="commissioner.surface" variant="button" label={label}
      args={{ screen: 'landing', action: 'tray', value: tray }}>{text}</Act>
  );
}
