import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, readdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { pathToFileURL, fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { activeFiles, checkGovernance, metadata } from './check-governance.mjs';

const checker = fileURLToPath(new URL('./check-governance.mjs', import.meta.url));
function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), 'delivery-governance-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const put = (file, value) => {
    const target = join(root, ...file.split('/'));
    mkdirSync(dirname(target), { recursive: true });
    writeFileSync(target, value);
  };
  for (const file of activeFiles) put(file, '# Active guidance\n');
  put('.specify/memory/constitution.md', ['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII', 'IX'].map(id => `### ${id}. Principle\n`).join(''));
  put('docs/adr/README.md', '| 0001 | [Decision](0001-test.md) | 2026-09-21 | Accepted |\n');
  put('docs/adr/0001-test.md', '# Decision\n\nStatus: Accepted\n\n## Context\nHistorical Principle XIV.\n');
  put('docs/work.md', '# Authorized process work\n');
  put('.squad/identity/now.md', '---\nupdated_at: 2026-09-21\nfocus_area: Testing\nowner: Maintainer\nwork_artifact: docs/work.md\ntasks_artifact: docs/work.md\n---\n');
  put('.github/skills/example/SKILL.md', '---\nname: example\ndescription: A bounded example skill\n---\n');
  put('.github/agents/aurearia-reviewer.agent.md', '---\nname: aurearia-reviewer\ndescription: Read-only review\ntools: ["read", "search"]\n---\n');
  return { root, put };
}
const cases = [
  ['INSTRUCTION', 'missing applyTo', f => f.put('.github/instructions/example.instructions.md', '---\ndescription: Missing scope\n---\n')],
  ['INSTRUCTION', 'invalid applyTo', f => f.put('.github/instructions/example.instructions.md', '---\napplyTo: ["src/**"]\n---\n')],
  ['INSTRUCTION', 'escaping applyTo', f => f.put('.github/instructions/example.instructions.md', '---\napplyTo: "../outside/**"\n---\n')],
  ['INSTRUCTION', 'empty applyTo pattern', f => f.put('.github/instructions/example.instructions.md', '---\napplyTo: "src/**,"\n---\n')],
  ['REVIEWER', 'omitted reviewer tools', f => f.put('.github/agents/aurearia-reviewer.agent.md', '---\nname: aurearia-reviewer\ndescription: Review\n---\n')],
  ['REVIEWER', 'write-capable reviewer', f => f.put('.github/agents/aurearia-reviewer.agent.md', '---\nname: aurearia-reviewer\ndescription: Review\ntools: ["read", "search", "edit"]\n---\n')],
  ['REVIEWER', 'reviewer model override', f => f.put('.github/agents/aurearia-reviewer.agent.md', '---\nname: aurearia-reviewer\ndescription: Review\ntools: ["read", "search"]\nmodel: override\n---\n')],
  ['FILE', 'missing active file', f => rmSync(join(f.root, 'CONTRIBUTING.md'))],
  ['PRINCIPLE', 'obsolete identifier', f => f.put('CONTRIBUTING.md', 'Apply Principle XI.\n')],
  ['PRINCIPLE', 'obsolete identifier in a list', f => f.put('CONTRIBUTING.md', 'Principles I, XIV and IX apply.\n')],
  ['REFERENCE', 'missing linked spec', f => f.put('CONTRIBUTING.md', '[Spec](specs/999-missing/spec.md)\n')],
  ['REFERENCE', 'escaping reference', f => f.put('CONTRIBUTING.md', '[Bad](../outside.md)\n')],
  ['REFERENCE', 'bad reference definition', f => f.put('CONTRIBUTING.md', '[guide]: docs/missing.md\n')],
  ['ADR', 'status mismatch', f => f.put('docs/adr/0001-test.md', 'Status: Proposed\n\n## Context\n')],
  ['ADR', 'invalid status', f => f.put('docs/adr/0001-test.md', 'Status: Probably approved\n\n## Context\n')],
  ['ADR', 'wrong index target', f => f.put('docs/adr/README.md', '| 0001 | [Decision](0002-missing.md) | date | Accepted |\n')],
  ['ADR', 'missing status target', f => f.put('docs/adr/0001-test.md', 'Status: Superseded by 9999\n\n## Context\n')],
  ['STATE', 'malformed state', f => f.put('.squad/identity/now.md', '---\nowner: unclosed\n')],
  ['STATE', 'missing selected spec', f => f.put('.squad/identity/now.md', '---\nupdated_at: 2026-09-21\nfocus_area: Missing\nowner: Maintainer\nwork_artifact: specs/999-missing/spec.md\ntasks_artifact: specs/999-missing/tasks.md\n---\n')],
  ['STATE', 'invalid date', f => f.put('.squad/identity/now.md', '---\nupdated_at: 2026-02-30\nfocus_area: Test\nowner: Maintainer\nwork_artifact: docs/work.md\ntasks_artifact: docs/work.md\n---\n')],
  ['STATE', 'duplicate metadata', f => f.put('.squad/identity/now.md', '---\nowner: one\nowner: two\n---\n')],
  ['STATE', 'noncanonical state target', f => f.put('.squad/identity/now.md', '---\nupdated_at: 2026-09-21\nfocus_area: Test\nowner: Maintainer\nwork_artifact: docs/../CONTRIBUTING.md\ntasks_artifact: docs/work.md\n---\n')],
  ['SKILL', 'invalid native metadata', f => f.put('.github/skills/example/SKILL.md', '---\nname: Wrong_Name\ndescription: example\n---\n')],
  ['SKILL', 'missing description', f => f.put('.github/skills/example/SKILL.md', '---\nname: example\n---\n')],
  ['SKILL', 'missing native entry point', f => { f.put('.github/skills/missing/README.md', '# Wrong entry point'); }],
  ['SKILL', 'non-string description', f => f.put('.github/skills/example/SKILL.md', '---\nname: example\ndescription: []\n---\n')],
  ['SKILL', 'unclosed quoted scalar', f => f.put('.github/skills/example/SKILL.md', '---\nname: example\ndescription: "unclosed\n---\n')],
];
for (const [code, name, mutate] of cases) {
  test(`rejects ${name} with ${code} diagnostic`, t => {
    const f = fixture(t);
    assert.deepEqual(checkGovernance(f.root), []);
    mutate(f);
    const errors = checkGovernance(f.root).filter(item => item.severity === 'error');
    assert.ok(errors.some(item => item.code === code), JSON.stringify(errors));
    assert.ok(errors.every(item => item.file && item.line >= 1));
  });
}
test('accepts supported metadata, Nygard status sections, and warning-only budgets', t => {
  const f = fixture(t);
  f.put('docs/adr/0001-test.md', '# ADR\n\n## Status\n\nACCEPTED\n\n## Context\nOld Principle XIV.\n');
  f.put('.github/skills/example/SKILL.md', '---\nname: "example"\ndescription: >-\n  First line\n  second line\n---\n');
  f.put('.github/copilot-instructions.md', '# Guidance\n'.repeat(151));
  assert.ok(checkGovernance(f.root).every(item => item.severity === 'warning'));
  assert.ok(checkGovernance(f.root).some(item => item.code === 'BUDGET'));
  assert.equal(metadata("---\nname: 'it''s text'\n---", ['name']).name, "it's text");
});
test('does not lint immutable archives, selected requirement bodies, examples or runtime paths', t => {
  const f = fixture(t);
  f.put('.squad/decisions-archive.md', 'Principle XX. [Missing](missing.md)\n');
  f.put('.squad/agents/example/history-archive.md', 'Principle XVII\n');
  f.put('specs/001-landed/spec.md', 'Historical Principle XII\n');
  f.put('CONTRIBUTING.md', '```text\nPrinciple XV. [Example](missing.md)\n```\nIf `.specify/extensions.yml` exists, inspect it.\nUse `src/agent/.venv/Scripts/python.exe` after setup.\n');
  f.put('.github/skills/README.md', '# Native skill index\n');
  assert.deepEqual(checkGovernance(f.root), []);
});
test('bounded issue work does not require an invented feature spec', t => {
  const f = fixture(t);
  f.put('.squad/identity/now.md', '---\nupdated_at: 2026-09-21\nfocus_area: Issue fix\nowner: Maintainer\nwork_artifact: https://github.com/briandenicola/Aurearia/issues/721\ntasks_artifact: https://github.com/briandenicola/Aurearia/issues/721\n---\n');
  assert.deepEqual(checkGovernance(f.root), []);
});
test('Proposed lifecycle and size findings warn without waiving invalid status links', t => {
  const f = fixture(t);
  f.put('docs/adr/README.md', '| 0001 | [Decision](0001-test.md) | date | Proposed |\n');
  f.put('docs/adr/0001-test.md', 'Status: Proposed\n\n## Context\n');
  f.put('.squad/decisions.md', 'x'.repeat(20 * 1024 + 1));
  f.put('.squad/agents/example/history.md', 'x'.repeat(12 * 1024 + 1));
  f.put('.squad/skills/legacy/SKILL.md', 'No native metadata; wrapper migration belongs to P4.\n');
  const diagnostics = checkGovernance(f.root);
  assert.ok(diagnostics.every(item => item.severity === 'warning'));
  assert.ok(diagnostics.some(item => item.code === 'LIFECYCLE'));
  assert.equal(diagnostics.filter(item => item.code === 'BUDGET').length, 2);
  f.put('docs/adr/README.md', '| 0001 | [Decision](0002-missing.md) | date | Proposed |\n');
  assert.ok(checkGovernance(f.root).some(item => item.code === 'ADR' && item.severity === 'error'));
});
test('sparse-excluded links warn but ordinary missing tracked files still fail', t => {
  const f = fixture(t);
  f.put('docs/sparse.md', '# Target\n');
  const git = args => {
    const result = spawnSync('git', ['-C', f.root, ...args], { encoding: 'utf8' });
    assert.equal(result.status, 0, result.stderr);
  };
  git(['init', '--quiet']);
  git(['add', 'docs/sparse.md']);
  git(['update-index', '--skip-worktree', 'docs/sparse.md']);
  rmSync(join(f.root, 'docs/sparse.md'));
  f.put('CONTRIBUTING.md', '[Target](docs/sparse.md)\n');
  assert.ok(checkGovernance(f.root).some(item => item.code === 'SPARSE' && item.severity === 'warning'));
  assert.ok(checkGovernance(f.root).every(item => item.severity !== 'error'));
  git(['update-index', '--no-skip-worktree', 'docs/sparse.md']);
  assert.ok(checkGovernance(f.root).some(item => item.code === 'REFERENCE' && item.severity === 'error'));
});
test('checker is deterministic, read-only, and has correct CLI exit statuses', t => {
  const f = fixture(t);
  const snapshot = dir => readdirSync(dir, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))
    .flatMap(entry => entry.isDirectory() ? snapshot(join(dir, entry.name)) : [[join(dir, entry.name), readFileSync(join(dir, entry.name), 'hex')]]);
  const before = snapshot(f.root);
  const first = spawnSync(process.execPath, [checker, '--root', f.root], { encoding: 'utf8' });
  const second = spawnSync(process.execPath, [checker, '--root', f.root], { encoding: 'utf8' });
  assert.equal(first.status, 0, first.stderr);
  assert.equal(first.stdout, second.stdout);
  assert.deepEqual(snapshot(f.root), before);
  f.put('CONTRIBUTING.md', 'Principle XIV.\n');
  const failed = spawnSync(process.execPath, [checker, '--root', f.root], { encoding: 'utf8' });
  assert.equal(failed.status, 1);
  assert.match(failed.stdout, /CONTRIBUTING.md:1: error PRINCIPLE/);
  const invalid = spawnSync(process.execPath, [checker, '--unknown'], { encoding: 'utf8' });
  assert.equal(invalid.status, 1);
  assert.match(invalid.stderr, /error INPUT/);
});
test('each blocking rule is load-bearing: disabling its diagnostic breaks its fixture assertion', async t => {
  const source = readFileSync(checker, 'utf8');
  for (const code of ['FILE', 'PRINCIPLE', 'REFERENCE', 'ADR', 'STATE', 'SKILL', 'INSTRUCTION', 'REVIEWER']) {
    const f = fixture(t);
    cases.find(item => item[0] === code)[2](f);
    const contract = implementation => assert.ok(implementation(f.root).some(item => item.code === code && item.severity === 'error'), code);
    contract(checkGovernance);
    const mutant = source.replace('diagnostics.push({ file, line, severity, code, message });',
      `code === '${code}' ? undefined : diagnostics.push({ file, line, severity, code, message });`);
    assert.notEqual(mutant, source);
    const file = join(f.root, `mutant-${code}.mjs`);
    writeFileSync(file, mutant);
    const changed = await import(pathToFileURL(file).href);
    assert.throws(() => contract(changed.checkGovernance), assert.AssertionError);
    contract(checkGovernance);
  }
});
