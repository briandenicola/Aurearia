import assert from 'node:assert/strict';
import { test } from 'node:test';
import { changedAreas, formatOutputs, FILE_LIST_LIMIT } from './changed-areas.mjs';

const everything = { api: true, web: true, agent: true, image: true };

test('pushes, schedules and manual runs always run everything', () => {
  for (const event of ['push', 'schedule', 'workflow_dispatch', '']) {
    assert.deepEqual(changedAreas(['docs/readme.md'], event), everything);
  }
});

test('pull requests run only the areas their files touch', () => {
  assert.deepEqual(changedAreas(['src/web/src/App.vue'], 'pull_request'),
    { api: false, web: true, agent: false, image: false });
  assert.deepEqual(changedAreas(['src/api/main.go', 'src/agent/app.py'], 'pull_request'),
    { api: true, web: false, agent: true, image: false });
  assert.deepEqual(changedAreas(['docs/openapi.json'], 'pull_request'),
    { api: true, web: true, agent: false, image: false });
  assert.deepEqual(changedAreas(['Dockerfile'], 'pull_request'),
    { api: false, web: false, agent: false, image: true });
});

test('documentation-only pull requests run no area jobs', () => {
  assert.deepEqual(changedAreas(['docs/features/coin-sets.md', 'README.md', 'specs/1/plan.md'], 'pull_request'),
    { api: false, web: false, agent: false, image: false });
});

test('CI, build tooling, unknown paths and unreadable file lists run everything', () => {
  for (const file of ['.github/workflows/ci.yml', 'Taskfile.yml', 'scripts/delivery/x.mjs', 'VERSION', 'new-root-file.json', 'src/new-service/main.rs']) {
    assert.deepEqual(changedAreas(['src/web/a.ts', file], 'pull_request'), everything, file);
  }
  assert.deepEqual(changedAreas([], 'pull_request'), everything);
  assert.deepEqual(changedAreas(Array(FILE_LIST_LIMIT).fill('docs/a.md'), 'pull_request'), everything);
});

test('outputs are written in GITHUB_OUTPUT format', () => {
  assert.equal(formatOutputs({ api: true, web: false, agent: false, image: true }),
    'api=true\nweb=false\nagent=false\nimage=true\n');
});
