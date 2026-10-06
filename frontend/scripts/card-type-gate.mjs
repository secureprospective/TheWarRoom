import ts from 'typescript';
import assert from 'node:assert/strict';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const guard = resolve(root, 'src/app/cards/.type-proof.ts');
const source = `
import type { CardProps } from './Card';
import type { PlayerCardProps } from './PlayerCard';
const card: 'density' extends keyof CardProps ? false : true = true;
const player: 'density' extends keyof PlayerCardProps ? false : true = true;
`;
const config = ts.readConfigFile(resolve(root, 'tsconfig.json'), ts.sys.readFile);
const options = ts.parseJsonConfigFileContent(config.config, ts.sys, root).options;
function check(mutant) {
  const host = ts.createCompilerHost(options);
  const original = host.getSourceFile.bind(host);
  host.getSourceFile = (file, language, ...rest) => {
    if (file === guard) return ts.createSourceFile(file, source, language);
    const tree = original(file, language, ...rest);
    if (mutant && file === resolve(root, `src/app/cards/${mutant}.tsx`)) {
      return ts.createSourceFile(file, tree.text.replace('gravity?:', 'density: string; gravity?:')
        .replace('gravity: Gravity;', 'density: string; gravity: Gravity;'), language);
    }
    return tree;
  };
  const program = ts.createProgram([guard], options, host);
  return ts.getPreEmitDiagnostics(program).filter((d) => d.file?.fileName === guard);
}
assert.equal(check().length, 0);
for (const mutant of ['Card', 'PlayerCard']) {
  assert.ok(check(mutant).some((d) => d.code === 2322));
  console.log(`card-type-gate: compiler rejected density on ${mutant}Props`);
}
