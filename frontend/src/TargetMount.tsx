import { TargetApp } from './app/shell/TargetApp';
import { useShellBindings } from './app/shell/useShellBindings';
export function TargetMount() {
  useShellBindings();
  return <TargetApp />;
}
