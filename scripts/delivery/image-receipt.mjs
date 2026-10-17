import { readFileSync, writeFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { requireEvidence } from './github-evidence.mjs';

export function assertImageReceipt(receipt, { sourceSha, tree, kind }) {
  requireEvidence(/^[a-f0-9]{40}$/.test(sourceSha) && /^[a-f0-9]{40}$/.test(tree) &&
    ['app', 'agent'].includes(kind), 'Invalid expected image source');
  requireEvidence(receipt?.sourceSha === sourceSha && receipt.tree === tree && receipt.kind === kind &&
    /^sha256:[a-f0-9]{64}$/.test(receipt.digest), 'Image receipt does not match the verified source and image kind');
  return receipt.digest;
}

export function promoteImage(receipt, expected, image, candidateSha, run = spawnSync) {
  const digest = assertImageReceipt(receipt, expected);
  requireEvidence(/^[a-z0-9]+(?:[._-][a-z0-9]+)*\/[a-z0-9]+(?:[._-][a-z0-9]+)*$/.test(image) &&
    /^[a-f0-9]{40}$/.test(candidateSha), 'Invalid promotion image or candidate');
  const result = run('docker', ['buildx', 'imagetools', 'create', '--prefer-index=false',
    '--tag', `${image}:sha-${candidateSha}`, '--tag', `${image}:sha-${candidateSha.slice(0, 7)}`,
    '--tag', `${image}:latest`, `${image}@${digest}`], { stdio: 'inherit', shell: false });
  if (result.error) throw result.error;
  requireEvidence(result.status === 0, `Image promotion failed (${result.status})`);
  return digest;
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const [mode, kind, path] = process.argv.slice(2);
    const sourceSha = process.env.SOURCE_SHA;
    const tree = process.env.SOURCE_TREE;
    if (mode === 'write') {
      const receipt = { sourceSha, tree, kind, digest: process.env.IMAGE_DIGEST };
      assertImageReceipt(receipt, { sourceSha, tree, kind });
      writeFileSync(path, `${JSON.stringify(receipt)}\n`);
    } else {
      requireEvidence(mode === 'promote', 'Expected image receipt write or promote mode');
      const receipt = JSON.parse(readFileSync(path, 'utf8'));
      const digest = promoteImage(receipt, { sourceSha, tree, kind }, process.env.TARGET_IMAGE, process.env.CANDIDATE_SHA);
      console.log(`Promoted verified ${kind} digest ${digest}; no image rebuilt.`);
    }
  } catch (error) {
    console.error(`Image receipt/promotion blocked: ${error.message}`);
    process.exitCode = 1;
  }
}
