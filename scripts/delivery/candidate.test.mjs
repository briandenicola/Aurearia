import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { spawnSync } from 'node:child_process';
import { validationPolicy, verifyCandidate, verifyBetaChecks, EvidencePending } from './verify-candidate.mjs';
import { assertImageReceipt, promoteImage } from './image-receipt.mjs';
import { candidateFixture } from './candidate-fixtures.mjs';
import { githubEvidence } from './github-evidence.mjs';

const failures = [
  ['changed merge tree', f => { f.commits[f.candidateSha].tree.sha = 'e'.repeat(40); }],
  ['head-only PR checkout', f => { f.commits[f.candidateSha].parents = []; }],
  ['wrong merge base', f => { f.commits[f.candidateSha].parents[0].sha = 'f'.repeat(40); }],
  ['foreign head repository', f => { f.event.pull_request.head.repo.full_name = 'foreign/repo'; }],
  ['foreign base repository', f => { f.event.pull_request.base.repo.full_name = 'foreign/repo'; }],
  ['non-beta PR', f => { f.event.pull_request.head.ref = 'feature'; }],
  ['wrong check provider', f => { f.checks[0].app.id = 5; }],
  ['wrong check SHA', f => { f.checks[0].head_sha = f.candidateSha; }],
  ['failed check', f => { f.checks[0].conclusion = 'failure'; }],
  ['skipped check', f => { f.checks[0].conclusion = 'skipped'; }],
  ['neutral check', f => { f.checks[0].conclusion = 'neutral'; }],
  ['ambiguous check', f => { f.checks.push({ ...f.checks[0] }); }],
  ['spoofed run URL', f => { f.checks[0].details_url = 'https://githud.com/owner/example/actions/runs/400/job/1'; }],
  ['wrong workflow path', f => { f.runs.get(400).path = '.github/workflows/other.yml'; }],
  ['PR results masquerading as beta results', f => { f.runs.get(400).event = 'pull_request'; }],
  ['manual results replacing mandatory push validation', f => { f.runs.get(400).event = 'workflow_dispatch'; }],
  ['wrong run branch', f => { f.runs.get(400).head_branch = 'main'; }],
  ['wrong run SHA', f => { f.runs.get(400).head_sha = f.candidateSha; }],
  ['wrong check suite', f => { f.checks[0].check_suite.id++; }],
  ['failed validation workflow', f => { f.runs.get(400).conclusion = 'failure'; }],
  ['expired receipt', f => { f.artifacts[0].expired = true; }],
  ['different receipt runs', f => {
    f.artifacts[1].workflow_run.id = 901;
    f.runs.set(901, { ...f.publisher, id: 901 });
  }],
  ['wrong publisher source', f => { f.publisher.head_sha = f.candidateSha; }],
  ['wrong publisher path', f => { f.publisher.path = '.github/workflows/other.yml'; }],
  ['legacy unrelated publisher event', f => { f.publisher.event = 'workflow_run'; }],
  ['failed publisher', f => { f.publisher.conclusion = 'failure'; }],
];
test('same-tree PR and main reuse all original beta coverage and paired image receipts', async () => {
  assert.equal(validationPolicy.checks.length, 19);
  assert.equal(new Set(validationPolicy.checks.map(item => item.context)).size, 19);
  const f = candidateFixture();
  assert.deepEqual(await verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'), f.result);
  assert.deepEqual(await verifyCandidate(f.api, f.repository, f.candidateSha,
    { ref: 'refs/heads/main', after: f.candidateSha }, 'push'), f.result);
  await verifyBetaChecks(f.api, f.repository, f.sourceSha);
});
for (const [name, mutate] of failures) {
  test(`candidate reuse rejects ${name}`, async () => {
    const f = candidateFixture();
    mutate(f);
    await assert.rejects(verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'), Error, name);
  });
}
test('only genuinely pending evidence may be waited for; failures are not retried as success', async () => {
  for (const mutate of [
    f => { f.checks.pop(); }, f => { f.checks[0].status = 'in_progress'; },
    f => { f.runs.get(400).status = 'in_progress'; }, f => { f.artifacts.pop(); },
    f => { f.publisher.status = 'in_progress'; },
  ]) {
    const f = candidateFixture();
    mutate(f);
    await assert.rejects(verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'), EvidencePending);
  }
  const f = candidateFixture();
  f.artifacts[1].workflow_run.id = 901;
  f.runs.set(901, { ...f.publisher, id: 901, status: 'in_progress', conclusion: null });
  await assert.rejects(verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'), EvidencePending);
});
test('main cannot reuse an unmerged, ambiguous or unrelated PR', async () => {
  for (const change of [
    pr => { pr.merged_at = null; }, pr => { pr.merge_commit_sha = 'e'.repeat(40); },
    pr => { pr.head.repo.full_name = 'foreign/repo'; },
  ]) {
    const f = candidateFixture();
    change(f.pr);
    await assert.rejects(verifyCandidate(f.api, f.repository, f.candidateSha,
      { ref: 'refs/heads/main', after: f.candidateSha }, 'push'));
  }
  const f = candidateFixture();
  await assert.rejects(verifyCandidate(f.api, f.repository, f.candidateSha, {}, 'workflow_dispatch'));
});
test('optional aggregate/runtime checks cannot hide failures when present', async () => {
  for (const item of validationPolicy.optional_checks) {
    const f = candidateFixture();
    f.checks.push({ name: item.context, app: { id: item.app_id }, head_sha: f.sourceSha,
      status: 'completed', conclusion: 'failure' });
    await assert.rejects(verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'));
  }
});
function runtimeFixture(event) {
  const f = candidateFixture();
  const required = validationPolicy.optional_checks.find(item => item.context === 'Isolated runtime acceptance');
  const run = { ...f.runs.get(400), id: 950, path: required.workflow, event, check_suite_id: 1950 };
  const check = { ...f.checks[0], name: required.context, app: { id: required.app_id },
    check_suite: { id: run.check_suite_id },
    details_url: `https://github.com/${f.repository}/actions/runs/${run.id}/job/1951` };
  f.runs.set(run.id, run);
  f.checks.push(check);
  return { ...f, runtime: run, runtimeCheck: check };
}
test('successful optional runtime push and manual runs preserve mandatory push validation', async () => {
  for (const event of ['push', 'workflow_dispatch']) {
    const f = runtimeFixture(event);
    assert.deepEqual(await verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'), f.result);
  }
});
test('optional manual runtime results still reject failed, foreign and untrusted evidence', async () => {
  for (const mutate of [
    f => { f.runtimeCheck.app.id = 5; },
    f => { f.runtime.repository = { full_name: 'foreign/repo' }; },
    f => { f.runtime.event = 'pull_request'; },
    f => { f.runtime.head_branch = 'main'; },
    f => { f.runtime.head_sha = f.candidateSha; },
    f => { f.runtime.path = '.github/workflows/other.yml'; },
    f => { f.runtimeCheck.check_suite.id++; },
    f => { f.runtimeCheck.conclusion = 'failure'; },
    f => { f.runtime.conclusion = 'failure'; },
  ]) {
    const f = runtimeFixture('workflow_dispatch');
    mutate(f);
    await assert.rejects(verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'));
  }
});
test('image promotion uses the exact receipt digest without rebuilding or invoking a shell', () => {
  const f = candidateFixture();
  for (const kind of ['app', 'agent']) {
    const receipt = { sourceSha: f.sourceSha, tree: f.tree, kind, digest: `sha256:${'a'.repeat(64)}` };
    const expected = { sourceSha: f.sourceSha, tree: f.tree, kind };
    let calls = 0;
    assert.equal(promoteImage(receipt, expected, `owner/${kind}`, f.candidateSha, (command, args, options) => {
      calls++;
      assert.equal(command, 'docker');
      assert.deepEqual(args, ['buildx', 'imagetools', 'create', '--prefer-index=false',
        '--tag', `owner/${kind}:sha-${f.candidateSha}`, '--tag', `owner/${kind}:sha-${f.candidateSha.slice(0, 7)}`,
        '--tag', `owner/${kind}:latest`, `owner/${kind}@${receipt.digest}`]);
      assert.equal(options.shell, false);
      return { status: 0 };
    }), receipt.digest);
    assert.equal(calls, 1);
    for (const change of [{ kind: 'other' }, { sourceSha: f.candidateSha }, { tree: 'e'.repeat(40) },
      { digest: 'latest' }, { digest: `sha256:${'z'.repeat(64)}` }]) {
      assert.throws(() => assertImageReceipt({ ...receipt, ...change }, expected));
    }
    assert.throws(() => promoteImage(receipt, expected, 'owner/app', f.candidateSha, () => ({ status: 1 })));
    assert.throws(() => promoteImage(receipt, expected, '--bad', f.candidateSha, () => assert.fail()));
  }
});
test('receipt writer CLI records the exact build identity and fails explicitly on invalid digests', t => {
  const dir = mkdtempSync(join(tmpdir(), 'image-receipt-'));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  const f = candidateFixture();
  const path = join(dir, 'receipt.json');
  const cli = fileURLToPath(new URL('./image-receipt.mjs', import.meta.url));
  const env = { ...process.env, SOURCE_SHA: f.sourceSha, SOURCE_TREE: f.tree, IMAGE_DIGEST: `sha256:${'a'.repeat(64)}` };
  const result = spawnSync(process.execPath, [cli, 'write', 'app', path], { env, encoding: 'utf8' });
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(JSON.parse(readFileSync(path, 'utf8')),
    { sourceSha: f.sourceSha, tree: f.tree, kind: 'app', digest: env.IMAGE_DIGEST });
  const bad = spawnSync(process.execPath, [cli, 'write', 'app', path],
    { env: { ...env, IMAGE_DIGEST: 'latest' }, encoding: 'utf8' });
  assert.notEqual(bad.status, 0);
  assert.match(bad.stderr, /Image receipt\/promotion blocked/);
});
test('candidate evidence client is read-only, bounded, authenticated and fail-closed', async () => {
  const env = { GITHUB_API_URL: 'https://api.github.com', GITHUB_REPOSITORY: 'owner/example', GITHUB_TOKEN: 'fixture-token' };
  const client = githubEvidence(env, async (url, options) => {
    assert.ok(url.startsWith('https://api.github.com/repos/owner/example/'));
    assert.equal(options.method, 'GET');
    assert.equal(options.redirect, 'error');
    assert.equal(options.headers.Authorization, 'Bearer fixture-token');
    return { ok: true, json: async () => ({ items: [] }) };
  });
  assert.deepEqual(await client.list('actions/artifacts', 'items'), []);
  await assert.rejects(client.get('https://foreign.invalid'));
  await assert.rejects(githubEvidence(env, async () => ({ ok: false, status: 403 })).get('actions/runs/1'), /403/);
  await assert.rejects(githubEvidence(env, async () => ({ ok: true, json: async () => ({}) })).list('actions/artifacts', 'items'), /Malformed/);
  assert.throws(() => githubEvidence({ ...env, GITHUB_API_URL: 'https://foreign.invalid' }));
  await assert.rejects(githubEvidence(env, async () => ({
    ok: true, json: async () => ({ items: Array.from({ length: 100 }, () => ({})) }),
  })).list('actions/artifacts', 'items'), /pagination limit/);
});
function assertWorkflowContract(read) {
  for (const name of ['ci.yml', 'security-scan.yml', 'feature362-compatibility.yml', 'codeql.yml']) {
    const text = read(name);
    assert.match(text, /pull_request:\r?\n\s+branches: \[beta\]/, name);
    assert.match(text, /push:\r?\n\s+branches: \[beta\]/, name);
    assert.doesNotMatch(text, /branches: \[(?:main|main, beta|beta, main)\]/, name);
  }
  const gate = read('candidate.yml');
  assert.match(gate, /pull_request:\r?\n\s+branches: \[main\]/);
  assert.match(gate, /push:\r?\n\s+branches: \[main\]/);
  assert.match(gate, /name: Verified candidate/);
  assert.match(gate, /run: node scripts\/delivery\/verify-candidate\.mjs/);
  assert.doesNotMatch(gate, /continue-on-error|npm ci|uv sync|go build|docker\/build/);
  const publisher = read('docker-publish.yml');
  assert.match(publisher, /workflows: \[Verified Candidate\]/);
  assert.doesNotMatch(publisher, /docker\/build-push-action|cache-to:|BUILD_DATE/);
  for (const kind of ['app', 'agent']) {
    assert.ok(publisher.includes(`image-receipt.mjs promote ${kind}`));
    assert.ok(publisher.includes(`name: beta-${kind}-`));
  }
  const beta = read('docker-publish-beta.yml');
  assert.match(beta, /push:\r?\n\s+branches: \[beta\]/);
  assert.match(beta, /verify-candidate\.mjs --beta/);
  assert.equal((beta.match(/needs: validate/g) ?? []).length, 2);
  assert.equal((beta.match(/if-no-files-found: error/g) ?? []).length, 2);
}
test('workflow wiring removes duplicate full rounds and never turns reuse into skipped coverage', () => {
  const read = name => readFileSync(new URL(`../../.github/workflows/${name}`, import.meta.url), 'utf8');
  assertWorkflowContract(read);
  for (const [file, from, to] of [
    ['ci.yml', 'branches: [beta]', 'branches: [main, beta]'],
    ['candidate.yml', 'run: node scripts/delivery/verify-candidate.mjs', 'run: echo assumed success'],
    ['docker-publish.yml', 'workflows: [Verified Candidate]', 'workflows: [Quality Gate]'],
    ['docker-publish.yml', 'image-receipt.mjs promote app', 'image-receipt.mjs rebuild app'],
    ['docker-publish-beta.yml', 'needs: validate', '# missing validation dependency'],
  ]) {
    const changed = read(file).replace(from, to);
    assert.notEqual(changed, read(file));
    assert.throws(() => assertWorkflowContract(name => name === file ? changed : read(name)), assert.AssertionError);
  }
});
test('bypassing evidence rejection breaks candidate negative contracts', async t => {
  const dir = mkdtempSync(join(tmpdir(), 'candidate-mutation-'));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  for (const file of ['verify-candidate.mjs', 'validation-policy.json', 'github-evidence.mjs']) {
    let text = readFileSync(new URL(file, import.meta.url), 'utf8');
    if (file === 'github-evidence.mjs') text = text.replace('if (!condition) throw new Error(message);',
      'if (false) throw new Error(message);');
    writeFileSync(join(dir, file), text);
  }
  const mutant = await import(pathToFileURL(join(dir, 'verify-candidate.mjs')).href);
  for (const [name, mutate] of failures.filter(([name]) => name !== 'wrong check provider')) {
    const f = candidateFixture();
    mutate(f);
    const outcome = await mutant.verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request')
      .then(() => 'accepted', () => 'rejected');
    assert.equal(outcome, 'accepted', `${name}: guard mutation must make the negative case fail`);
  }
  const runtime = runtimeFixture('pull_request');
  assert.deepEqual(await mutant.verifyCandidate(runtime.api, runtime.repository, runtime.candidateSha,
    runtime.event, 'pull_request'), runtime.result);
  const original = readFileSync(new URL('./verify-candidate.mjs', import.meta.url), 'utf8');
  const providerMutant = original.replace('check.app?.id === required.app_id', 'true')
    .replace("from './github-evidence.mjs'", "from './provider-github-evidence.mjs'");
  assert.notEqual(providerMutant, original);
  writeFileSync(join(dir, 'provider-mutant.mjs'), providerMutant);
  writeFileSync(join(dir, 'provider-github-evidence.mjs'), readFileSync(new URL('./github-evidence.mjs', import.meta.url), 'utf8'));
  const provider = await import(pathToFileURL(join(dir, 'provider-mutant.mjs')).href);
  const f = candidateFixture();
  f.checks[0].app.id = 5;
  assert.deepEqual(await provider.verifyCandidate(f.api, f.repository, f.candidateSha, f.event, 'pull_request'), f.result);
  writeFileSync(join(dir, 'image-receipt.mjs'), readFileSync(new URL('./image-receipt.mjs', import.meta.url), 'utf8'));
  const image = await import(pathToFileURL(join(dir, 'image-receipt.mjs')).href);
  const receipt = { sourceSha: f.sourceSha, tree: f.tree, kind: 'app', digest: `sha256:${'a'.repeat(64)}` };
  const expected = { sourceSha: f.sourceSha, tree: f.tree, kind: 'app' };
  for (const change of [{ sourceSha: f.candidateSha }, { tree: 'e'.repeat(40) }, { kind: 'agent' }, { digest: 'latest' }]) {
    const invalid = { ...receipt, ...change };
    assert.throws(() => assertImageReceipt(invalid, expected));
    assert.doesNotThrow(() => image.assertImageReceipt(invalid, expected));
  }
});
