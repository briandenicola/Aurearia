// Decides which parts of CI a pull request needs to run, from its changed files.
//
// Only pull requests are narrowed. Pushes to beta always run every job,
// because promotion to main (verify-candidate.mjs) requires every check to have
// passed on the exact beta commit.
//
// Fails closed: a path that matches no rule below runs everything.
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';

export const AREAS = ['api', 'web', 'agent', 'image'];

const AREA_RULES = {
  api: [/^src\/api\//, /^docs\/openapi\.json$/],
  web: [/^src\/web\//, /^docs\/openapi\.json$/],
  agent: [/^src\/agent\//],
  image: [/^Dockerfile$/, /^\.dockerignore$/, /^docker-compose[^/]*\.ya?ml$/],
};

// Paths that cannot change any build, test or image result.
const NO_CODE_RULES = [
  /^docs\//,
  /^specs\//,
  /^\.squad\//,
  /^\.specify\//,
  /^\.claude\//,
  /^\.vscode\//,
  /^\.devcontainer\//,
  /^[^/]+\.md$/,
  /^LICENSE$/,
  /^\.squad-phase2-commit-msg\.txt$/,
];

// GitHub's pull request files API stops listing at 3000 files.
export const FILE_LIST_LIMIT = 3000;

const all = () => Object.fromEntries(AREAS.map(area => [area, true]));

export function changedAreas(files, eventName) {
  if (eventName !== 'pull_request') return all();
  if (files.length === 0 || files.length >= FILE_LIST_LIMIT) return all();
  const result = Object.fromEntries(AREAS.map(area => [area, false]));
  for (const file of files) {
    const areas = AREAS.filter(area => AREA_RULES[area].some(rule => rule.test(file)));
    if (areas.length) {
      for (const area of areas) result[area] = true;
    } else if (!NO_CODE_RULES.some(rule => rule.test(file))) {
      return all();
    }
  }
  return result;
}

export function formatOutputs(areas) {
  return AREAS.map(area => `${area}=${areas[area]}`).join('\n') + '\n';
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const eventName = process.env.EVENT_NAME ?? '';
  const files = eventName === 'pull_request'
    ? readFileSync(0, 'utf8').split('\n').map(line => line.trim()).filter(Boolean)
    : [];
  const areas = changedAreas(files, eventName);
  process.stdout.write(formatOutputs(areas));
  console.error(`Changed files: ${files.length}; areas: ${JSON.stringify(areas)}`);
}
