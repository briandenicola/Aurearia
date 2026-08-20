// Compares the compiled render function of each given .vue file against its
// committed version, to prove a template reformat is semantically inert.
//
// Vite's production build is not hash-stable across runs, so diffing dist
// output cannot answer this question. Compiling the SFC templates directly can.
//
// Usage, from anywhere in the repo:
//   node src/web/scripts/compare-template-output.mjs <path-from-repo-root.vue> [...]

import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { parse, compileTemplate } from 'vue/compiler-sfc'

const repoRoot = execFileSync('git', ['rev-parse', '--show-toplevel'], {
  encoding: 'utf8',
}).trim()

function compiledRender(rawSource, filename) {
  // `git show HEAD:<file>` yields the repository's LF content while the working
  // tree is CRLF under core.autocrlf=true. Without normalising, every
  // multi-line expression inside an attribute reports a spurious \r difference.
  const source = rawSource.replace(/\r\n/g, '\n')
  const { descriptor, errors } = parse(source, { filename })
  if (errors.length) throw new Error(`${filename}: parse failed: ${errors[0].message}`)
  if (!descriptor.template) return '<no template block>'

  const { code, errors: compileErrors } = compileTemplate({
    source: descriptor.template.content,
    filename,
    id: 'compare',
    // Vue's default. Explicit so this comparison cannot silently drift from
    // what the build actually does.
    compilerOptions: { whitespace: 'condense' },
  })
  if (compileErrors.length) throw new Error(`${filename}: compile failed: ${compileErrors[0]}`)
  return code
}

const files = process.argv.slice(2)
if (!files.length) {
  console.error('usage: node src/web/scripts/compare-template-output.mjs <file.vue> [...]')
  process.exit(2)
}

let same = 0
let differing = 0
let skipped = 0

for (const file of files) {
  let committed
  try {
    committed = execFileSync('git', ['show', `HEAD:${file}`], {
      encoding: 'utf8',
      cwd: repoRoot,
      maxBuffer: 32 * 1024 * 1024,
      stdio: ['ignore', 'pipe', 'ignore'],
    })
  } catch {
    // A file absent from HEAD cannot be compared. Counted separately so a
    // path mistake can never masquerade as a clean result.
    console.log(`SKIP    ${file} (not found in HEAD)`)
    skipped++
    continue
  }

  const current = readFileSync(join(repoRoot, file), 'utf8')
  const before = compiledRender(committed, file)
  const after = compiledRender(current, file)

  if (before === after) {
    console.log(`SAME    ${file}`)
    same++
  } else {
    console.log(`DIFFERS ${file}`)
    differing++
    const b = before.split('\n')
    const a = after.split('\n')
    for (let i = 0; i < Math.max(b.length, a.length); i++) {
      if (b[i] !== a[i]) {
        console.log(`          line ${i + 1}`)
        console.log(`          before: ${JSON.stringify(b[i])}`)
        console.log(`          after:  ${JSON.stringify(a[i])}`)
      }
    }
  }
}

console.log(`\ncompared ${same} · differing ${differing} · skipped ${skipped}`)

if (skipped > 0) {
  console.log('FAILED: some files could not be compared, so this run proves nothing.')
  process.exit(2)
}
if (differing > 0) {
  console.log('FAILED: compiled render output changed.')
  process.exit(1)
}
console.log(`PASSED: all ${same} templates compile to identical render functions.`)
