import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import { posix } from 'node:path';
import { instructionPatterns, metadata, checkReviewer } from './check-governance.mjs';

const root = new URL('../../', import.meta.url);
const read = path => readFileSync(new URL(path, root), 'utf8');
const instructions = name => `.github/instructions/${name}.instructions.md`;
const names = ['go', 'python', 'web', 'coin-workflows', 'delivery'];
const scopes = new Map(names.map(name => [name, instructionPatterns(read(instructions(name)))]));

test('native instruction scopes cover intended paths without leaking into unrelated layers', () => {
  // These are scope examples, not a replacement for fresh-client discovery.
  for (const [path, expected] of [
    ['src/api/main.go', ['go', 'coin-workflows']],
    ['src/web/src/views/CoinDetailView.vue', ['web', 'coin-workflows']],
    ['src/agent/app/main.py', ['python']],
    ['src/agent/Dockerfile', ['python', 'delivery']],
    ['Dockerfile', ['go', 'web', 'delivery']],
    ['.github/skills/python-locking/SKILL.md', ['delivery']],
    ['scripts/delivery/check-governance.mjs', ['delivery']],
    ['docs/agentic-native-integration.md', ['delivery']],
    ['specs/363-collector-curator-watchlist-provenance/spec.md', []],
    ['docs/features.md', []],
  ]) {
    assert.deepEqual([...scopes].filter(([, patterns]) => patterns.some(pattern =>
      posix.matchesGlob(path, pattern))).map(([name]) => name), expected, path);
  }
});

function migrationContract(inventory, load) {
  assert.match(inventory.baseline_commit, /^[a-f0-9]{40}$/);
  assert.equal(inventory.entries.length, 20);
  assert.equal(new Set(inventory.entries.map(entry => entry.old_path)).size, 20);
  assert.equal(new Set(inventory.entries.map(entry => entry.native_path)).size, 19);
  for (const entry of inventory.entries) {
    assert.match(entry.old_path, /^\.squad\/skills\/[a-z0-9-]+(?:\/SKILL)?\.md$/);
    assert.match(entry.native_path, /^\.github\/skills\/[a-z0-9-]+\/SKILL\.md$/);
    assert.match(entry.original_blob, /^[a-f0-9]{40}$/);
    const wrapper = load(entry.old_path);
    const target = posix.relative(posix.dirname(entry.old_path), entry.native_path);
    assert.ok(wrapper.includes(`](${target})`), entry.old_path);
    assert.ok(wrapper.includes(`/blob/${inventory.baseline_commit}/${entry.old_path}`), entry.old_path);
    assert.ok(wrapper.trim().split(/\r?\n/).length <= 10, 'Legacy entries must be pointers, not duplicate recipes');
    const native = metadata(load(entry.native_path), ['name', 'description']);
    assert.equal(native.name, posix.basename(posix.dirname(entry.native_path)));
  }
}
const inventory = JSON.parse(read('.squad/artifacts/native-skill-migration-2026-09-21.json'));

test('all legacy skill entry points resolve to canonical native packages with provenance', () => {
  migrationContract(inventory, read);
  const nativeNames = new Set(readdirSync(new URL('.github/skills/', root), { withFileTypes: true })
    .filter(entry => entry.isDirectory()).map(entry => entry.name));
  for (const entry of inventory.entries) assert.ok(nativeNames.has(posix.basename(posix.dirname(entry.native_path))));
  assert.ok(!nativeNames.has('npm-audit-transitive-override'));
  assert.ok(!nativeNames.has('post-major-work-qc-audit'), 'Do not shadow the personal audit skill');
});

test('migration assertions reject omitted records, broken links and duplicate policy bodies', () => {
  assert.throws(() => migrationContract({ ...inventory, entries: inventory.entries.slice(1) }, read));
  const old = inventory.entries[0].old_path;
  for (const replacement of ['# Missing pointer\n', `${read(old)}\n${'Duplicate policy\n'.repeat(12)}`]) {
    assert.throws(() => migrationContract(inventory, path => path === old ? replacement : read(path)));
  }
  migrationContract(inventory, read);
});

test('reviewer rejects each capability expansion and universal guidance stays bounded', () => {
  const profile = read('.github/agents/aurearia-reviewer.agent.md');
  checkReviewer(profile);
  for (const tool of ['execute', 'edit', 'agent', '*', 'example-mcp/*']) {
    assert.throws(() => checkReviewer(profile.replace('["read", "search"]', JSON.stringify(['read', 'search', tool]))));
  }
  assert.throws(() => checkReviewer(profile.replace('tools: ["read", "search"]', 'tools: ["read", "read"]')));
  const universal = read('.github/copilot-instructions.md');
  assert.ok(universal.trimEnd().split(/\r?\n/).length <= 150);
  assert.doesNotMatch(universal, /(?:^|\s)@(?:\.\/)?(?:\.github\/)?instructions\//m);
  assert.match(read('.squad/routing.md'), /aurearia-reviewer\.agent\.md/);
  assert.match(read('.github/agents/squad.agent.md'), /aurearia-reviewer\.agent\.md/);
  assert.match(read('Taskfile.yml'), /copilot --agent aurearia-reviewer --excluded-tools sql skill \{\{\.CLI_ARGS\}\}/);
});
