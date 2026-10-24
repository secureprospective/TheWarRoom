import { readdirSync } from 'node:fs';
import { resolve, relative } from 'node:path';
import { pathToFileURL } from 'node:url';

export function caseConflicts(paths) {
  const seen = new Map();
  const conflicts = [];
  for (const path of paths) {
    // TypeScript also folds module stems across .ts and .tsx.
    const key = path.replace(/\.tsx?$/, '.ts').toLowerCase();
    if (seen.has(key) && seen.get(key) !== path) conflicts.push([seen.get(key), path]);
    else seen.set(key, path);
  }
  return conflicts;
}

function pathsUnder(root, dir = root) {
  return readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const path = resolve(dir, entry.name);
    return [relative(root, path), ...(entry.isDirectory() ? pathsUnder(root, path) : [])];
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const conflicts = caseConflicts(pathsUnder(resolve('src')));
  if (conflicts.length) {
    console.error('Case-insensitive source paths collide:', conflicts);
    process.exitCode = 1;
  }
}
