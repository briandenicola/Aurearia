import { readFileSync, writeFileSync, copyFileSync, existsSync } from 'node:fs';
import { join, resolve, delimiter } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

export function versionAnnotation(source, version) {
  if (!/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$/.test(version)) {
    throw new Error('VERSION must contain one semantic version');
  }
  const pattern = /^(\/\/[ \t]+@version[ \t]+)\S+/gm;
  if ([...source.matchAll(pattern)].length !== 1) {
    throw new Error('Expected exactly one @version annotation in src/api/main.go');
  }
  return source.replace(pattern, (_match, prefix) => prefix + version);
}

export function generate(root, { check = false, run = spawnSync } = {}) {
  const api = join(root, 'src', 'api');
  const invoke = (command, args, cwd = root) => {
    const result = run(command, args, {
      cwd, encoding: 'utf8',
      env: { ...process.env, GOTOOLCHAIN: 'local', GOPROXY: 'off' },
    });
    if (result.error) throw new Error(`${command}: ${result.error.message}`);
    if (result.status !== 0) {
      throw new Error(`${command} ${args.join(' ')} failed (${result.status}):\n${result.stdout ?? ''}${result.stderr ?? ''}`);
    }
    return result.stdout ?? '';
  };
  const executable = process.platform === 'win32' ? 'swag.exe' : 'swag';
  const gobin = invoke('go', ['env', 'GOBIN']).trim();
  const gopath = invoke('go', ['env', 'GOPATH']).trim().split(delimiter)[0];
  const swag = join(gobin || join(gopath, 'bin'), executable);
  if (!existsSync(swag)) throw new Error('Missing swag; authorize task setup:openapi first');
  // The v1.16.6 module still advertises v1.16.4 in its CLI banner.
  if (!/^\s*mod\s+github\.com\/swaggo\/swag\s+v1\.16\.6(?:\s|$)/m.test(invoke('go', ['version', '-m', swag]))) {
    throw new Error('Expected swag v1.16.6; authorize task setup:openapi to reconcile');
  }
  const main = join(api, 'main.go');
  const version = readFileSync(join(root, 'VERSION'), 'utf8').trim();
  writeFileSync(main, versionAnnotation(readFileSync(main, 'utf8'), version));
  invoke(swag, ['init', '-g', 'main.go', '-o', './docs', '--parseDependency', '--parseInternal'], api);
  const snapshot = join(api, 'docs', 'swagger.json');
  JSON.parse(readFileSync(snapshot, 'utf8'));
  copyFileSync(snapshot, join(root, 'docs', 'openapi.json'));
  if (check) {
    invoke('git', ['diff', '--exit-code', '--', 'src/api/main.go',
      'src/api/docs/docs.go', 'src/api/docs/swagger.json',
      'src/api/docs/swagger.yaml', 'docs/openapi.json']);
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (process.argv.slice(2).some(arg => arg !== '--check')) throw new Error('Usage: openapi.mjs [--check]');
    generate(fileURLToPath(new URL('../..', import.meta.url)), { check: process.argv.includes('--check') });
    console.log('OpenAPI regenerated; no tools or dependencies installed.');
  } catch (error) {
    console.error(`OpenAPI: ${error.message}`);
    process.exitCode = 1;
  }
}
