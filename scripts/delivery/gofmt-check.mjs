import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

// Lists unformatted Go files under cwd using the gofmt that ships with the local
// toolchain, so the gate never downloads a toolchain and works without gofmt on PATH.
export function unformattedGoFiles(cwd, { run = spawnSync, platform = process.platform } = {}) {
  const env = { ...process.env, GOTOOLCHAIN: 'local', GOPROXY: 'off' };
  const invoke = (command, args) => {
    const result = run(command, args, { cwd, encoding: 'utf8', env });
    if (result.error) throw new Error(`${command}: ${result.error.message}`);
    if (result.status !== 0) throw new Error(`${command} ${args.join(' ')} failed (${result.status}):\n${result.stdout ?? ''}${result.stderr ?? ''}`);
    return result.stdout ?? '';
  };
  const goroot = invoke('go', ['env', 'GOROOT']).trim();
  if (!goroot) throw new Error('go env GOROOT returned nothing');
  const gofmt = join(goroot, 'bin', platform === 'win32' ? 'gofmt.exe' : 'gofmt');
  return invoke(gofmt, ['-l', '.']).split(/\r?\n/).map(line => line.trim()).filter(Boolean);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const files = unformattedGoFiles(process.cwd());
    if (files.length) {
      console.error(`gofmt required (run gofmt -w on these files):\n${files.join('\n')}`);
      process.exit(1);
    }
  } catch (error) {
    console.error(`gofmt check failed: ${error.message}`);
    process.exit(1);
  }
}
