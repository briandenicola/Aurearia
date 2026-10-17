export const requireEvidence = (condition, message) => {
  if (!condition) throw new Error(message);
};

export function githubEvidence(env = process.env, fetchImpl = fetch) {
  requireEvidence(env.GITHUB_API_URL === 'https://api.github.com', 'Unexpected GitHub API origin');
  requireEvidence(/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(env.GITHUB_REPOSITORY ?? '') &&
    env.GITHUB_TOKEN, 'Missing GitHub repository or read token');
  const base = `${env.GITHUB_API_URL}/repos/${env.GITHUB_REPOSITORY}`;
  const get = async path => {
    requireEvidence(!path.startsWith('/') && !path.includes('..') && !path.includes('://'),
      'Unexpected GitHub evidence path');
    const response = await fetchImpl(`${base}/${path}`, {
      method: 'GET', redirect: 'error', signal: AbortSignal.timeout(15000),
      headers: { Accept: 'application/vnd.github+json', Authorization: `Bearer ${env.GITHUB_TOKEN}`,
        'X-GitHub-Api-Version': '2022-11-28' },
    });
    requireEvidence(response.ok, `GitHub evidence request failed (${response.status}): ${path}`);
    return response.json();
  };
  const list = async (path, field) => {
    const items = [];
    for (let page = 1; page <= 10; page++) {
      const data = await get(`${path}${path.includes('?') ? '&' : '?'}per_page=100&page=${page}`);
      const entries = field ? data?.[field] : data;
      requireEvidence(Array.isArray(entries), `Malformed GitHub list: ${field ?? path}`);
      items.push(...entries);
      if (entries.length < 100) return items;
    }
    throw new Error(`GitHub evidence exceeds the bounded pagination limit: ${field ?? path}`);
  };
  return { get, list };
}
