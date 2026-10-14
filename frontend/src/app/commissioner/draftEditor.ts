import type { Edit } from './ruleDraft';

// draftEditor connects the add and remove verbs to the rule form that is open. The form registers itself
// while mounted; a verb sent with no form open does nothing.
type Args = { action: 'add' | 'remove'; path: string; index: number };
export const draftEditor = {
  handler: undefined as ((edit: Edit) => void) | undefined,
  run(args: Args) {
    const path: unknown = JSON.parse(args.path);
    if (!Array.isArray(path) || path.some(s => typeof s !== 'string' && typeof s !== 'number'))
      throw new Error('Edit a rule value: invalid path');
    if (args.action !== 'add' && args.action !== 'remove')
      throw new Error('Edit a rule value: invalid action');
    this.handler?.({ action: args.action, path: path as Edit['path'], index: args.index });
  },
};
