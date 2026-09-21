import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { policy, candidateFrom, assertPublishEvidence, verifyMainPublish } from './verify-main-publish.mjs';

const sha = 'a'.repeat(40);
function evidence() {
  const repository = { full_name: 'owner/example', owner: { id: 7, type: 'User' } };
  const upstream = { id: 700, name: 'Quality Gate', path: '.github/workflows/ci.yml',
    event: 'push', head_branch: 'main', head_sha: sha, status: 'completed', conclusion: 'success',
    repository };
  return {
    repository: repository.full_name,
    event: { action: 'completed', repository, workflow_run: structuredClone(upstream) },
    upstream,
    main: { object: { sha } },
    environment: {
      name: 'release', can_admins_bypass: false,
      deployment_branch_policy: { protected_branches: false, custom_branch_policies: true },
      protection_rules: [{ type: 'required_reviewers', prevent_self_review: false,
        reviewers: [{ type: 'User', reviewer: { id: 7 } }] }],
    },
    branches: [{ name: 'main', type: 'branch' }],
    checks: policy.checks.filter(check => check.required_for_publish !== false).map(check => ({ name: check.context, app: { id: check.app_id },
      head_sha: sha, status: 'completed', conclusion: 'success' })),
    approvals: [{ state: 'approved', user: { id: 7, type: 'User' }, environments: [{ name: 'release' }] }],
  };
}
const failures = [
  ['foreign repository event', f => { f.event.repository.full_name = 'other/example'; }],
  ['PR trigger', f => { f.event.workflow_run.event = 'pull_request'; }],
  ['beta trigger', f => { f.event.workflow_run.head_branch = 'beta'; }],
  ['failed upstream event', f => { f.event.workflow_run.conclusion = 'failure'; }],
  ['invalid candidate SHA', f => { f.event.workflow_run.head_sha = 'main'; }],
  ['missing individual owner', f => { f.event.repository.owner.type = 'Organization'; }],
  ['wrong upstream workflow', f => { f.upstream.path = '.github/workflows/other.yml'; }],
  ['stale upstream API result', f => { f.upstream.head_sha = 'b'.repeat(40); }],
  ['candidate no longer main tip', f => { f.main.object.sha = 'b'.repeat(40); }],
  ['unconfigured auto-created environment', f => { f.environment.protection_rules = []; }],
  ['environment admin bypass', f => { f.environment.can_admins_bypass = true; }],
  ['unavailable second human', f => { f.environment.protection_rules[0].prevent_self_review = true; }],
  ['wrong required reviewer', f => { f.environment.protection_rules[0].reviewers[0].reviewer.id = 8; }],
  ['wildcard environment branch', f => { f.branches[0].name = '*'; }],
  ['tag masquerading as main', f => { f.branches[0].type = 'tag'; }],
  ['additional deployment branch', f => { f.branches.push({ name: 'beta', type: 'branch' }); }],
  ['missing check', f => { f.checks.pop(); }],
  ['spoofed check provider', f => { f.checks[0].app.id = 123; }],
  ['stale check SHA', f => { f.checks[0].head_sha = 'b'.repeat(40); }],
  ['failed check', f => { f.checks[0].conclusion = 'failure'; }],
  ['pending check', f => { f.checks[0].status = 'in_progress'; }],
  ['neutral check', f => { f.checks[0].conclusion = 'neutral'; }],
  ['skipped check', f => { f.checks[0].conclusion = 'skipped'; }],
  ['ambiguous check', f => { f.checks.push({ ...f.checks[0] }); }],
  ['failed PR-only aggregate when present', f => {
    f.checks.push({ name: 'CodeQL', app: { id: 57789 }, head_sha: sha, status: 'completed', conclusion: 'failure' });
  }],
  ['agent-authored checkbox without approval', f => { f.approvals = []; f.accepted = true; }],
  ['non-owner approval', f => { f.approvals[0].user.id = 8; }],
  ['approval for another environment', f => { f.approvals[0].environments[0].name = 'copilot'; }],
  ['rejected release', f => { f.approvals.push({ ...f.approvals[0], state: 'rejected' }); }],
  ['pending release decision', f => { f.approvals[0].state = 'pending'; }],
];
test('accepts only exact-candidate success with GitHub owner approval', () => {
  assert.equal(policy.checks.length, 20);
  assert.equal(new Set(policy.checks.map(check => check.context)).size, 20);
  assert.deepEqual(policy.checks.filter(check => check.required_for_publish === false),
    [{ context: 'CodeQL', app_id: 57789, required_for_publish: false }]);
  assert.equal(evidence().checks.length, 19);
  assert.equal(candidateFrom(evidence().event, 'owner/example'), sha);
  assert.equal(assertPublishEvidence(evidence()), sha);
  const withAggregate = evidence();
  withAggregate.checks.push({ name: 'CodeQL', app: { id: 57789 }, head_sha: sha, status: 'completed', conclusion: 'success' });
  assert.equal(assertPublishEvidence(withAggregate), sha);
});
for (const [name, mutate] of failures) {
  test(`publication rejects ${name}`, () => {
    const fixture = evidence();
    mutate(fixture);
    assert.throws(() => assertPublishEvidence(fixture), Error, name);
  });
}

function apiFixture(t) {
  const dir = mkdtempSync(join(tmpdir(), 'publish-evidence-'));
  t.after(() => rmSync(dir, { recursive: true, force: true }));
  const data = evidence();
  const eventPath = join(dir, 'event.json');
  writeFileSync(eventPath, JSON.stringify(data.event));
  const env = { GITHUB_EVENT_NAME: 'workflow_run', GITHUB_API_URL: 'https://api.github.com',
    GITHUB_REPOSITORY: data.repository, GITHUB_RUN_ID: '901', GITHUB_TOKEN: 'fixture-token',
    GITHUB_EVENT_PATH: eventPath };
  const requests = [];
  const fetchImpl = async (url, options) => {
    requests.push({ url, options });
    assert.equal(options.method, 'GET');
    assert.equal(options.redirect, 'error');
    assert.equal(options.headers.Authorization, 'Bearer fixture-token');
    const path = new URL(url).pathname;
    let body;
    if (path.endsWith('/actions/runs/700')) body = data.upstream;
    else if (path.endsWith('/git/ref/heads/main')) body = data.main;
    else if (path.endsWith('/environments/release')) body = data.environment;
    else if (path.endsWith('/deployment-branch-policies')) body = { branch_policies: data.branches };
    else if (path.endsWith('/check-runs')) body = { check_runs: data.checks };
    else if (path.endsWith('/actions/runs/901/approvals')) body = data.approvals;
    else throw new Error(`Unexpected API route: ${url}`);
    return { ok: true, status: 200, json: async () => body };
  };
  return { dir, env, requests, fetchImpl };
}
test('runtime uses only read APIs and approvals for the publishing run, never the upstream run', async t => {
  const f = apiFixture(t);
  assert.equal(await verifyMainPublish(f.env, f.fetchImpl), sha);
  assert.equal(f.requests.length, 6);
  assert.ok(f.requests.some(request => request.url.endsWith('/actions/runs/901/approvals')));
  assert.ok(f.requests.some(request => request.url.includes(`/commits/${sha}/check-runs?filter=latest`)));
  assert.ok(f.requests.every(request => request.url.startsWith('https://api.github.com/repos/owner/example/')));
});
test('malformed approval records are not ignored beside a valid owner approval', () => {
  for (const invalid of [{}, null, { environments: {}, state: 'approved' },
    { environments: [null], state: 'approved' }]) {
    const fixture = evidence();
    fixture.approvals.push(invalid);
    assert.throws(() => assertPublishEvidence(fixture), /Malformed GitHub approval history/);
  }
});
test('runtime fails closed on missing permissions, malformed lists, untrusted origins and trigger overrides', async t => {
  const f = apiFixture(t);
  await assert.rejects(verifyMainPublish(f.env, async () => ({ ok: false, status: 403 })), /403/);
  await assert.rejects(verifyMainPublish(f.env, async () => ({ ok: true, json: async () => ({}) })), /Malformed GitHub list/);
  for (const change of [
    { GITHUB_TOKEN: '' }, { GITHUB_EVENT_NAME: 'workflow_dispatch' },
    { GITHUB_API_URL: 'https://untrusted.invalid' }, { GITHUB_RUN_ID: '../700' },
  ]) {
    await assert.rejects(verifyMainPublish({ ...f.env, ...change }, () => {
      assert.fail('Invalid input must not make network requests');
    }));
  }
});
test('check evidence paginates and rejects unbounded responses', async t => {
  const f = apiFixture(t);
  const fillers = Array.from({ length: 100 }, (_, i) => ({ name: `other-${i}` }));
  const firstPage = async (url, options) => {
    if (url.includes('/check-runs?') && url.endsWith('page=1')) {
      return { ok: true, json: async () => ({ check_runs: fillers }) };
    }
    return f.fetchImpl(url, options);
  };
  assert.equal(await verifyMainPublish(f.env, firstPage), sha);
  let pages = 0;
  await assert.rejects(verifyMainPublish(f.env, async (url, options) => url.includes('/check-runs?')
    ? (pages++, { ok: true, json: async () => ({ check_runs: fillers }) }) : f.fetchImpl(url, options)),
  /pagination limit/);
  assert.equal(pages, 10);
});
test('bypassing evidence rejection makes every negative contract fail', async t => {
  const f = apiFixture(t);
  const source = readFileSync(new URL('./verify-main-publish.mjs', import.meta.url), 'utf8');
  const mutant = source.replace('if (!condition) throw new Error(message);', 'if (false) throw new Error(message);');
  assert.notEqual(mutant, source);
  writeFileSync(join(f.dir, 'publish-policy.json'), JSON.stringify(policy));
  const target = join(f.dir, 'mutant.mjs');
  writeFileSync(target, mutant);
  const changed = await import(pathToFileURL(target).href);
  for (const [name, mutate] of failures) {
    const fixture = evidence();
    mutate(fixture);
    assert.throws(() => assertPublishEvidence(fixture), Error, name);
    assert.throws(() => assert.throws(() => changed.assertPublishEvidence(fixture), Error, name), assert.AssertionError, name);
  }
});
function assertMainWorkflow(main) {
  assert.match(main, /name: release\r?\n/);
  assert.match(main, /run: node scripts\/delivery\/verify-main-publish\.mjs/);
  assert.match(main, /actions: read/);
  assert.match(main, /checks: read/);
  assert.match(main, /group: main-publish\r?\n\s+cancel-in-progress: false/);
  for (const id of ['build-and-push-app', 'build-and-push-agent']) {
    assert.match(main, new RegExp(`${id}:\\r?\\n\\s+needs: approve-release`));
    const job = main.split(`  ${id}:`)[1].split(/\r?\n  [\w-]+:/)[0];
    assert.match(job, /actions: read/);
    assert.match(job, /checks: read/);
    assert.ok(job.indexOf('run: node scripts/delivery/verify-main-publish.mjs') > 0);
    assert.ok(job.indexOf('run: node scripts/delivery/verify-main-publish.mjs') < job.indexOf('uses: docker/'));
  }
  assert.equal((main.match(/ref: \$\{\{ github\.event\.workflow_run\.head_sha \}\}/g) ?? []).length, 3);
  assert.doesNotMatch(main, /ref: (?:main|beta)\b|always\(\)/);
}
test('both main publishers depend on approval and retain checked-SHA checkout; beta stays automatic', () => {
  const read = name => readFileSync(new URL(`../../.github/workflows/${name}`, import.meta.url), 'utf8');
  const main = read('docker-publish.yml');
  assertMainWorkflow(main);
  for (const id of ['build-and-push-app', 'build-and-push-agent']) {
    const boundary = main.indexOf(`  ${id}:`);
    for (const guard of ['needs: approve-release', 'run: node scripts/delivery/verify-main-publish.mjs']) {
      const mutant = main.slice(0, boundary) + main.slice(boundary).replace(guard, '# omitted for negative control');
      assert.notEqual(mutant, main);
      assert.throws(() => assertMainWorkflow(mutant), assert.AssertionError, `${id}: ${guard}`);
    }
  }
  const beta = read('docker-publish-beta.yml');
  assert.doesNotMatch(beta, /approve-release|environment:/);
  assert.equal((beta.match(/ref: \$\{\{ github\.event\.workflow_run\.head_sha \}\}/g) ?? []).length, 2);
  assert.match(beta, /IMAGE_APP\}:beta/);
});
