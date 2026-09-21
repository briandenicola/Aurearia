import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export const policy = JSON.parse(readFileSync(new URL('./publish-policy.json', import.meta.url), 'utf8'));
const requireEvidence = (condition, message) => {
  if (!condition) throw new Error(message);
};

export function candidateFrom(event, repository) {
  requireEvidence(/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(repository ?? ''), 'Invalid repository identity');
  const run = event?.workflow_run;
  requireEvidence(event?.action === 'completed' && event.repository?.full_name === repository,
    'Expected a completed workflow_run from this repository');
  requireEvidence(run?.name === 'Quality Gate' && run.event === 'push' &&
    run.head_branch === policy.branch && run.status === 'completed' && run.conclusion === 'success',
  'Expected a successful Quality Gate push on main');
  requireEvidence(Number.isSafeInteger(run.id) && run.id > 0 && /^[a-f0-9]{40}$/.test(run.head_sha),
    'Invalid upstream run or candidate SHA');
  requireEvidence(event.repository.owner?.type === 'User' &&
    Number.isSafeInteger(event.repository.owner.id) && event.repository.owner.id > 0,
  'An explicit individual repository owner is required by this solo-maintainer policy');
  return run.head_sha;
}

export function assertPublishEvidence({ event, repository, upstream, main, environment, branches, checks, approvals }) {
  const sha = candidateFrom(event, repository);
  const owner = event.repository.owner.id;
  requireEvidence(upstream?.id === event.workflow_run.id && upstream.repository?.full_name === repository &&
    upstream.path === '.github/workflows/ci.yml' && upstream.event === 'push' &&
    upstream.head_branch === policy.branch && upstream.head_sha === sha &&
    upstream.status === 'completed' && upstream.conclusion === 'success',
  'Upstream API evidence does not match the successful candidate run');
  requireEvidence(main?.object?.sha === sha, 'Candidate is no longer the main tip; do not publish stale latest tags');

  requireEvidence(Array.isArray(environment?.protection_rules), 'Environment protection evidence is missing');
  const rules = environment.protection_rules.filter(rule => rule.type === 'required_reviewers');
  const reviewers = rules?.[0]?.reviewers;
  requireEvidence(environment?.name === policy.environment && typeof environment.can_admins_bypass === 'boolean' &&
    rules?.length === 1 && rules[0].prevent_self_review === false && reviewers?.length === 1 &&
    reviewers[0].type === 'User' && reviewers[0].reviewer?.id === owner,
  'Release environment must require the owner, allow solo self-review, and report admin-bypass availability');
  requireEvidence(environment.deployment_branch_policy?.protected_branches === false &&
    environment.deployment_branch_policy.custom_branch_policies === true &&
    branches?.length === 1 && branches[0].name === policy.branch && branches[0].type === 'branch',
  'Release environment must allow only the main branch, not tags or wildcard branches');

  requireEvidence(Array.isArray(checks), 'Check-run evidence is missing');
  for (const required of policy.checks) {
    const matching = checks.filter(check => check.name === required.context && check.app?.id === required.app_id);
    if (required.required_for_publish === false && matching.length === 0) continue;
    requireEvidence(matching.length === 1 && matching[0].head_sha === sha &&
      matching[0].status === 'completed' && matching[0].conclusion === 'success',
    `Missing, ambiguous, stale or unsuccessful check: ${required.context}`);
  }
  requireEvidence(Array.isArray(approvals), 'Approval history is missing');
  requireEvidence(approvals.every(review => review && Array.isArray(review.environments) &&
    review.environments.every(item => item && typeof item.name === 'string') &&
    ['approved', 'rejected', 'pending'].includes(review.state)),
  'Malformed GitHub approval history');
  const relevant = approvals.filter(review =>
    review.environments.some(item => item.name === policy.environment));
  requireEvidence(!relevant.some(review => review.state !== 'approved') &&
    relevant.some(review => review.state === 'approved' && review.user?.id === owner && review.user.type === 'User'),
  'This publishing run has no unambiguous GitHub-recorded owner approval for release');
  return sha;
}

export async function verifyMainPublish(env = process.env, fetchImpl = fetch) {
  requireEvidence(env.GITHUB_EVENT_NAME === 'workflow_run', 'Only the workflow_run publishing entry point is allowed');
  requireEvidence(env.GITHUB_API_URL === 'https://api.github.com', 'Unexpected GitHub API origin');
  requireEvidence(env.GITHUB_TOKEN && env.GITHUB_EVENT_PATH && /^[1-9]\d*$/.test(env.GITHUB_RUN_ID ?? ''),
    'Missing GitHub event, token or publishing run identity');
  const event = JSON.parse(readFileSync(env.GITHUB_EVENT_PATH, 'utf8'));
  const repository = env.GITHUB_REPOSITORY;
  const sha = candidateFrom(event, repository);
  const base = `${env.GITHUB_API_URL}/repos/${repository}`;
  const get = async path => {
    const response = await fetchImpl(`${base}/${path}`, {
      method: 'GET', redirect: 'error', signal: AbortSignal.timeout(15000),
      headers: { Accept: 'application/vnd.github+json', Authorization: `Bearer ${env.GITHUB_TOKEN}`,
        'X-GitHub-Api-Version': '2022-11-28' },
    });
    requireEvidence(response.ok, `GitHub evidence request failed (${response.status}): ${path}`);
    return response.json();
  };
  const pages = async (path, field) => {
    const items = [];
    for (let page = 1; page <= 10; page++) {
      const data = await get(`${path}${path.includes('?') ? '&' : '?'}per_page=100&page=${page}`);
      requireEvidence(Array.isArray(data?.[field]), `Malformed GitHub list: ${field}`);
      items.push(...data[field]);
      if (data[field].length < 100) return items;
    }
    throw new Error(`GitHub evidence exceeds the bounded pagination limit: ${field}`);
  };
  const [upstream, main, environment, branches, checks, approvals] = await Promise.all([
    get(`actions/runs/${event.workflow_run.id}`),
    get(`git/ref/heads/${policy.branch}`),
    get(`environments/${policy.environment}`),
    pages(`environments/${policy.environment}/deployment-branch-policies`, 'branch_policies'),
    pages(`commits/${sha}/check-runs?filter=latest`, 'check_runs'),
    get(`actions/runs/${env.GITHUB_RUN_ID}/approvals`),
  ]);
  return assertPublishEvidence({ event, repository, upstream, main, environment, branches, checks, approvals });
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const sha = await verifyMainPublish();
    console.log(`Required checks and GitHub owner-approval evidence verified for ${sha}. No image published by this checker.`);
  } catch (error) {
    console.error(`Main publication blocked: ${error.message}`);
    process.exitCode = 1;
  }
}
