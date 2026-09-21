import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, copyFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { tmpdir } from 'node:os';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { generate, versionAnnotation } from './openapi.mjs';

const root = fileURLToPath(new URL('../..', import.meta.url));
function fixture(t) {
  const dir = mkdtempSync(join(tmpdir(), 'delivery-validation-'));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  const put = (file, content) => {
    const target = join(dir, ...file.split('/'));
    mkdirSync(dirname(target), { recursive: true });
    writeFileSync(target, content);
  };
  copyFileSync(join(root, 'Taskfile.yml'), join(dir, 'Taskfile.yml'));
  for (const path of ['src/api', 'src/web', 'src/agent/.venv']) mkdirSync(join(dir, path), { recursive: true });
  put('src/agent/.venv/pyvenv.cfg', '# Dry-run fixture only; never used as an environment\n');
  return { dir, put };
}
function task(dir, args, env = {}) {
  const result = spawnSync('task', ['--dir', dir, ...args], {
    encoding: 'utf8', env: { ...process.env, ...env }, timeout: 30000,
  });
  if (result.error) throw result.error;
  return { ...result, output: result.stdout + result.stderr };
}
test('Task expands all required commands without implicit setup', t => {
  const f = fixture(t);
  const expected = {
    'check:go': ['go build ./...', 'go vet ./...', 'go test -v ./...'].map(cmd => `GOTOOLCHAIN=local GOPROXY=off ${cmd}`),
    'check:web': ['lint', 'type-check', 'test', 'build'].map(script => `${process.platform === 'win32' ? 'npm.cmd' : 'npm'} run ${script}`),
    'check:agent': ['uv sync --locked --check --offline --extra dev', 'uv run --no-sync --offline ruff check app/ tests/', 'uv run --no-sync --offline pytest tests/ -v'].map(cmd => `UV_PYTHON_DOWNLOADS=never ${cmd}`),
    'check:openapi': ['node scripts/delivery/openapi.mjs --check'],
    'test-race': ['GOTOOLCHAIN=local GOPROXY=off CGO_ENABLED=1 go test -race ./...'],
  };
  for (const [target, commands] of Object.entries(expected)) {
    const result = task(f.dir, ['--dry', target]);
    assert.equal(result.status, 0, result.output);
    const actual = result.output.split(/\r?\n/).filter(line => /^task: \[[^\]]+\] /.test(line))
      .map(line => line.replace(/^task: \[[^\]]+\] /, ''));
    assert.deepEqual(actual, commands, `${target}: shared gate command drift`);
    assert.doesNotMatch(result.output, /\b(?:install|npm ci)\b|--quiet\b|--if-present\b/);
  }
  assert.match(task(f.dir, ['--dry', 'check:web']).output, process.platform === 'win32' ? /npm\.cmd run lint/ : /npm run lint/);
});
test('missing prepared agent environment fails before invoking uv', t => {
  const f = fixture(t);
  rmSync(join(f.dir, 'src/agent/.venv/pyvenv.cfg'));
  const result = task(f.dir, ['check:agent']);
  assert.notEqual(result.status, 0);
  assert.match(result.output, /Missing prepared agent venv/);
  assert.doesNotMatch(result.output, /task: \[[^\]]+\] .*uv /);
});
test('command-local policy overrides conflicting inherited Go, race and Python settings', t => {
  const f = fixture(t);
  const probe = join(f.dir, 'environment.cjs');
  f.put('environment.cjs', 'console.log(JSON.stringify({ phase: process.argv[2], go: process.env.GOTOOLCHAIN, proxy: process.env.GOPROXY, cgo: process.env.CGO_ENABLED, python: process.env.UV_PYTHON_DOWNLOADS }));\n');
  const substitutions = [
    ['go build ./...', 'build'], ['go vet ./...', 'vet'], ['go test -v ./...', 'test'],
    ['go test -race ./...', 'race'],
    ['uv sync --locked --check --offline --extra dev', 'lock'],
    ['uv run --no-sync --offline ruff check app/ tests/', 'lint'],
    ['uv run --no-sync --offline pytest tests/ -v', 'pytest'],
  ];
  let source = readFileSync(join(f.dir, 'Taskfile.yml'), 'utf8');
  for (const [command, phase] of substitutions) {
    assert.ok(source.includes(command), command);
    source = source.replace(command, `node ${JSON.stringify(probe)} ${phase}`);
  }
  f.put('Taskfile.yml', source);
  const conflict = { GOTOOLCHAIN: 'auto', GOPROXY: 'https://proxy.golang.org,direct', CGO_ENABLED: '0', UV_PYTHON_DOWNLOADS: 'always' };
  const execute = target => {
    const result = task(f.dir, [target], conflict);
    assert.equal(result.status, 0, result.output);
    return result.stdout.trim().split(/\r?\n/).map(line => JSON.parse(line));
  };
  const go = [...execute('check:go'), ...execute('test-race')];
  assert.deepEqual(go.map(item => item.phase), ['build', 'vet', 'test', 'race']);
  for (const item of go) {
    assert.equal(item.go, 'local', item.phase);
    assert.equal(item.proxy, 'off', item.phase);
  }
  assert.equal(go.find(item => item.phase === 'race').cgo, '1');
  const python = execute('check:agent');
  assert.deepEqual(python.map(item => item.phase), ['lock', 'lint', 'pytest']);
  for (const item of python) assert.equal(item.python, 'never', item.phase);

  f.put('Taskfile.yml', source.replaceAll('GOTOOLCHAIN=local GOPROXY=off ', '').replaceAll('UV_PYTHON_DOWNLOADS=never ', ''));
  assert.throws(() => assert.equal(execute('check:go')[0].go, 'local'), assert.AssertionError);
  assert.throws(() => assert.equal(execute('check:agent')[0].python, 'never'), assert.AssertionError);
});
test('real Task invocation propagates lint failure and never skips a missing linter', t => {
  const f = fixture(t);
  f.put('src/web/hook.cjs', `const fs = require('node:fs'); const phase = process.argv[2]; fs.appendFileSync('calls.txt', phase + '\\n'); if (phase === 'lint' && process.env.FAIL_LINT === '1') process.exit(17);\n`);
  const scripts = Object.fromEntries(['lint', 'type-check', 'test', 'build'].map(name => [name, `node hook.cjs ${name}`]));
  f.put('src/web/package.json', JSON.stringify({ scripts }));
  const pass = task(f.dir, ['check:web']);
  assert.equal(pass.status, 0, pass.output);
  assert.equal(readFileSync(join(f.dir, 'src/web/calls.txt'), 'utf8'), 'lint\ntype-check\ntest\nbuild\n');
  f.put('src/web/calls.txt', '');
  const failed = task(f.dir, ['check:web'], { FAIL_LINT: '1' });
  assert.notEqual(failed.status, 0);
  assert.equal(readFileSync(join(f.dir, 'src/web/calls.txt'), 'utf8'), 'lint\n');
  delete scripts.lint;
  f.put('src/web/package.json', JSON.stringify({ scripts }));
  f.put('src/web/calls.txt', '');
  const missing = task(f.dir, ['check:web']);
  assert.notEqual(missing.status, 0);
  assert.equal(readFileSync(join(f.dir, 'src/web/calls.txt'), 'utf8'), '');
});
test('CI and docs use the shared gates; package and runner-only coverage remain explicit', () => {
  const ci = readFileSync(join(root, '.github/workflows/ci.yml'), 'utf8').replaceAll('\r\n', '\n');
  const docs = readFileSync(join(root, 'docs/testing.md'), 'utf8');
  const pkg = JSON.parse(readFileSync(join(root, 'src/web/package.json'), 'utf8'));
  for (const target of ['check:go', 'check:web', 'check:agent', 'check:openapi', 'check:delivery', 'test-race']) {
    assert.ok(ci.includes(`run: task ${target}`), `CI omitted ${target}`);
    assert.ok(docs.includes(`task ${target}`), `Documentation omitted ${target}`);
  }
  assert.match(pkg.scripts.lint, /--max-warnings 0(?:\s|$)/);
  assert.doesNotMatch(pkg.scripts.lint, /--quiet/);
  assert.equal(pkg.scripts['type-check'], 'vue-tsc --build');
  assert.match(pkg.scripts.test, /node --test .* && vitest run/);
  assert.match(ci, /CGO_ENABLED: "1"/);
  assert.equal((ci.match(/run: task setup:go/g) ?? []).length, 2);
  assert.match(ci, /os: \[ubuntu-latest, windows-latest\]/);
  const delivery = ci.split('\n  go-api:')[0];
  const checkouts = delivery.split(/      - uses: actions\/checkout@[^\n]+\n/).slice(1);
  assert.equal(checkouts.length, 2);
  assert.match(checkouts[0], /^        if: runner.os != 'Windows'\s*$/);
  const windowsCheckout = checkouts[1].split('      - uses: actions/setup-node@')[0];
  assert.match(windowsCheckout, /^        if: runner.os == 'Windows'\n/);
  assert.match(windowsCheckout, /sparse-checkout-cone-mode: false/);
  assert.match(windowsCheckout, /sparse-checkout: \|\n            \/\*\n            !\/\.squad\/log\/\*\*\n            !\/\.squad\/orchestration-log\/\*\*\s*$/);
  assert.doesNotMatch(ci, /skipping|continue-on-error|--if-present/i);
  const security = readFileSync(join(root, '.github/workflows/security-scan.yml'), 'utf8');
  for (const job of ['gitleaks', 'govulncheck', 'npm-audit', 'pip-audit', 'agent-image-pip-check', 'container-security']) {
    assert.match(security, new RegExp(`^  ${job}:`, 'm'));
  }
  assert.doesNotMatch(security, /continue-on-error:\s*true/);
});
test('version annotation preserves source and line endings without shell replacement ambiguity', () => {
  const source = 'package main\r\n//\t@version\t4.0.0\r\nfunc main() {}\r\n';
  assert.equal(versionAnnotation(source, '4.3.0'), source.replace('4.0.0', '4.3.0'));
  assert.throws(() => versionAnnotation('package main\n', '4.3.0'), /exactly one/);
  assert.throws(() => versionAnnotation(source + source, '4.3.0'), /exactly one/);
  assert.throws(() => versionAnnotation(source, '4.3.0\nbad'), /semantic version/);
});
test('OpenAPI runs the pinned generator, propagates failures and checks every generated artifact', t => {
  const f = fixture(t);
  f.put('VERSION', '4.3.0\n');
  const main = 'package main\n//\t@version\t4.0.0\nfunc main() {}\n';
  f.put('src/api/main.go', main);
  f.put(`bin/${process.platform === 'win32' ? 'swag.exe' : 'swag'}`, '');
  f.put('docs/openapi.json', '{}');
  const calls = [];
  let badVersion = false;
  let failGeneration = false;
  let drift = false;
  const run = (command, args, options) => {
    assert.equal(options.env.GOTOOLCHAIN, 'local');
    assert.equal(options.env.GOPROXY, 'off');
    calls.push({ command, args, cwd: options.cwd });
    if (command === 'go' && args[0] === 'env') return { status: 0, stdout: args[1] === 'GOBIN' ? join(f.dir, 'bin') : f.dir };
    if (command === 'go' && args[0] === 'version') return { status: 0, stdout: `\tmod\tgithub.com/swaggo/swag\t${badVersion ? 'v1.0.0' : 'v1.16.6'}\th1:fixture\n` };
    if (args[0] === 'init') {
      if (failGeneration) return { status: 1, stderr: 'generator failed' };
      f.put('src/api/docs/swagger.json', '{"swagger":"2.0"}');
      return { status: 0 };
    }
    if (command === 'git') return { status: drift ? 1 : 0, stdout: drift ? 'snapshot diff' : '' };
    throw new Error(`Unexpected command ${command}`);
  };
  badVersion = true;
  assert.throws(() => generate(f.dir, { run }), /Expected swag/);
  assert.equal(readFileSync(join(f.dir, 'src/api/main.go'), 'utf8'), main);
  badVersion = false;
  generate(f.dir, { run, check: true });
  assert.equal(readFileSync(join(f.dir, 'VERSION'), 'utf8'), '4.3.0\n');
  assert.equal(readFileSync(join(f.dir, 'src/api/main.go'), 'utf8'), main.replace('4.0.0', '4.3.0'));
  assert.equal(readFileSync(join(f.dir, 'docs/openapi.json'), 'utf8'), '{"swagger":"2.0"}');
  assert.deepEqual(calls.find(call => call.args[0] === 'init').args,
    ['init', '-g', 'main.go', '-o', './docs', '--parseDependency', '--parseInternal']);
  assert.deepEqual(calls.find(call => call.command === 'git').args,
    ['diff', '--exit-code', '--', 'src/api/main.go', 'src/api/docs/docs.go',
      'src/api/docs/swagger.json', 'src/api/docs/swagger.yaml', 'docs/openapi.json']);
  drift = true;
  assert.throws(() => generate(f.dir, { run, check: true }), /snapshot diff/);
  drift = false;
  failGeneration = true;
  assert.throws(() => generate(f.dir, { run }), /generator failed/);
});
