import { validationPolicy } from './verify-candidate.mjs';

export function candidateFixture(candidateSha = 'a'.repeat(40)) {
  const sourceSha = 'b'.repeat(40);
  const tree = 'd'.repeat(40);
  const repository = 'owner/example';
  const pr = { merge_commit_sha: candidateSha, merged_at: '2026-10-16T12:00:00Z',
    base: { ref: 'main', sha: 'c'.repeat(40), repo: { full_name: repository } },
    head: { ref: 'beta', sha: sourceSha, repo: { full_name: repository } } };
  const commits = {
    [candidateSha]: { sha: candidateSha, tree: { sha: tree }, parents: [{ sha: pr.base.sha }, { sha: sourceSha }] },
    [sourceSha]: { sha: sourceSha, tree: { sha: tree } },
  };
  const runs = new Map();
  const checks = validationPolicy.checks.map(required => {
    let run = [...runs.values()].find(item => item.path === required.workflow);
    if (!run) {
      const id = 400 + runs.size;
      run = { id, repository: { full_name: repository }, head_sha: sourceSha, head_branch: 'beta',
        event: 'push', path: required.workflow, status: 'completed', conclusion: 'success', check_suite_id: id + 1000 };
      runs.set(id, run);
    }
    return { name: required.context, app: { id: required.app_id }, head_sha: sourceSha,
      status: 'completed', conclusion: 'success', check_suite: { id: run.check_suite_id },
      details_url: `https://github.com/${repository}/actions/runs/${run.id}/job/${1000 + runs.size}` };
  });
  const publisher = { id: 900, repository: { full_name: repository },
    path: '.github/workflows/docker-publish-beta.yml', event: 'push', head_branch: 'beta',
    head_sha: sourceSha, status: 'completed', conclusion: 'success' };
  runs.set(publisher.id, publisher);
  const artifacts = ['app', 'agent'].map((kind, index) => ({
    id: 1000 + index, name: `beta-${kind}-${sourceSha}`, expired: false, workflow_run: { id: publisher.id },
  }));
  const event = { pull_request: structuredClone(pr), repository: { full_name: repository } };
  const get = async path => {
    const [route, query] = path.split('?');
    if (route === `commits/${candidateSha}/pulls`) return [pr];
    if (route.startsWith('git/commits/')) return commits[route.split('/').at(-1)];
    if (route === `commits/${sourceSha}/check-runs`) return { check_runs: checks };
    if (route.startsWith('actions/runs/')) {
      const run = runs.get(Number(route.split('/').at(-1)));
      if (run) return run;
    }
    if (route === 'actions/artifacts') {
      const name = new URLSearchParams(query).get('name');
      return { artifacts: artifacts.filter(item => item.name === name) };
    }
    throw new Error(`Unexpected candidate fixture API route: ${path}`);
  };
  const list = async (path, field) => {
    const data = await get(path);
    return field ? data[field] : data;
  };
  return { candidateSha, sourceSha, tree, repository, pr, commits, checks, runs, publisher,
    artifacts, event, api: { get, list }, result: { sourceSha, tree, publisherRunId: publisher.id } };
}
