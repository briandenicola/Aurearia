import { appendFileSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { githubEvidence, requireEvidence } from './github-evidence.mjs';

export const validationPolicy = JSON.parse(readFileSync(new URL('./validation-policy.json', import.meta.url), 'utf8'));
export class EvidencePending extends Error {}
const shaPattern = /^[a-f0-9]{40}$/;
const pending = (condition, message) => {
  if (!condition) throw new EvidencePending(message);
};

export async function verifyBetaChecks(api, repository, sha) {
  requireEvidence(shaPattern.test(sha), 'Invalid beta candidate SHA');
  const checks = await api.list(`commits/${sha}/check-runs?filter=latest`, 'check_runs');
  const runs = new Map();
  for (const required of [...validationPolicy.checks, ...validationPolicy.optional_checks]) {
    const optional = validationPolicy.optional_checks.includes(required);
    const named = checks.filter(check => check.name === required.context);
    if (!named.length && optional) continue;
    pending(named.length > 0, `Waiting for beta check: ${required.context}`);
    const matching = named.filter(check => check.app?.id === required.app_id);
    requireEvidence(matching.length === 1, `Ambiguous or untrusted beta check: ${required.context}`);
    const check = matching[0];
    requireEvidence(check.head_sha === sha, `Wrong beta check SHA: ${required.context}`);
    pending(check.status === 'completed', `Waiting for beta check: ${required.context}`);
    requireEvidence(check.conclusion === 'success', `Unsuccessful beta check: ${required.context}`);
    if (!required.workflow) continue;
    const prefix = `https://github.com/${repository}/actions/runs/`;
    requireEvidence(typeof check.details_url === 'string' && check.details_url.startsWith(prefix),
      `Untrusted beta run URL: ${required.context}`);
    const match = check.details_url.slice(prefix.length).match(/^([1-9]\d*)\/job\/[1-9]\d*(?:\?.*)?$/);
    requireEvidence(match, `Invalid beta run identity: ${required.context}`);
    const id = match[1];
    if (!runs.has(id)) runs.set(id, await api.get(`actions/runs/${id}`));
    const run = runs.get(id);
    requireEvidence(String(run.id) === id && run.repository?.full_name === repository &&
      run.head_sha === sha && run.head_branch === 'beta' &&
      (optional ? required.events?.includes(run.event) : run.event === 'push') &&
      run.path === required.workflow && run.check_suite_id === check.check_suite?.id,
    `Beta workflow provenance mismatch: ${required.context}`);
    pending(run.status === 'completed', `Waiting for beta workflow: ${required.workflow}`);
    requireEvidence(run.conclusion === 'success', `Unsuccessful beta workflow: ${required.workflow}`);
  }
}

export async function candidateSource(api, repository, candidateSha, event, eventName) {
  requireEvidence(shaPattern.test(candidateSha), 'Invalid merge candidate SHA');
  let pr;
  if (eventName === 'pull_request') {
    pr = event.pull_request;
  } else {
    requireEvidence(eventName === 'push' && event.ref === 'refs/heads/main' && event.after === candidateSha,
      'Expected a main push or beta-to-main PR');
    const prs = await api.list(`commits/${candidateSha}/pulls`);
    const matches = prs.filter(item => item.merge_commit_sha === candidateSha && item.merged_at &&
      item.base?.ref === 'main' && item.head?.ref === 'beta');
    requireEvidence(matches.length === 1, 'Main candidate must identify one merged beta-to-main PR');
    pr = matches[0];
  }
  requireEvidence(pr?.base?.ref === 'main' && pr.base.repo?.full_name === repository &&
    pr.head?.ref === 'beta' && pr.head.repo?.full_name === repository && shaPattern.test(pr.head.sha),
  'Only this repository beta-to-main candidate may reuse beta evidence');
  const sourceSha = pr.head.sha;
  const [candidate, source] = await Promise.all([
    api.get(`git/commits/${candidateSha}`), api.get(`git/commits/${sourceSha}`),
  ]);
  if (eventName === 'pull_request') requireEvidence(shaPattern.test(pr.base.sha) &&
    candidate.parents?.length === 2 && candidate.parents[0].sha === pr.base.sha &&
    candidate.parents[1].sha === sourceSha, 'Candidate is not the proposed PR merge commit');
  requireEvidence(candidate.sha === candidateSha && source.sha === sourceSha &&
    shaPattern.test(candidate.tree?.sha) && candidate.tree.sha === source.tree?.sha,
  'Content differs from tested beta: reconcile the merge into beta and run full beta validation');
  return { sourceSha, tree: source.tree.sha };
}

export async function verifyCandidate(api, repository, candidateSha, event, eventName) {
  const source = await candidateSource(api, repository, candidateSha, event, eventName);
  await verifyBetaChecks(api, repository, source.sourceSha);
  const artifacts = await Promise.all(['app', 'agent'].map(async kind => {
    const name = `beta-${kind}-${source.sourceSha}`;
    const found = (await api.list(`actions/artifacts?name=${name}`, 'artifacts'))
      .filter(item => item.name === name).sort((a, b) => b.id - a.id);
    pending(found.length > 0, `Waiting for beta ${kind} image receipt`);
    const artifact = found[0];
    requireEvidence(!artifact.expired && Number.isSafeInteger(artifact.id) &&
      Number.isSafeInteger(artifact.workflow_run?.id), `Expired or malformed beta ${kind} image receipt`);
    return artifact;
  }));
  const publisherRunId = Math.max(...artifacts.map(item => item.workflow_run.id));
  const publisher = await api.get(`actions/runs/${publisherRunId}`);
  requireEvidence(publisher.id === publisherRunId && publisher.repository?.full_name === repository &&
    publisher.path === '.github/workflows/docker-publish-beta.yml' && publisher.event === 'push' &&
    publisher.head_branch === 'beta' && publisher.head_sha === source.sourceSha,
  'Beta image publisher provenance mismatch');
  pending(publisher.status === 'completed', 'Waiting for beta image publisher');
  requireEvidence(publisher.conclusion === 'success', 'Unsuccessful beta image publisher');
  requireEvidence(artifacts.every(item => item.workflow_run.id === publisherRunId),
    'Beta app and agent image receipts belong to different publisher runs');
  return { ...source, publisherRunId };
}

export function writeCandidateOutputs(result, path) {
  if (path) appendFileSync(path,
    `source_sha=${result.sourceSha}\nsource_tree=${result.tree}\npublisher_run_id=${result.publisherRunId}\n`);
}

async function main(env) {
  const api = githubEvidence(env);
  const event = JSON.parse(readFileSync(env.GITHUB_EVENT_PATH, 'utf8'));
  requireEvidence(event.repository?.full_name === env.GITHUB_REPOSITORY, 'Foreign repository event');
  const beta = process.argv.includes('--beta');
  if (beta) requireEvidence(env.GITHUB_EVENT_NAME === 'push' && event.ref === 'refs/heads/beta' &&
    event.after === env.GITHUB_SHA, 'Beta publication requires an exact beta push');
  const deadline = Date.now() + 20 * 60 * 1000;
  while (true) {
    try {
      if (beta) await verifyBetaChecks(api, env.GITHUB_REPOSITORY, env.GITHUB_SHA);
      else {
        const result = await verifyCandidate(api, env.GITHUB_REPOSITORY, env.GITHUB_SHA, event, env.GITHUB_EVENT_NAME);
        writeCandidateOutputs(result, env.GITHUB_OUTPUT);
      }
      console.log(`Verified ${beta ? 'full beta validation' : 'identical tested tree and beta image receipts'} for ${env.GITHUB_SHA}.`);
      return;
    } catch (error) {
      if (!(error instanceof EvidencePending) || Date.now() >= deadline) throw error;
      console.log(error.message);
      await new Promise(resolve => setTimeout(resolve, 30000));
    }
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.env).catch(error => {
    console.error(`Candidate verification blocked: ${error.message}`);
    process.exitCode = 1;
  });
}
