// Writes what Pele makes of the cases in ./cases as fixtures for limu's tests to compare with.
//
//   node --experimental-transform-types --no-warnings --import ./register.mjs run.mjs <pele> <limu> [suite…]
//
// scripts/oracle.sh runs it. Each suite is a module in ./suites whose default export is an
// async function given the helpers below; with no suites named, all of them run.
import { execFileSync } from 'node:child_process';
import { mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { gzipSync } from 'node:zlib';

const [pele, limu, ...only] = process.argv.slice(2);
if (!pele || !limu) {
  console.error('usage: run.mjs <pele> <limu> [suite…]');
  process.exit(2);
}

let commit = '';
try {
  commit = execFileSync('git', ['-C', pele, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
} catch {}
const { version } = JSON.parse(readFileSync(join(pele, 'package.json'), 'utf8'));
const origin = { version, commit };

const helpers = {
  pele,
  // Imports a module of Pele's sources, by its path under src.
  src: (path) => import(join(pele, 'src', path)),
  // Reads ./cases/<name>.json.
  cases: (name) => JSON.parse(readFileSync(new URL(`./cases/${name}.json`, import.meta.url), 'utf8')),
  // Writes a fixture, under limu, with the version of Pele it came from, gzipped at
  // path + '.gz'; internal/fixture reads it back.
  write(path, data) {
    const out = join(limu, path + '.gz');
    mkdirSync(dirname(out), { recursive: true });
    writeFileSync(out, gzipSync(JSON.stringify({ pele: origin, ...data }) + '\n', { level: 9 }));
    console.log(`wrote ${path}.gz`);
  },
  // A random number generator in [0, 1) that gives the same numbers for the same seed.
  random(seed) {
    let s = seed >>> 0;
    return () => (s = (Math.imul(s, 1664525) + 1013904223) >>> 0) / 2 ** 32;
  },
};

const dir = new URL('./suites/', import.meta.url);
const names = readdirSync(dir)
  .filter((f) => f.endsWith('.mjs'))
  .map((f) => f.slice(0, -4))
  .filter((name) => only.length === 0 || only.includes(name))
  .sort();
for (const name of names) {
  const suite = await import(new URL(`${name}.mjs`, dir));
  await suite.default(helpers);
}
