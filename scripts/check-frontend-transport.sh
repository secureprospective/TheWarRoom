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
let failures = 0;
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
  const reported = new Set();
  function check(feature, node, reason, allowAdapter = true) {
    if (allowAdapter && file === path.join(root, feature, 'api.ts')) return;
    const { line } = source.getLineAndCharacterOfPosition(node.getStart(source));
    const key = `${feature}:${line}`;
    if (reported.has(key)) return;
    reported.add(key);
    console.error(`check-frontend-transport: ${path.relative(process.cwd(), file)}:${line + 1}: ${reason}; use ${feature}/api.ts`);
    failures++;
  }
  function moduleAccess(spec, node) {
    const match = spec.match(/wailsjs\/go\/([^/]+)\//);
    if (match && match[1] !== 'main') check(match[1], node, 'generated feature transport outside its adapter');
    const feature = ['comms', 'counterparties'].find(name => file.startsWith(path.join(root, name) + path.sep));
    if (feature &&
        /wailsjs\/(?:go\/main\/App|runtime\/)/.test(spec))
      check(feature, node, 'backend access outside its feature adapter', !/go\/main\/App/.test(spec));
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
    const at = parts.indexOf('window');
    if (at >= 0 && parts[at + 1] === 'go' && parts[at + 2]) {
      const feature = parts[at + 2];
      if (feature !== 'main') check(feature, node, 'window.go feature access outside its adapter');
      else for (const name of ['comms', 'counterparties']) {
        if (file.startsWith(path.join(root, name) + path.sep)) check(name, node, 'feature still accesses the root App', false);
      }
    }
    if (modelAliases.has(parts[0]) && features.has(parts[1])) check(parts[1], node, 'generated feature models outside its adapter');
    ts.forEachChild(node, visit);
  }
  visit(source);
}
if (failures) process.exit(1);
console.log('check-frontend-transport: feature transport boundaries hold');
JS
