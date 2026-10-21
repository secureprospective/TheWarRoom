#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
node <<'JS'
const fs = require('node:fs');
const path = require('node:path');
const ts = require('./frontend/node_modules/typescript');
const root = path.resolve('frontend/src/app');
const features = new Set(fs.readdirSync('frontend/wailsjs/go', { withFileTypes: true })
  .filter(entry => entry.isDirectory() && entry.name !== 'main').map(entry => entry.name));
// Go package -> frontend/src/app folder holding that feature's only adapter (api.ts).
// One line per moved feature, alphabetical, so parallel split lanes merge cleanly.
const folders = new Map([
  ['archive', 'archive'],
  ['comms', 'comms'],
  ['counterparties', 'counterparties'],
  ['draft', 'draftreports'],
  ['sources', 'datasources'],
]);
// Folders whose feature has no root App bindings left: nothing in them may touch main/App or runtime.
const whole = ['archive', 'comms', 'counterparties'];
const moved = new Map([
  ['TargetArchive', 'archive'],
  ['TargetComms', 'comms'], ['TargetCommsRefresh', 'comms'], ['TargetOpenComms', 'comms'],
  ['TargetCounterparties', 'counterparties'], ['TargetCounterpartTeam', 'counterparties'],
  ...['TargetDraftLab', 'TargetDraftReports', 'TargetDraftReportsRefresh', 'TargetNFLDraft',
    'TargetOpenMFLDraftRoom'].map(name => [name, 'draft']),
  ...['GetCrosswalkReport', 'LoadCrosswalk', 'SetLeagueArchiveDir', 'TargetCrosswalk',
    'TargetDataRefresh', 'TargetDataSources'].map(name => [name, 'sources']),
]);
let failures = 0;
for (const feature of features) {
  if (!folders.has(feature)) {
    console.error(`check-frontend-transport: unmapped generated Go package ${feature}`);
    failures++;
  }
}
function files(dir) {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const file = path.join(dir, entry.name);
    return entry.isDirectory() ? files(file) : /\.tsx?$/.test(file) ? [file] : [];
  });
}
function chain(node) {
  if (ts.isParenthesizedExpression(node) || ts.isAsExpression(node) ||
      ts.isTypeAssertionExpression(node) || ts.isNonNullExpression(node)) return chain(node.expression);
  if (ts.isPropertyAccessExpression(node)) return [...chain(node.expression), node.name.text];
  if (ts.isElementAccessExpression(node) && ts.isStringLiteral(node.argumentExpression))
    return [...chain(node.expression), node.argumentExpression.text];
  if (ts.isQualifiedName(node)) return [...chain(node.left), node.right.text];
  return ts.isIdentifier(node) ? [node.text] : [];
}
for (const file of files(path.resolve('frontend/src'))) {
  const source = ts.createSourceFile(file, fs.readFileSync(file, 'utf8'), ts.ScriptTarget.Latest, true);
  const modelAliases = new Set();
  const rootAliases = new Set();
  const reported = new Set();
  function check(feature, node, reason, allowAdapter = true) {
    if (allowAdapter && file === path.join(root, folders.get(feature) || feature, 'api.ts')) return;
    const { line } = source.getLineAndCharacterOfPosition(node.getStart(source));
    const key = `${feature}:${line}`;
    if (reported.has(key)) return;
    reported.add(key);
    console.error(`check-frontend-transport: ${path.relative(process.cwd(), file)}:${line + 1}: ${reason}; use ${feature}/api.ts`);
    failures++;
  }
  const ownFeature = whole.find(feature => file.startsWith(path.join(root, folders.get(feature)) + path.sep));
  function moduleAccess(spec, node) {
    if (/wailsjs\/go\/main\/App$/.test(spec)) {
      if (ts.isImportDeclaration(node) && node.importClause?.namedBindings &&
          ts.isNamedImports(node.importClause.namedBindings)) {
        for (const name of node.importClause.namedBindings.elements) {
          const method = (name.propertyName || name.name).text;
          if (moved.has(method)) check(moved.get(method), name, 'moved root binding import', false);
        }
      } else if (ts.isImportDeclaration(node) && node.importClause?.namedBindings &&
                 ts.isNamespaceImport(node.importClause.namedBindings)) {
        rootAliases.add(node.importClause.namedBindings.name.text);
      } else if (ts.isExportDeclaration(node) && node.exportClause && ts.isNamedExports(node.exportClause)) {
        for (const name of node.exportClause.elements) {
          const method = (name.propertyName || name.name).text;
          if (moved.has(method)) check(moved.get(method), name, 'moved root binding export', false);
        }
      }
      // Other features retain their legacy namespace/dynamic root imports.
    }
    const match = spec.match(/wailsjs\/go\/([^/]+)\//);
    if (match && match[1] !== 'main') check(match[1], node, 'generated feature transport outside its adapter');
    if (ownFeature &&
        /wailsjs\/(?:go\/main\/App|runtime\/)/.test(spec))
      check(ownFeature, node, 'Backend access outside its feature adapter', !/go\/main\/App/.test(spec));
  }
  // Discover model namespace aliases before inspecting accesses, including multiline imports.
  for (const statement of source.statements) {
    if (!ts.isImportDeclaration(statement) || !ts.isStringLiteral(statement.moduleSpecifier)) continue;
    moduleAccess(statement.moduleSpecifier.text, statement);
    if (!/wailsjs\/go\/models$/.test(statement.moduleSpecifier.text)) continue;
    const bindings = statement.importClause?.namedBindings;
    if (bindings && ts.isNamespaceImport(bindings)) modelAliases.add(bindings.name.text);
    if (bindings && ts.isNamedImports(bindings)) for (const name of bindings.elements) {
      const feature = (name.propertyName || name.name).text;
      if (features.has(feature)) check(feature, name, 'generated feature models outside its adapter');
    }
  }
  function visit(node) {
    if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) {
      if (node.moduleSpecifier && ts.isStringLiteral(node.moduleSpecifier)) moduleAccess(node.moduleSpecifier.text, node);
    }
    if (ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword &&
        node.arguments[0] && ts.isStringLiteral(node.arguments[0])) moduleAccess(node.arguments[0].text, node);
    if (ts.isImportTypeNode(node) && ts.isLiteralTypeNode(node.argument) && ts.isStringLiteral(node.argument.literal))
      moduleAccess(node.argument.literal.text, node);
    const parts = chain(node);
    if (rootAliases.has(parts[0]) && moved.has(parts[1]))
      check(moved.get(parts[1]), node, 'moved root binding namespace access', false);
    const at = parts.indexOf('window');
    if (at >= 0 && parts[at + 1] === 'go' && parts[at + 2]) {
      const feature = parts[at + 2];
      if (feature !== 'main') check(feature, node, 'window.go feature access outside its adapter');
      else if (parts[at + 3] === 'App' && moved.has(parts[at + 4]))
        check(moved.get(parts[at + 4]), node, 'moved root binding access', false);
      else if (ownFeature) check(ownFeature, node, 'Feature still accesses the root App', false);
    }
    if (modelAliases.has(parts[0]) && features.has(parts[1])) check(parts[1], node, 'generated feature models outside its adapter');
    ts.forEachChild(node, visit);
  }
  visit(source);
}
if (failures) process.exit(1);
console.log('check-frontend-transport: feature transport boundaries hold');
JS
