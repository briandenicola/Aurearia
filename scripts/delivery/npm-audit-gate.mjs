import { readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

// npm audit gate with reviewed, expiring exceptions. npm audit cannot ignore a
// single advisory, so this reads `npm audit --json` and fails on every high or
// critical finding unless each advisory behind it is listed in the exceptions
// file, for that package, and not yet expired. An expired exception fails
// again; an exception nothing matches any more is reported so it can be removed.

const BLOCKING = new Set(['high', 'critical']);
const FIELDS = ['advisory', 'package', 'reason', 'approvedBy', 'expires'];

function advisoryId(url) {
  return String(url ?? '').split('/').pop();
}

// validExceptions checks the exceptions file and returns its entries.
export function validExceptions(file) {
  const list = file?.exceptions;
  if (!Array.isArray(list)) throw new Error('exceptions file must have an "exceptions" array');
  for (const [i, e] of list.entries()) {
    for (const field of FIELDS) {
      if (typeof e?.[field] !== 'string' || e[field].trim() === '') {
        throw new Error(`exception ${i + 1} needs a non-empty "${field}"`);
      }
    }
    if (!/^GHSA(-[23456789cfghjmpqrvwx]{4}){3}$/.test(e.advisory)) throw new Error(`exception ${i + 1}: "${e.advisory}" is not a GHSA id`);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(e.expires) || Number.isNaN(Date.parse(`${e.expires}T00:00:00Z`))) {
      throw new Error(`exception ${i + 1}: expires must be a date like 2027-01-31`);
    }
  }
  return list;
}

// evaluate decides the gate from an `npm audit --json` report.
export function evaluate(report, exceptions, today = new Date()) {
  const vulnerabilities = report?.vulnerabilities;
  if (!vulnerabilities || typeof vulnerabilities !== 'object') {
    const reason = report?.error?.summary ?? report?.message ?? 'no vulnerabilities section';
    return { ok: false, failures: [`npm audit gave no usable report: ${reason}`], excepted: [], unused: [] };
  }
  const day = today.toISOString().slice(0, 10);
  const roots = (name, seen = new Set()) => {
    const v = vulnerabilities[name];
    if (!v || seen.has(name)) return [];
    seen.add(name);
    return (v.via ?? []).flatMap((via) =>
      typeof via === 'string' ? roots(via, seen) : [{ id: advisoryId(via.url), pkg: via.name, title: via.title, severity: via.severity }],
    );
  };
  const failures = [];
  const excepted = new Map();
  const matched = new Set();
  for (const [name, v] of Object.entries(vulnerabilities)) {
    if (!BLOCKING.has(v.severity)) continue;
    const found = roots(name);
    const open = [];
    for (const r of found) {
      const e = exceptions.find((x) => x.advisory === r.id && x.package === r.pkg);
      if (!e) {
        open.push(`${r.pkg} ${r.id} (${r.severity}): ${r.title}`);
        continue;
      }
      matched.add(e);
      if (e.expires < day) open.push(`${r.pkg} ${r.id}: its exception expired on ${e.expires}; fix it or review the exception again`);
      else excepted.set(`${r.pkg} ${r.id}`, e);
    }
    if (found.length === 0) open.push(`${name} (${v.severity}) with no advisory listed`);
    if (open.length) failures.push(`${name} (${v.severity}): ${[...new Set(open)].join('; ')}`);
  }
  const unused = exceptions.filter((e) => !matched.has(e)).map((e) => `${e.package} ${e.advisory}`);
  return { ok: failures.length === 0, failures, excepted: [...excepted.entries()], unused };
}

function main(exceptionsPath) {
  if (!exceptionsPath) {
    console.error('usage: node npm-audit-gate.mjs <exceptions.json>   (run in the package directory)');
    return 2;
  }
  const exceptions = validExceptions(JSON.parse(readFileSync(exceptionsPath, 'utf8')));
  const audit = spawnSync('npm', ['audit', '--json'], { encoding: 'utf8', shell: process.platform === 'win32', maxBuffer: 64 * 1024 * 1024 });
  let report;
  try {
    report = JSON.parse(audit.stdout);
  } catch {
    console.error(`npm audit did not return JSON (exit ${audit.status}):\n${audit.stderr ?? ''}`);
    return 1;
  }
  const result = evaluate(report, exceptions);
  for (const [finding, e] of result.excepted) {
    console.log(`excepted until ${e.expires}: ${finding} (${e.reason} Approved: ${e.approvedBy})`);
  }
  for (const u of result.unused) console.log(`exception no longer needed, remove it: ${u}`);
  if (!result.ok) {
    console.error('High or critical npm advisories:');
    for (const f of result.failures) console.error(`  ${f}`);
    return 1;
  }
  console.log('npm audit gate passed: no high or critical advisory outside a reviewed exception.');
  return 0;
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  process.exit(main(process.argv[2]));
}
