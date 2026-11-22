import { RuleChangeForm } from './RuleChangeForm';
import SideTray from './SideTray';
import type { Rules } from './rules';

export default function RuleTray({ rules, field, opener, background, onClose }: {
  rules: Rules; field: string; opener: HTMLElement; background: HTMLElement | null; onClose(): void;
}) {
  return (
    <SideTray title="Change a rule" screen="landing" opener={opener} background={background}
      onClose={onClose} closeLabel="Close rule change">
      <RuleChangeForm rules={rules} initialField={field} />
    </SideTray>
  );
}
