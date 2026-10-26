import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { evaluate, validExceptions } from './npm-audit-gate.mjs';

const zodException = {
  advisory: 'GHSA-mrhx-c7xc-m3w5', package: 'zod', reason: 'imgly needs zod 3', approvedBy: 'owner', expires: '2027-01-31',
};

// The shape npm 10 prints for the zod advisory reached through imgly.
function report(extra = {}) {
  return {
    vulnerabilities: {
      zod: {
        name: 'zod', severity: 'high',
        via: [{ source: 1, name: 'zod', title: 'Zod can exhaust memory', url: 'https://github.com/advisories/GHSA-mrhx-c7xc-m3w5', severity: 'high', range: '<4.6.6' }],
      },
      '@imgly/background-removal': { name: '@imgly/background-removal', severity: 'high', via: ['zod'] },
      ...extra,
    },
  };
}

const today = new Date('2026-10-26T12:00:00Z');

test('a reviewed exception covers the advisory and the package that depends on it', () => {
  const r = evaluate(report(), [zodException], today);
  assert.equal(r.ok, true, r.failures.join('\n'));
  assert.deepEqual(r.excepted.map(([k]) => k), ['zod GHSA-mrhx-c7xc-m3w5']);
  assert.deepEqual(r.unused, []);
});

test('without the exception the same report fails', () => {
  const r = evaluate(report(), [], today);
  assert.equal(r.ok, false);
  assert.equal(r.failures.length, 2);
});

test('any other high or critical advisory still fails', () => {
  const r = evaluate(report({
    dompurify: { name: 'dompurify', severity: 'critical', via: [{ name: 'dompurify', title: 'bypass', url: 'https://github.com/advisories/GHSA-cg2g-xh64-v4q7', severity: 'critical' }] },
  }), [zodException], today);
  assert.equal(r.ok, false);
  assert.match(r.failures.join('\n'), /dompurify .*GHSA-cg2g-xh64-v4q7/);
});

test('moderate and low findings do not block', () => {
  const r = evaluate({ vulnerabilities: { x: { name: 'x', severity: 'moderate', via: [{ name: 'x', url: 'u/GHSA-xxxx-xxxx-xxxx', severity: 'moderate' }] } } }, [], today);
  assert.equal(r.ok, true);
});

test('an exception is for one package: the same id elsewhere still fails', () => {
  const r = evaluate(report(), [{ ...zodException, package: 'not-zod' }], today);
  assert.equal(r.ok, false);
});

test('an expired exception fails again', () => {
  const r = evaluate(report(), [{ ...zodException, expires: '2026-10-25' }], today);
  assert.equal(r.ok, false);
  assert.match(r.failures.join('\n'), /expired on 2026-10-25/);
});

test('an exception nothing needs any more is reported, not silently kept', () => {
  const r = evaluate({ vulnerabilities: {} }, [zodException], today);
  assert.equal(r.ok, true);
  assert.deepEqual(r.unused, ['zod GHSA-mrhx-c7xc-m3w5']);
});

test('a report npm could not produce fails closed', () => {
  const r = evaluate({ error: { summary: 'network timeout' } }, [zodException], today);
  assert.equal(r.ok, false);
  assert.match(r.failures[0], /network timeout/);
});

test('exceptions must be complete and dated', () => {
  assert.throws(() => validExceptions({}), /"exceptions" array/);
  assert.throws(() => validExceptions({ exceptions: [{ ...zodException, reason: ' ' }] }), /"reason"/);
  assert.throws(() => validExceptions({ exceptions: [{ ...zodException, expires: 'soon' }] }), /expires/);
  assert.throws(() => validExceptions({ exceptions: [{ ...zodException, advisory: 'CVE-2026-1' }] }), /GHSA/);
});

test('the repository exceptions file is valid', () => {
  const file = fileURLToPath(new URL('../../src/web/audit-exceptions.json', import.meta.url));
  const list = validExceptions(JSON.parse(readFileSync(file, 'utf8')));
  assert.ok(list.length > 0);
});
