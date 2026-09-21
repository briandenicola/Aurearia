import { existsSync, readFileSync, readdirSync, lstatSync, realpathSync } from 'node:fs';
import { resolve, join, dirname, relative, isAbsolute, basename } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

export const activeFiles = [
  '.github/copilot-instructions.md', '.github/pull_request_template.md',
  'CONTRIBUTING.md', 'docs/testing.md', '.squad/routing.md',
  '.squad/ceremonies.md', '.squad/decisions.md', '.squad/identity/now.md',
  '.github/agents/aurearia-reviewer.agent.md', 'docs/agentic-native-integration.md',
];
const activeDirectories = ['.github/agents', '.github/prompts', '.github/instructions', '.specify/templates'];
const skillDirectories = ['.github/skills', '.agents/skills'];
const constitution = '.specify/memory/constitution.md';
const currentWork = '.squad/identity/now.md';
const posix = path => path.replaceAll('\\', '/');
const placeholder = value => /[<>{}*[\]]|(?:^|\/)(?:NNN|NNNN|F0NN)(?:[-/.]|$)/.test(value);

// Only required scalar fields are interpreted; this is not a general YAML linter.
export function metadata(text, required) {
  const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
  if (lines[0] !== '---') throw new Error('Missing opening YAML frontmatter delimiter');
  const end = lines.indexOf('---', 1);
  if (end < 0) throw new Error('Missing closing YAML frontmatter delimiter');
  const values = {};
  const fields = new Map();
  for (let i = 1; i < end; i++) {
    if (!lines[i].trim() || /^\s*#/.test(lines[i]) || /^\s/.test(lines[i])) continue;
    const match = lines[i].match(/^([a-z][a-z0-9_-]*):(?:\s+(.*))?$/i);
    if (!match) throw new Error(`Malformed frontmatter at line ${i + 1}`);
    if (fields.has(match[1])) throw new Error(`Duplicate field ${match[1]} at line ${i + 1}`);
    fields.set(match[1], i);
  }
  for (const key of required) {
    const index = fields.get(key);
    if (index === undefined) throw new Error(`Missing required field ${key}`);
    let value = lines[index].slice(lines[index].indexOf(':') + 1).trim();
    if (/^[|>][-+]?$/.test(value)) {
      const parts = [];
      for (let i = index + 1; i < end && (!lines[i].trim() || /^\s/.test(lines[i])); i++) parts.push(lines[i].trim());
      value = parts.join(' ').trim();
    } else if (value.startsWith('"')) {
      value = JSON.parse(value);
    } else if (value.startsWith("'")) {
      if (!/^'(?:[^']|'')*'$/.test(value)) throw new Error(`Malformed quoted field ${key}`);
      value = value.slice(1, -1).replaceAll("''", "'");
    } else {
      value = value.replace(/\s+#.*$/, '').trim();
      if (/^(?:null|true|false|~|\d+)$|^[{[&*!]/i.test(value)) {
        throw new Error(`${key} must be a string scalar`);
      }
    }
    if (typeof value !== 'string' || !value.trim()) throw new Error(`Empty or invalid field ${key}`);
    values[key] = value;
  }
  return values;
}

export function instructionPatterns(text) {
  const patterns = metadata(text, ['applyTo']).applyTo.split(',').map(value => value.trim());
  if (new Set(patterns).size !== patterns.length || patterns.some(pattern =>
    !/^[a-zA-Z0-9_.*-]+(?:\/[a-zA-Z0-9_.*-]+)*$/.test(pattern) ||
    pattern.split('/').some(part => part === '.' || part === '..'))) {
    throw new Error('applyTo must contain unique repository-relative globs using the supported literal/*/** subset');
  }
  return patterns;
}

export function checkReviewer(text) {
  const values = metadata(text, ['name', 'description']);
  const lines = text.replace(/^\uFEFF/, '').split(/\r?\n/);
  const header = lines.slice(1, lines.indexOf('---', 1)).filter(line => line.trim());
  if (values.name !== 'aurearia-reviewer' || header.length !== 3 ||
      header.some(line => !/^(?:name|description|tools):\s/.test(line))) {
    throw new Error('Reviewer permits only name, description and tools metadata; no model or additional capabilities');
  }
  const tools = JSON.parse(header.find(line => line.startsWith('tools:'))?.slice(6).trim() ?? 'null');
  if (!Array.isArray(tools) || tools.length !== 2 || !tools.includes('read') || !tools.includes('search')) {
    throw new Error('Reviewer tools must be exactly ["read", "search"]');
  }
}

export function checkGovernance(root) {
  root = realpathSync(root);
  const diagnostics = [];
  const report = (file, line, code, message, severity = 'error') =>
    diagnostics.push({ file, line, severity, code, message });
  const local = file => resolve(root, ...posix(file).split('/'));
  const sparseFiles = new Set();
  if (existsSync(join(root, '.git'))) {
    const result = spawnSync('git', ['ls-files', '-t', '-z'], { cwd: root, encoding: 'utf8' });
    if (result.error || result.status !== 0) throw new Error(`Cannot inspect sparse references: ${result.error?.message ?? result.stderr}`);
    for (const entry of result.stdout.split('\0')) if (entry.startsWith('S ')) sparseFiles.add(entry.slice(2));
  }
  const inside = target => {
    const rel = relative(root, target);
    return rel !== '..' && !rel.startsWith(`..${process.platform === 'win32' ? '\\' : '/'}`) && !isAbsolute(rel);
  };
  const present = file => {
    const target = local(file);
    return inside(target) && existsSync(target) && inside(realpathSync(target));
  };
  const read = file => {
    if (!present(file)) {
      report(file, 1, 'FILE', 'Required active file is missing or outside the repository');
      return '';
    }
    return readFileSync(local(file), 'utf8');
  };
  const walk = directory => {
    if (!present(directory)) return [];
    return readdirSync(local(directory), { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name)).flatMap(entry => {
      const file = `${directory}/${entry.name}`;
      if (entry.isSymbolicLink()) {
        report(file, 1, 'FILE', 'Active governance discovery does not follow symlinks');
        return [];
      }
      return entry.isDirectory() ? walk(file) : [file];
    });
  };
  const policy = read(constitution);
  const identifiers = new Set([...policy.matchAll(/^### ([IVXLCDM]+)\. /gm)].map(match => match[1]));
  if (!identifiers.size) report(constitution, 1, 'PRINCIPLE', 'No supported principle identifiers found');
  const roles = walk('.squad/agents').filter(file => /\/(?:charter|history)\.md$/.test(file));
  const skills = [];
  for (const directory of skillDirectories) {
    if (!present(directory)) continue;
    for (const entry of readdirSync(local(directory), { withFileTypes: true })) {
      if (entry.name.startsWith('.')) continue;
      if (!entry.isDirectory() && !entry.isSymbolicLink()) continue;
      const skill = `${directory}/${entry.name}/SKILL.md`;
      if (entry.isSymbolicLink() || !entry.isDirectory() || !present(skill)) {
        report(`${directory}/${entry.name}`, 1, 'SKILL', 'Native skill packages require a directory containing SKILL.md; symlinks are not followed');
      } else skills.push(skill);
    }
  }
  const files = [...new Set([...activeFiles, ...activeDirectories.flatMap(walk).filter(file => file.endsWith('.md')), ...roles, ...skills])].sort();
  const contents = new Map();
  function reference(file, line, target) {
    if (placeholder(target) || /^(?:[a-z]+:|#)/i.test(target)) return;
    let decoded;
    try { decoded = decodeURIComponent(target.split('#')[0]); }
    catch { report(file, line, 'REFERENCE', `Invalid encoded reference: ${target}`); return; }
    if (!decoded) return;
    const resolved = resolve(dirname(local(file)), posix(decoded));
    const relativeTarget = posix(relative(root, resolved));
    if (inside(resolved) && sparseFiles.has(relativeTarget) && !existsSync(resolved)) {
      report(file, line, 'SPARSE', `Target is tracked but sparse-excluded: ${target}`, 'warning');
    } else if (!inside(resolved) || !present(relativeTarget)) {
      report(file, line, 'REFERENCE', `Missing or out-of-repository target: ${target}`);
    }
  }
  for (const file of files) {
    const text = read(file);
    contents.set(file, text);
    if (file.startsWith('.github/instructions/') && file.endsWith('.instructions.md')) {
      try { instructionPatterns(text); }
      catch (error) { report(file, 1, 'INSTRUCTION', error.message); }
    }
    if (file === '.github/agents/aurearia-reviewer.agent.md') {
      try { checkReviewer(text); }
      catch (error) { report(file, 1, 'REVIEWER', error.message); }
    }
    let fence;
    text.split(/\r?\n/).forEach((line, index) => {
      const marker = line.match(/^\s{0,3}(`{3,}|~{3,})/);
      if (marker) {
        if (!fence) fence = marker[1];
        else if (marker[1][0] === fence[0] && marker[1].length >= fence.length) fence = undefined;
        return;
      }
      if (fence || /^\s*<!--/.test(line)) return;
      const pattern = /\bPrinciples?\s+((?:[IVXLCDM]+|\d+)(?:(?:\s*[,/–-]\s*|\s+(?:and|through|to)\s+)(?:[IVXLCDM]+|\d+))*)\b/g;
      for (const match of line.matchAll(pattern)) {
        for (const id of match[1].match(/[IVXLCDM]+|\d+/g)) {
          if (!identifiers.has(id)) report(file, index + 1, 'PRINCIPLE', `Unsupported principle identifier ${id}`);
        }
      }
      for (const match of line.matchAll(/\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g)) reference(file, index + 1, match[1]);
      const definition = line.match(/^\s*\[[^\]]+\]:\s*(\S+)/);
      if (definition) reference(file, index + 1, definition[1]);
    });
  }

  const indexFile = 'docs/adr/README.md';
  const indexText = read(indexFile);
  const entries = new Map();
  const status = text => text.trim().match(/^(Proposed|Accepted|Deprecated|Superseded)\b/i)?.[1].toLowerCase();
  indexText.split(/\r?\n/).forEach((line, i) => {
    const match = line.match(/^\|\s*(\d{4})\s*\|\s*\[[^\]]+\]\(([^)]+)\)\s*\|[^|]*\|\s*([^|]+)\|/);
    if (!match) return;
    const [, id, target, label] = match;
    if (entries.has(id)) report(indexFile, i + 1, 'ADR', `Duplicate ADR ${id}`);
    entries.set(id, { target, label, line: i + 1 });
    if (!target.startsWith(`${id}-`) || target.includes('/') || target.includes('\\') || !present(`docs/adr/${target}`)) {
      report(indexFile, i + 1, 'ADR', `Invalid ADR ${id} index target: ${target}`);
    }
  });
  for (const file of walk('docs/adr').filter(file => /\/\d{4}-[^/]+\.md$/.test(file))) {
    const id = basename(file).slice(0, 4);
    const text = read(file);
    const header = text.split(/^## (?!Status\b)/m)[0];
    const lines = header.split(/\r?\n/);
    let position = lines.findIndex(line => /^\s*(?:-\s*)?Status:/i.test(line.replaceAll('*', '')));
    let label = position < 0 ? '' : lines[position].replaceAll('*', '').replace(/^\s*(?:-\s*)?Status:\s*/i, '');
    if (position < 0) {
      const section = lines.findIndex(line => /^## Status\s*$/i.test(line));
      if (section >= 0) {
        position = lines.findIndex((line, i) => i > section && line.trim());
        label = lines[position] ?? '';
      }
    }
    const entry = entries.get(id);
    if (!entry || entry.target !== basename(file) || !status(label) || status(label) !== status(entry.label)) {
      report(file, Math.max(1, position + 1), 'ADR', 'ADR header and index must have the same valid status and target');
    }
    for (const match of header.matchAll(/\[[^\]]*\]\(([^)\s]+)\)/g)) reference(file, header.slice(0, match.index).split('\n').length, match[1]);
    for (const match of header.matchAll(/(?:Superseded by|Partial supersession:|Supersedes:)[^\n]*?\b(?:ADR )?(\d{4})\b/gi)) {
      if (!entries.has(match[1])) report(file, 1, 'ADR', `Unknown supersession ADR ${match[1]}`);
    }
    if (status(label) === 'proposed') {
      report(file, Math.max(1, position + 1), 'LIFECYCLE', 'Proposed is not accepted; approval history requires human review', 'warning');
    }
  }

  try {
    const state = metadata(contents.get(currentWork) ?? '', ['updated_at', 'focus_area', 'owner', 'work_artifact', 'tasks_artifact']);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(state.updated_at) ||
        !Number.isFinite(Date.parse(state.updated_at)) ||
        new Date(state.updated_at).toISOString().slice(0, 10) !== state.updated_at) {
      report(currentWork, 2, 'STATE', 'updated_at must be a real YYYY-MM-DD date');
    }
    for (const field of ['work_artifact', 'tasks_artifact']) {
      const value = state[field];
      if (/^https:\/\/github\.com\/briandenicola\/Aurearia\/issues\/[1-9]\d*$/.test(value)) continue;
      if (placeholder(value) || !/^(?:docs|specs)\//.test(value) ||
          posix(relative(root, local(value))) !== value ||
          !value.endsWith('.md') || !present(value) || !lstatSync(local(value)).isFile()) {
        report(currentWork, 1, 'STATE', `${field} must name an existing canonical docs/specs Markdown path or repository issue URL`);
      }
    }
    if (state.work_artifact.startsWith('specs/') &&
        (!/^specs\/[^/]+\/spec\.md$/.test(state.work_artifact) ||
         state.tasks_artifact !== state.work_artifact.replace(/spec\.md$/, 'tasks.md'))) {
      report(currentWork, 1, 'STATE', 'Selected feature requires spec.md and matching tasks.md in the same directory');
    }
  } catch (error) { report(currentWork, 1, 'STATE', error.message); }

  for (const file of skills) {
    try {
      const skill = metadata(contents.get(file), ['name', 'description']);
      if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(skill.name) || skill.name.length > 64 || skill.name !== basename(dirname(local(file)))) {
        report(file, 2, 'SKILL', 'name must match its directory and be a lowercase kebab-case identifier (max 64 characters)');
      }
      if (skill.description.length > 1024) report(file, 3, 'SKILL', 'description exceeds 1024 characters');
    } catch (error) { report(file, 1, 'SKILL', error.message); }
  }
  const countLines = text => text.trimEnd().split(/\r?\n/).length;
  for (const [file, text] of contents) {
    const limit = file === '.squad/decisions.md' ? 20 * 1024 : file.endsWith('/history.md') ? 12 * 1024 : undefined;
    if (limit && Buffer.byteLength(text) > limit) report(file, 1, 'BUDGET', `Context exceeds ${limit} bytes; preserve evidence before curation`, 'warning');
    const lines = file === currentWork ? 100 : file === '.github/copilot-instructions.md' ? 150 : undefined;
    if (lines && countLines(text) > lines) report(file, 1, 'BUDGET', `Context exceeds ${lines} lines`, 'warning');
  }
  const words = [policy, ...['.github/copilot-instructions.md', '.squad/decisions.md', currentWork].map(file => contents.get(file) ?? '')]
    .reduce((total, text) => total + (text.match(/\S+/g)?.length ?? 0), 0);
  if (words > 6000) report(currentWork, 1, 'BUDGET', `Conservative initial context is ${words} words (warning budget 6000)`, 'warning');
  return diagnostics.sort((a, b) => a.file.localeCompare(b.file) || a.line - b.line || a.code.localeCompare(b.code) || a.message.localeCompare(b.message));
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const args = process.argv.slice(2);
    if (args.length && (args.length !== 2 || args[0] !== '--root')) throw new Error('Usage: check-governance.mjs [--root path]');
    const diagnostics = checkGovernance(args[1] ?? fileURLToPath(new URL('../..', import.meta.url)));
    for (const item of diagnostics) console.log(`${item.file}:${item.line}: ${item.severity} ${item.code}: ${item.message}`);
    const errors = diagnostics.filter(item => item.severity === 'error').length;
    console.log(`Governance: ${errors} error(s), ${diagnostics.length - errors} warning(s). Read-only/offline; no review or acceptance inferred.`);
    process.exitCode = errors ? 1 : 0;
  } catch (error) {
    console.error(`scripts/delivery/check-governance.mjs:1: error INPUT: ${error.message}`);
    process.exitCode = 1;
  }
}
