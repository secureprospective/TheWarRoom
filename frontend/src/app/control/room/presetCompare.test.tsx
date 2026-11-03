import { describe, expect, it } from 'vitest';
import { all, button, click, control, present, settle, type } from '../../commissioner/commissionerDom';
import { receipt } from '../../commissioner/commissionerStub';
import { league, room, roomHooks, tab } from './roomTestKit';

roomHooks();

const presets = [
  { id: 'default_32', label: '32-Team Default', description: '', suggestedTeams: 32 },
  { id: 'legacy_nfl', label: 'Legacy NFL', description: '', suggestedTeams: 32 },
];
const comparison = (differences: unknown[]) => ({
  league, preset: 'default_32', presetLabel: '32-Team Default', revision: 3, presets, differences,
  summary: differences.length === 0 ? 'This league already matches the 32-Team Default on every setting.'
    : `${differences.length} settings differ from the 32-Team Default.`,
});
const differences = [
  { field: 'roster.limits', label: 'Active roster size', section: 'Roster', league: '18 to 80',
    preset: '18 to 44', value: { min: 18, max: 44 } },
  { field: 'taxi.basis', label: 'Taxi experience measure', section: 'Taxi squad', league: 'Calendar years',
    preset: 'NFL seasons', value: 'nfl_seasons' },
];
const change = (field: string, value: unknown, sent: { reason: string; effective: string; request: string }) => ({
  league, command: 'commish.rules.set', args: { field, value }, reason: sent.reason, expected: 3,
  request: `${sent.request}.${field}`, effective: sent.effective, visibility: 'league',
});

describe('Control Room · Compare with a preset', () => {
  it('lists the differences, applies the chosen ones as one rules change, after the ordinary preview', async () => {
    const applied: Record<string, unknown>[] = [];
    const previewed: string[] = [];
    const { host } = await room({
      TargetCommissionerPresetCompare: async () => comparison(differences),
      TargetCommissionerPresetApply: async body => {
        const sent = JSON.parse(body) as { fields: string[]; dryRun: boolean; reason: string; effective: string;
          request: string };
        applied.push(sent);
        const changes = differences.filter(d => sent.fields.includes(d.field))
          .map(d => change(d.field, d.value, sent));
        return { changes, receipts: sent.dryRun ? [] : changes.map(() => receipt()) };
      },
      TargetCommissionerPreview: async body => {
        const sent = JSON.parse(body) as { args: { field: string }; expected: number };
        previewed.push(sent.args.field);
        return { question: 'Change this rule?', target: { league, name: 'Rules league', version: sent.expected },
          fields: [{ label: sent.args.field, before: 'was', beforeSource: 's', after: 'will be', afterSource: 's',
            when: 'Now' }], affects: { readings: [], count: 0, words: 'No franchise' }, blockers: [],
          safeguard: 'reason' };
      },
    });
    await tab(host, 'RULES');
    await click(button(host, 'COMPARE WITH A PRESET ›'));
    await settle();
    expect(present(host, '2 settings differ from the 32-Team Default.')).toBe(true);
    const row = all(host, 'tr').find(t => t.getAttribute('data-preset-field') === 'roster.limits')!;
    expect(row.textContent).toContain('Active roster size');
    expect(row.textContent).toContain('18 to 80');
    expect(row.textContent).toContain('18 to 44');
    await click(button(host, 'Skip Taxi experience measure'));
    await click(button(host, 'Now'));
    await type(control(host, 'Reason'), 'Adopt the default roster size');
    await click(button(host, 'Preview 1 rule change'));
    await settle();
    expect(previewed).toEqual(['roster.limits']);
    expect(present(host, 'Needs a typed reason before it can be recorded.')).toBe(true);
    await click(button(host, 'Record these rule changes'));
    await settle();
    const recorded = applied.filter(a => a.dryRun === false);
    expect(recorded).toHaveLength(1);
    expect(recorded[0]).toMatchObject({ league, preset: 'default_32', fields: ['roster.limits'], expected: 3,
      effective: 'now', reason: 'Adopt the default roster size' });
    expect(recorded[0]!.request).toBe(applied[0]!.request);
    expect(present(host, 'Recorded 1 rule change from the 32-Team Default.')).toBe(true);
  });

  it('says so when the league already matches the preset', async () => {
    const { host } = await room({ TargetCommissionerPresetCompare: async () => comparison([]) });
    await tab(host, 'RULES');
    await click(button(host, 'COMPARE WITH A PRESET ›'));
    await settle();
    expect(present(host, 'This league already matches the 32-Team Default on every setting.')).toBe(true);
    expect(present(host, 'Record these rule changes')).toBe(false);
  });

  it('shows Go\'s refusal word for word and records nothing', async () => {
    let recorded = 0;
    const { host } = await room({
      TargetCommissionerPresetCompare: async () => comparison(differences),
      TargetCommissionerPresetApply: async body => {
        const sent = JSON.parse(body) as { dryRun: boolean };
        if (!sent.dryRun) recorded++;
        throw new Error('rule change refused: player pools can\'t change once the season has games or player moves');
      },
    });
    await tab(host, 'RULES');
    await click(button(host, 'COMPARE WITH A PRESET ›'));
    await settle();
    await click(button(host, 'Preview 2 rule changes'));
    await settle();
    expect(present(host, 'player pools can\'t change once the season has games')).toBe(true);
    expect(recorded).toBe(0);
  });
});
