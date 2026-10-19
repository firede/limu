// Builds the playground for browsers: limumygo/examples/playground compiled to WebAssembly,
// with Go's loader of it, wasm_exec.js, into public/playground, each named by a hash of its
// content, so that the site can have browsers keep them for good. It writes their names
// and the WebAssembly's size to src/generated/playground.json, which the page that loads
// them, src/pages/playground.astro, reads. Run it from the website directory when the
// playground or limu changes; pnpm build does not, so that a change to the documents does
// not compile it again:
//
//	node scripts/playground.mjs
//
// It fails when the WebAssembly passes 24 MiB: Cloudflare serves an asset of at most 25 MiB.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const maxSize = 24 * 1024 * 1024;
const out = 'public/playground';

const tmp = mkdtempSync(join(tmpdir(), 'limu-playground-'));
try {
  const wasmFile = join(tmp, 'playground.wasm');
  execFileSync('go', ['build', '-trimpath', '-tags', 'mygo_noinspector', '-ldflags=-s -w', '-o', wasmFile, './examples/playground'], {
    cwd: '../limumygo',
    env: { ...process.env, GOOS: 'js', GOARCH: 'wasm' },
    stdio: 'inherit',
  });
  const wasm = readFileSync(wasmFile);
  if (wasm.length > maxSize) {
    throw new Error(`The playground's WebAssembly is ${wasm.length} bytes, past the ${maxSize} the site allows.`);
  }
  const goroot = execFileSync('go', ['env', 'GOROOT'], { encoding: 'utf8' }).trim();
  const loader = readFileSync(join(goroot, 'lib', 'wasm', 'wasm_exec.js'));

  const hash = (bytes) => createHash('sha256').update(bytes).digest('hex').slice(0, 16);
  const names = {
    wasm: `playground.${hash(wasm)}.wasm`,
    loader: `wasm_exec.${hash(loader)}.js`,
  };
  rmSync(out, { recursive: true, force: true });
  mkdirSync(out, { recursive: true });
  writeFileSync(join(out, names.wasm), wasm);
  writeFileSync(join(out, names.loader), loader);
  mkdirSync('src/generated', { recursive: true });
  writeFileSync('src/generated/playground.json', `${JSON.stringify({
    wasm: `/playground/${names.wasm}`,
    loader: `/playground/${names.loader}`,
    size: wasm.length,
  }, null, 2)}\n`);
  console.log(`playground: ${names.wasm}, ${(wasm.length / 1024 / 1024).toFixed(1)} MiB`);
} finally {
  rmSync(tmp, { recursive: true, force: true });
}
