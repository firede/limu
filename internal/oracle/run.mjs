// Writes what Pele makes of the cases its suites hold as fixtures for limu's tests to compare
// with.
//
//   node --experimental-transform-types --no-warnings --import ./register.mjs run.mjs <pele> <limu> [suite…]
//
// scripts/oracle.sh runs it. Each suite is a module in ./suites whose default export is an
// async function given the helpers below; with no suites named, all of them run. The modules
// the suites share are in ./lib.
//
// A suite is named for the fixture it writes: x writes x.json in the package x, and x-y writes
// y.json there, as gantt-render writes diagram/gantt/testdata/pele/render.json. These are
// otherwise:
//
//   parse     diagram/testdata/pele/parse.json, for what every diagram's text goes through
//   common    diagram/internal/common/testdata/pele/common.json
//   draw      diagram/internal/draw/testdata/pele/draw.json
//   jsmath    internal/jsmath/testdata/pele/math.json
//   gantt     dates.json too, beside gantt.json
//
// The fixtures of the trigonometry are the last bits V8 gives on arm64, where clang fuses
// multiplies and adds that it does not fuse on x86-64, and limu's jsmath is ported to match
// them; so the oracle runs only on arm64.
import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { gunzipSync, gzipSync } from 'node:zlib';

const [pele, limu, ...only] = process.argv.slice(2);
if (!pele || !limu) {
  console.error('usage: run.mjs <pele> <limu> [suite…]');
  process.exit(2);
}
if (process.arch !== 'arm64') {
  console.error(
    `run.mjs: this is ${process.arch}; the fixtures hold what V8 on arm64 computes, as its sines and logarithms ` +
      'come out otherwise where clang does not fuse multiply-adds, so they are written only on arm64'
  );
  process.exit(2);
}

const dir = new URL('./suites/', import.meta.url);
const all = readdirSync(dir)
  .filter((f) => f.endsWith('.mjs'))
  .map((f) => f.slice(0, -4))
  .sort();
const unknown = only.filter((name) => !all.includes(name));
if (unknown.length > 0) {
  console.error(`run.mjs: no suite ${unknown.join(', ')}; the suites are ${all.join(' ')}`);
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
  // path + '.gz'; internal/fixture reads it back. A fixture whose JSON is what the file holds
  // already is left as it is.
  write(path, data) {
    const out = join(limu, path + '.gz');
    const json = JSON.stringify({ pele: origin, ...data }) + '\n';
    if (existsSync(out) && gunzipSync(readFileSync(out)).toString('utf8') === json) {
      console.log(`unchanged ${path}.gz`);
      return;
    }
    mkdirSync(dirname(out), { recursive: true });
    writeFileSync(out, gzipSync(json, { level: 9 }));
    console.log(`wrote ${path}.gz`);
  },
  // A random number generator in [0, 1) that gives the same numbers for the same seed.
  random(seed) {
    let s = seed >>> 0;
    return () => (s = (Math.imul(s, 1664525) + 1013904223) >>> 0) / 2 ** 32;
  },
};

for (const name of all.filter((name) => only.length === 0 || only.includes(name))) {
  const suite = await import(new URL(`${name}.mjs`, dir));
  await suite.default(helpers);
}
